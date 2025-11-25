package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	globalConfigDir  = ".bootapp"
	globalConfigFile = "projects.json"
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
	Path       string   `json:"path"`
	Subnet     string   `json:"subnet"`
	Domains    []string `json:"domains,omitempty"`
	SSLDomains []string `json:"ssl_domains,omitempty"`

	// Deprecated: use Domains instead (kept for backward compatibility)
	Domain string `json:"domain,omitempty"`
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

// ProjectChanges captures what changed between previous and current config
type ProjectChanges struct {
	PreviousDomain     string
	PreviousSSLDomains []string
	DomainChanged      bool
	RemovedSSLDomains  []string
}

// GetOrCreateProject returns existing project or creates a new one
// Also returns changes detected from previous config
func (m *ProjectManager) GetOrCreateProject(projectName, projectPath string, domains []string, sslDomains []string) (*ProjectInfo, *ProjectChanges, error) {
	changes := &ProjectChanges{}

	// Check if project exists
	if info, ok := m.projects[projectName]; ok {
		// Capture previous values for change detection
		// Support old Domain field for backward compatibility
		prevDomains := info.Domains
		if len(prevDomains) == 0 && info.Domain != "" {
			prevDomains = []string{info.Domain}
		}
		if len(prevDomains) > 0 {
			changes.PreviousDomain = prevDomains[0] // For legacy change detection
		}
		changes.PreviousSSLDomains = info.SSLDomains

		// Check domain change
		if !slicesEqual(prevDomains, domains) {
			changes.DomainChanged = true
		}

		// Find removed SSL domains
		changes.RemovedSSLDomains = findRemovedDomains(info.SSLDomains, sslDomains)

		// Update all fields
		needSave := false
		if info.Path != projectPath {
			info.Path = projectPath
			needSave = true
		}
		if !slicesEqual(info.Domains, domains) {
			info.Domains = domains
			info.Domain = "" // Clear deprecated field
			needSave = true
		}
		if !slicesEqual(info.SSLDomains, sslDomains) {
			info.SSLDomains = sslDomains
			needSave = true
		}
		if needSave {
			m.projects[projectName] = info
			m.saveGlobal()
		}
		return &info, changes, nil
	}

	// Allocate new subnet
	subnet, err := m.allocateSubnet()
	if err != nil {
		return nil, nil, err
	}

	// Create project info
	info := ProjectInfo{
		Path:       projectPath,
		Subnet:     subnet,
		Domains:    domains,
		SSLDomains: sslDomains,
	}
	m.projects[projectName] = info

	// Save global config
	if err := m.saveGlobal(); err != nil {
		return nil, nil, err
	}

	return &info, changes, nil
}

// findRemovedDomains returns domains that were in old but not in new
func findRemovedDomains(old, new []string) []string {
	newSet := make(map[string]bool)
	for _, d := range new {
		newSet[d] = true
	}

	var removed []string
	for _, d := range old {
		if !newSet[d] {
			removed = append(removed, d)
		}
	}
	return removed
}

// slicesEqual checks if two string slices are equal
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Compare as sets (order independent)
	aMap := make(map[string]bool)
	for _, v := range a {
		aMap[v] = true
	}
	for _, v := range b {
		if !aMap[v] {
			return false
		}
	}
	return true
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
	delete(m.projects, projectName)
	return m.saveGlobal()
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
