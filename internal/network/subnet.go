package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	globalConfigDir  = ".docker-bootapp"
	globalConfigFile = "projects.json"
	localConfigDir   = ".docker"
	localConfigFile  = "network.json"
	// Subnet range: 172.18.0.0/16 to 172.31.0.0/16
	subnetStart = 18
	subnetEnd   = 31
)

// ContainerInfo stores individual container's domains and IP
type ContainerInfo struct {
	Domains []string `json:"domains,omitempty"`
	IP      string   `json:"ip"`
}

// ProjectInfo stores global project information
type ProjectInfo struct {
	Path   string `json:"path"`
	Subnet string `json:"subnet"`
	Domain string `json:"domain"`
}

// NetworkConfig stores local project network configuration
type NetworkConfig struct {
	Project    string                   `json:"project"`
	Subnet     string                   `json:"subnet"`
	Containers map[string]ContainerInfo `json:"containers"`
}

// ProjectManager manages project configurations
type ProjectManager struct {
	globalPath string
	projects   map[string]ProjectInfo
}

// NewProjectManager creates a new project manager
func NewProjectManager() (*ProjectManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	globalPath := filepath.Join(homeDir, globalConfigDir, globalConfigFile)

	mgr := &ProjectManager{
		globalPath: globalPath,
		projects:   make(map[string]ProjectInfo),
	}

	// Load existing configuration
	if err := mgr.loadGlobal(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return mgr, nil
}

// GetOrCreateProject returns existing project or creates a new one
func (m *ProjectManager) GetOrCreateProject(projectName, projectPath, domain string) (*NetworkConfig, error) {
	// Check if project exists
	if info, ok := m.projects[projectName]; ok {
		// Load local config
		localConfig, err := m.loadLocalConfig(info.Path)
		if err == nil {
			return localConfig, nil
		}
		// Local config missing, recreate it
	}

	// Allocate new subnet
	subnet, err := m.allocateSubnet()
	if err != nil {
		return nil, err
	}

	// Create project info
	info := ProjectInfo{
		Path:   projectPath,
		Subnet: subnet,
		Domain: domain,
	}
	m.projects[projectName] = info

	// Save global config
	if err := m.saveGlobal(); err != nil {
		return nil, err
	}

	// Create local config
	localConfig := &NetworkConfig{
		Project:    projectName,
		Subnet:     subnet,
		Containers: make(map[string]ContainerInfo),
	}

	// Save local config
	if err := m.saveLocalConfig(projectPath, localConfig); err != nil {
		return nil, err
	}

	return localConfig, nil
}

// UpdateContainers updates container info in local config
func (m *ProjectManager) UpdateContainers(projectPath string, containers map[string]ContainerInfo) error {
	localConfig, err := m.loadLocalConfig(projectPath)
	if err != nil {
		return err
	}

	localConfig.Containers = containers
	return m.saveLocalConfig(projectPath, localConfig)
}

// GetProject returns project info
func (m *ProjectManager) GetProject(projectName string) (ProjectInfo, bool) {
	info, ok := m.projects[projectName]
	return info, ok
}

// ListProjects returns all registered projects
func (m *ProjectManager) ListProjects() map[string]ProjectInfo {
	result := make(map[string]ProjectInfo)
	for k, v := range m.projects {
		result[k] = v
	}
	return result
}

// RemoveProject removes a project
func (m *ProjectManager) RemoveProject(projectName string) error {
	if info, ok := m.projects[projectName]; ok {
		// Remove local config
		localPath := filepath.Join(info.Path, localConfigDir, localConfigFile)
		os.Remove(localPath)
	}

	delete(m.projects, projectName)
	return m.saveGlobal()
}

// GetLocalConfig loads local network config for a project path
func (m *ProjectManager) GetLocalConfig(projectPath string) (*NetworkConfig, error) {
	return m.loadLocalConfig(projectPath)
}

func (m *ProjectManager) allocateSubnet() (string, error) {
	usedNumbers := make(map[int]bool)
	for _, info := range m.projects {
		num := extractSubnetNumber(info.Subnet)
		if num > 0 {
			usedNumbers[num] = true
		}
	}

	for i := subnetStart; i <= subnetEnd; i++ {
		if !usedNumbers[i] {
			return fmt.Sprintf("172.%d.0.0/16", i), nil
		}
	}

	return "", fmt.Errorf("no available subnets (all %d-%d in use)", subnetStart, subnetEnd)
}

func (m *ProjectManager) loadGlobal() error {
	data, err := os.ReadFile(m.globalPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.projects)
}

func (m *ProjectManager) saveGlobal() error {
	dir := filepath.Dir(m.globalPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.globalPath, data, 0644)
}

func (m *ProjectManager) loadLocalConfig(projectPath string) (*NetworkConfig, error) {
	localPath := filepath.Join(projectPath, localConfigDir, localConfigFile)
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}

	var config NetworkConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (m *ProjectManager) saveLocalConfig(projectPath string, config *NetworkConfig) error {
	localDir := filepath.Join(projectPath, localConfigDir)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return err
	}

	localPath := filepath.Join(localDir, localConfigFile)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(localPath, data, 0644)
}

func extractSubnetNumber(subnet string) int {
	parts := strings.Split(subnet, ".")
	if len(parts) >= 2 {
		var num int
		fmt.Sscanf(parts[1], "%d", &num)
		return num
	}
	return 0
}

// GetDefaultIP returns the default app IP for a subnet (x.x.0.2)
func GetDefaultIP(subnet string) string {
	parts := strings.Split(subnet, "/")
	if len(parts) > 0 {
		ipParts := strings.Split(parts[0], ".")
		if len(ipParts) == 4 {
			return fmt.Sprintf("%s.%s.0.2", ipParts[0], ipParts[1])
		}
	}
	return ""
}

// GetContainerIP returns IP for a specific container index
func GetContainerIP(subnet string, index int) string {
	parts := strings.Split(subnet, "/")
	if len(parts) > 0 {
		ipParts := strings.Split(parts[0], ".")
		if len(ipParts) == 4 {
			return fmt.Sprintf("%s.%s.0.%d", ipParts[0], ipParts[1], index+2)
		}
	}
	return ""
}
