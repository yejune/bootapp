package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/yejune/docker-bootapp/internal/cert"
	"github.com/yejune/docker-bootapp/internal/compose"
	"github.com/yejune/docker-bootapp/internal/hosts"
	"github.com/yejune/docker-bootapp/internal/network"
	"github.com/yejune/docker-bootapp/internal/route"
)

var (
	noBuild       bool
	pull          bool
	detach        bool
	forceRecreate bool
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Create and start containers with network setup",
	Long: `Start containers using docker-compose and automatically:
- Allocate unique subnet for the project
- Create Docker network with the subnet
- Register all container domains in /etc/hosts
- Setup routing (macOS only)
- Save configuration to .docker/network.json`,
	RunE: runUp,
}

func init() {
	upCmd.Flags().BoolVar(&noBuild, "no-build", false, "Don't build images")
	upCmd.Flags().BoolVar(&pull, "pull", false, "Pull images before starting")
	upCmd.Flags().BoolVarP(&detach, "detach", "d", true, "Run containers in background")
	upCmd.Flags().BoolVarP(&forceRecreate, "force-recreate", "F", false, "Force recreate containers")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	// Validate sudo credentials upfront (required for /etc/hosts and cert trust)
	if err := ValidateSudo(); err != nil {
		return fmt.Errorf("sudo authentication failed: %w", err)
	}

	// Find or use specified docker-compose file
	var composePath string
	var err error

	if composeFile != "" {
		// Use specified file
		composePath, err = filepath.Abs(composeFile)
		if err != nil {
			return fmt.Errorf("invalid compose file path: %w", err)
		}
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			return fmt.Errorf("compose file not found: %s", composePath)
		}
	} else {
		// Auto-detect
		composePath, err = compose.FindComposeFile()
		if err != nil {
			// Check if multiple files found
			if multiErr, ok := err.(*compose.MultipleFilesError); ok {
				composePath, err = selectComposeFile(multiErr.Files)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	fmt.Printf("Using compose file: %s\n", composePath)

	// Parse compose file
	composeData, err := compose.ParseComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Validate compose file for bootapp compatibility
	if err := compose.ValidateForBootapp(composeData); err != nil {
		return fmt.Errorf("%s\n\n"+
			"docker-bootapp manages networks automatically and is intended for local development.\n"+
			"Please remove network configurations from your compose file, or use 'docker compose' directly.", err)
	}

	// Get project info
	projectPath := filepath.Dir(composePath)
	projectName := compose.GetProjectName(composePath)
	fmt.Printf("Project: %s\n", projectName)

	// Extract domains per service from compose file
	serviceDomains := compose.ExtractServiceDomains(composeData)
	if len(serviceDomains) > 0 {
		fmt.Println("Service Domains:")
		for svc, doms := range serviceDomains {
			fmt.Printf("  %s: %s\n", svc, strings.Join(doms, ", "))
		}
	}

	// Generate SSL certificates for SSL_DOMAINS only
	certDir := filepath.Join(projectPath, "var", "certs")
	var certsToTrust []string
	certsGenerated := false
	sslDomains := compose.ExtractSSLDomains(composeData)
	if len(sslDomains) > 0 {
		fmt.Println("\nSetting up SSL certificates...")

		// If force-recreate, delete existing certs and remove trust first
		if forceRecreate {
			fmt.Println("Force recreate: removing existing certificates...")
			for _, domain := range sslDomains {
				if cert.CertExists(domain, certDir) {
					// Remove from trust store first
					if err := cert.UninstallFromTrustStore(domain); err != nil {
						fmt.Printf("  ⚠️  %s: failed to untrust\n", domain)
					} else {
						fmt.Printf("  ✓ %s: untrusted\n", domain)
					}
					// Delete local cert files
					if err := cert.RemoveCert(domain, certDir); err != nil {
						fmt.Printf("  ⚠️  %s: failed to remove\n", domain)
					} else {
						fmt.Printf("  ✓ %s: removed\n", domain)
					}
				}
			}
		}

		info := cert.DefaultCertInfo()
		for _, domain := range sslDomains {
			// Generate if not exists
			if !cert.CertExists(domain, certDir) {
				if err := cert.GenerateCert(domain, certDir, info); err != nil {
					fmt.Printf("  ⚠️  %s: failed to generate\n", domain)
					continue
				}
				fmt.Printf("  ✓ %s: generated\n", domain)
				certsGenerated = true
			}
			// Collect certs that need trust
			if !cert.IsTrusted(domain) {
				certsToTrust = append(certsToTrust, domain)
			} else {
				fmt.Printf("  ✓ %s: trusted\n", domain)
			}
		}
	}

	// Initialize project manager
	projectMgr, err := network.NewProjectManager()
	if err != nil {
		return fmt.Errorf("failed to initialize project manager: %w", err)
	}

	// Collect all domains from serviceDomains
	allDomains := collectAllDomains(serviceDomains)
	if len(allDomains) == 0 {
		// No domains in compose file, use project name as default
		allDomains = []string{projectName + ".local"}
	}

	// Get or create project configuration (allocates unique subnet)
	projectInfo, changes, err := projectMgr.GetOrCreateProject(projectName, projectPath, allDomains, sslDomains)
	if err != nil {
		return fmt.Errorf("failed to setup project: %w", err)
	}
	fmt.Printf("Subnet: %s\n", projectInfo.Subnet)

	// Clean up removed SSL domains (certs + trust)
	if len(changes.RemovedSSLDomains) > 0 {
		fmt.Println("\nCleaning up removed SSL domains...")
		for _, domain := range changes.RemovedSSLDomains {
			// Remove from trust store
			if err := cert.UninstallFromTrustStore(domain); err != nil {
				fmt.Printf("  ⚠️  %s: failed to untrust\n", domain)
			} else {
				fmt.Printf("  ✓ %s: untrusted\n", domain)
			}
			// Delete local cert files
			if cert.CertExists(domain, certDir) {
				if err := cert.RemoveCert(domain, certDir); err != nil {
					fmt.Printf("  ⚠️  %s: failed to remove cert\n", domain)
				} else {
					fmt.Printf("  ✓ %s: cert removed\n", domain)
				}
			}
		}
	}

	// Clean up old hosts entries if domain changed
	if changes.DomainChanged && changes.PreviousDomain != "" {
		newDomain := allDomains[0]
		if len(allDomains) > 0 {
			newDomain = strings.Join(allDomains, ", ")
		}
		fmt.Printf("\nDomain changed: %s → %s\n", changes.PreviousDomain, newDomain)
		fmt.Println("Cleaning up old /etc/hosts entries...")
		if err := hosts.RemoveProjectEntries(projectName); err != nil {
			fmt.Printf("  ⚠️  Failed to remove old hosts entries: %v\n", err)
		} else {
			fmt.Println("  ✓ Old hosts entries removed")
		}
	}

	// Install certificates to trust store
	if len(certsToTrust) > 0 {
		fmt.Println("\nInstalling certificates to system trust store...")
		for _, domain := range certsToTrust {
			if err := cert.InstallToTrustStore(domain, certDir); err != nil {
				fmt.Printf("  ⚠️  %s: failed to trust\n", domain)
			}
		}
	}

	// Run docker-compose up (force recreate if certs were newly generated or --force-recreate)
	fmt.Println("\nStarting containers...")
	if err := runDockerCompose(composePath, projectName, forceRecreate || certsGenerated); err != nil {
		return err
	}

	// Get container IPs and network info from default compose network
	fmt.Println("\nDiscovering containers...")
	containerIPs, networkSubnet, err := getContainerIPsAndSubnet(projectName)
	if err != nil {
		fmt.Printf("Warning: Could not get container IPs: %v\n", err)
		containerIPs = make(map[string]string)
	}
	if networkSubnet != "" {
		fmt.Printf("Network subnet: %s\n", networkSubnet)
	}

	// Build container info with domains (only for services with domain config)
	containers := buildContainerInfo(containerIPs, serviceDomains)

	// Print container info
	fmt.Println("\nContainers:")
	for name, info := range containers {
		if len(info.Domains) > 0 {
			fmt.Printf("  %s: %s -> %s\n", name, strings.Join(info.Domains, ", "), info.IP)
		} else {
			fmt.Printf("  %s: %s (no domain)\n", name, info.IP)
		}
	}

	// Setup /etc/hosts for all containers
	fmt.Println("\nSetting up /etc/hosts...")
	if err := hosts.AddEntries(containers, projectName); err != nil {
		return fmt.Errorf("failed to update /etc/hosts: %w", err)
	}

	// Setup routing (macOS only) - use subnet from global config
	fmt.Println("\nSetting up routing...")
	// Get first container IP for connectivity test
	var testIP string
	for _, info := range containers {
		if info.IP != "" {
			testIP = info.IP
			break
		}
	}
	if err := route.SetupRouteWithTest(projectInfo.Subnet, testIP); err != nil {
		fmt.Printf("Warning: Route setup failed: %v\n", err)
	}

	// Print config file location
	fmt.Println("\n📁 Configuration: ~/.docker-bootapp/projects.json")

	// Get main app domain
	var appDomain string
	if info, ok := containers["app"]; ok && len(info.Domains) > 0 {
		appDomain = info.Domains[0]
	} else if len(allDomains) > 0 {
		appDomain = allDomains[0]
	}

	if appDomain != "" {
		fmt.Printf("\n✅ Ready! Access your app at: https://%s\n", appDomain)
	} else {
		fmt.Printf("\n✅ Ready!\n")
	}

	return nil
}

// buildContainerInfo creates ContainerInfo map with domains
// serviceDomains: map[serviceName][]domains from compose file
// Only services with explicit domain config get domains
// Other services get IP only (no domain)
func buildContainerInfo(ips map[string]string, serviceDomains map[string][]string) map[string]network.ContainerInfo {
	containers := make(map[string]network.ContainerInfo)

	for name, ip := range ips {
		// Check if this service has domain configuration
		if domains, ok := serviceDomains[name]; ok && len(domains) > 0 {
			containers[name] = network.ContainerInfo{
				Domains: domains,
				IP:      ip,
			}
		} else {
			// No domain config - IP only
			containers[name] = network.ContainerInfo{
				IP: ip,
			}
		}
	}

	return containers
}

func createDockerNetwork(name, subnet string) error {
	// Check if network already exists
	checkCmd := exec.Command("docker", "network", "inspect", name)
	if checkCmd.Run() == nil {
		// Network exists, check if subnet matches
		return nil // Keep existing network
	}

	// Create network with subnet
	args := []string{
		"network", "create",
		"--driver", "bridge",
		"--subnet", subnet,
		name,
	}
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runDockerCompose(composePath, projectName string, forceRecreate bool) error {
	// Use "docker compose" (V2) instead of "docker-compose"
	args := []string{"compose", "-f", composePath, "-p", projectName, "up"}

	if detach {
		args = append(args, "-d")
	}
	if !noBuild {
		args = append(args, "--build")
	}
	if pull {
		args = append(args, "--pull", "always")
	}
	if forceRecreate {
		args = append(args, "--force-recreate")
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = filepath.Dir(composePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// connectContainersToNetwork connects all project containers to the bootapp network
func connectContainersToNetwork(projectName, networkName string) error {
	// List containers for this project
	listCmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))
	output, err := listCmd.Output()
	if err != nil {
		return err
	}

	containerIDs := strings.Fields(string(output))
	if len(containerIDs) == 0 {
		return fmt.Errorf("no containers found for project %s", projectName)
	}

	for _, id := range containerIDs {
		// Connect container to bootapp network (ignore if already connected)
		connectCmd := exec.Command("docker", "network", "connect", networkName, id)
		if err := connectCmd.Run(); err != nil {
			// Ignore "already connected" errors
			continue
		}
	}

	return nil
}

// getContainerIPsFromNetwork gets container IPs specifically from the bootapp network
func getContainerIPsFromNetwork(projectName, networkName string) (map[string]string, error) {
	// List containers for this project
	listCmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))
	output, err := listCmd.Output()
	if err != nil {
		return nil, err
	}

	containerIDs := strings.Fields(string(output))
	if len(containerIDs) == 0 {
		return nil, fmt.Errorf("no containers found for project %s", projectName)
	}

	containers := make(map[string]string)

	for _, id := range containerIDs {
		// Get container info as JSON
		inspectCmd := exec.Command("docker", "inspect", id)
		output, err := inspectCmd.Output()
		if err != nil {
			continue
		}

		var infos []DockerContainerInfo
		if err := json.Unmarshal(output, &infos); err != nil || len(infos) == 0 {
			continue
		}

		info := infos[0]
		serviceName := info.Config.Labels["com.docker.compose.service"]

		// Get IP from the specific bootapp network
		if netInfo, ok := info.NetworkSettings.Networks[networkName]; ok && netInfo.IPAddress != "" {
			containers[serviceName] = netInfo.IPAddress
		}
	}

	return containers, nil
}

// getContainerIPsAndSubnet returns container IPs and the network subnet
func getContainerIPsAndSubnet(projectName string) (map[string]string, string, error) {
	// List containers for this project
	listCmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))
	output, err := listCmd.Output()
	if err != nil {
		return nil, "", err
	}

	containerIDs := strings.Fields(string(output))
	if len(containerIDs) == 0 {
		return nil, "", fmt.Errorf("no containers found for project %s", projectName)
	}

	containers := make(map[string]string)
	var networkName string

	for _, id := range containerIDs {
		inspectCmd := exec.Command("docker", "inspect", id)
		output, err := inspectCmd.Output()
		if err != nil {
			continue
		}

		var infos []DockerContainerInfo
		if err := json.Unmarshal(output, &infos); err != nil || len(infos) == 0 {
			continue
		}

		info := infos[0]
		serviceName := info.Config.Labels["com.docker.compose.service"]

		// Prefer *_default network
		for netName, net := range info.NetworkSettings.Networks {
			if net.IPAddress != "" && serviceName != "" {
				if strings.HasSuffix(netName, "_default") {
					containers[serviceName] = net.IPAddress
					networkName = netName
					break
				}
				if containers[serviceName] == "" {
					containers[serviceName] = net.IPAddress
					networkName = netName
				}
			}
		}
	}

	// Get subnet from the network
	subnet := getNetworkSubnet(networkName)

	return containers, subnet, nil
}

// getNetworkSubnet gets the subnet CIDR from a Docker network
func getNetworkSubnet(networkName string) string {
	if networkName == "" {
		return ""
	}

	cmd := exec.Command("docker", "network", "inspect", networkName, "--format", "{{range .IPAM.Config}}{{.Subnet}}{{end}}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// DockerContainerInfo for JSON parsing
type DockerContainerInfo struct {
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

func getContainerIPsJSON(projectName string) (map[string]string, error) {
	// Alternative method using JSON output
	listCmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))
	output, err := listCmd.Output()
	if err != nil {
		return nil, err
	}

	containerIDs := strings.Fields(string(output))
	containers := make(map[string]string)

	for _, id := range containerIDs {
		inspectCmd := exec.Command("docker", "inspect", id)
		output, err := inspectCmd.Output()
		if err != nil {
			continue
		}

		var infos []DockerContainerInfo
		if err := json.Unmarshal(output, &infos); err != nil || len(infos) == 0 {
			continue
		}

		info := infos[0]
		serviceName := info.Config.Labels["com.docker.compose.service"]
		for _, net := range info.NetworkSettings.Networks {
			if net.IPAddress != "" && serviceName != "" {
				containers[serviceName] = net.IPAddress
				break
			}
		}
	}

	return containers, nil
}

// selectComposeFile prompts user to select from multiple compose files
func selectComposeFile(files []string) (string, error) {
	// Create display items (just filenames for cleaner display)
	items := make([]string, len(files))
	for i, f := range files {
		items[i] = filepath.Base(f)
	}

	// Create interactive prompt
	prompt := promptui.Select{
		Label: "Select compose file",
		Items: items,
		Size:  len(items),
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}:",
			Active:   "▸ {{ . | cyan }}",
			Inactive: "  {{ . }}",
			Selected: "✓ {{ . | green }}",
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("selection cancelled: %w", err)
	}

	return files[index], nil
}

// collectAllDomains collects all unique domains from serviceDomains map
func collectAllDomains(serviceDomains map[string][]string) []string {
	domainSet := make(map[string]bool)
	for _, domains := range serviceDomains {
		for _, d := range domains {
			domainSet[d] = true
		}
	}

	var result []string
	for d := range domainSet {
		result = append(result, d)
	}
	return result
}
