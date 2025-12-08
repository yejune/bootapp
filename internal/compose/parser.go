package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents a docker-compose.yml structure
type ComposeFile struct {
	Name     string                 `yaml:"name"`
	Version  string                 `yaml:"version"`
	Services map[string]Service     `yaml:"services"`
	Networks map[string]Network     `yaml:"networks"`
	X        map[string]interface{} `yaml:",inline"`
}

// Service represents a docker-compose service
type Service struct {
	Image       string            `yaml:"image"`
	Build       interface{}       `yaml:"build"`
	Environment interface{}       `yaml:"environment"`
	Labels      interface{}       `yaml:"labels"`
	Networks    interface{}       `yaml:"networks"`
	Ports       []string          `yaml:"ports"`
	DependsOn   interface{}       `yaml:"depends_on"`
}

// Network represents a docker-compose network
type Network struct {
	Driver string     `yaml:"driver"`
	IPAM   *IPAMConfig `yaml:"ipam"`
}

// IPAMConfig represents IPAM configuration
type IPAMConfig struct {
	Config []IPAMPoolConfig `yaml:"config"`
}

// IPAMPoolConfig represents a single IPAM pool
type IPAMPoolConfig struct {
	Subnet string `yaml:"subnet"`
}

// FindComposeFile finds the docker-compose file in the current directory
// Returns error if no file found, or multiple files found (use FindComposeFiles for selection)
func FindComposeFile() (string, error) {
	files, err := FindComposeFiles()
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		cwd, _ := os.Getwd()
		return "", fmt.Errorf("no docker-compose file found in %s", cwd)
	}

	if len(files) == 1 {
		return files[0], nil
	}

	return "", &MultipleFilesError{Files: files}
}

// MultipleFilesError is returned when multiple compose files are found
type MultipleFilesError struct {
	Files []string
}

func (e *MultipleFilesError) Error() string {
	return fmt.Sprintf("multiple compose files found: %v", e.Files)
}

// FindComposeFiles returns all compose files in the current directory
func FindComposeFiles() ([]string, error) {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"docker-compose.*.yml",
		"docker-compose.*.yaml",
		"compose.yml",
		"compose.yaml",
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	var found []string
	seen := make(map[string]bool)

	for _, pattern := range candidates {
		matches, err := filepath.Glob(filepath.Join(cwd, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			if !seen[match] {
				seen[match] = true
				found = append(found, match)
			}
		}
	}

	return found, nil
}

// ParseComposeFile parses a docker-compose file
func ParseComposeFile(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, err
	}

	return &compose, nil
}

// GetProjectName derives project name from compose file or directory
func GetProjectName(composePath string, compose *ComposeFile) string {
	// Use name from compose file if specified
	if compose != nil && compose.Name != "" {
		return compose.Name
	}
	// Fallback to directory name (lowercased for Docker compatibility)
	dir := filepath.Dir(composePath)
	return strings.ToLower(filepath.Base(dir))
}

// ValidateForBootapp checks if compose file is compatible with bootapp
// Returns error if custom networks or IP assignments are found
func ValidateForBootapp(compose *ComposeFile) error {
	// Check for top-level networks definition
	if len(compose.Networks) > 0 {
		return fmt.Errorf("custom 'networks:' section detected")
	}

	// Check for service-level network config with IP
	for serviceName, service := range compose.Services {
		if service.Networks != nil {
			// Check if it's a map with ipv4_address
			if netMap, ok := service.Networks.(map[string]interface{}); ok {
				for netName, netConfig := range netMap {
					if configMap, ok := netConfig.(map[string]interface{}); ok {
						if _, hasIP := configMap["ipv4_address"]; hasIP {
							return fmt.Errorf("service '%s' has static IP in network '%s'", serviceName, netName)
						}
						if _, hasIP := configMap["ipv6_address"]; hasIP {
							return fmt.Errorf("service '%s' has static IP in network '%s'", serviceName, netName)
						}
					}
				}
			}
		}
	}

	return nil
}

// ExtractDomain extracts the first domain from compose file (for backward compatibility)
func ExtractDomain(compose *ComposeFile) string {
	domains := ExtractDomains(compose)
	if len(domains) > 0 {
		return domains[0]
	}
	return ""
}

// ExtractDomains extracts all domains from compose file environment or labels
// Supports: DOMAIN, DOMAINS, SSL_DOMAINS, APP_DOMAIN, VIRTUAL_HOST
// All can be comma-separated
func ExtractDomains(compose *ComposeFile) []string {
	var allDomains []string

	for _, service := range compose.Services {
		// Check environment variables
		domains := extractDomainsFromEnvironment(service.Environment)
		allDomains = append(allDomains, domains...)

		// Check labels (for Traefik, etc.)
		domains = extractDomainsFromLabels(service.Labels)
		allDomains = append(allDomains, domains...)
	}

	// Remove duplicates
	return uniqueDomains(allDomains)
}

// ExtractServiceDomains extracts domains per service
// Returns map[serviceName][]domains
// Only services with DOMAIN/DOMAINS/SSL_DOMAINS/APP_DOMAIN/VIRTUAL_HOST will have entries
func ExtractServiceDomains(compose *ComposeFile) map[string][]string {
	result := make(map[string][]string)

	for serviceName, service := range compose.Services {
		var domains []string

		// Check environment variables
		domains = append(domains, extractDomainsFromEnvironment(service.Environment)...)

		// Check labels (for Traefik, etc.)
		domains = append(domains, extractDomainsFromLabels(service.Labels)...)

		if len(domains) > 0 {
			result[serviceName] = uniqueDomains(domains)
		}
	}

	return result
}

func extractDomainsFromEnvironment(env interface{}) []string {
	var domains []string

	// Environment variable keys to check (in priority order)
	envKeys := []string{"DOMAIN", "DOMAINS", "SSL_DOMAINS", "APP_DOMAIN", "VIRTUAL_HOST"}

	switch e := env.(type) {
	case []interface{}:
		for _, item := range e {
			if str, ok := item.(string); ok {
				for _, key := range envKeys {
					prefix := key + "="
					if strings.HasPrefix(str, prefix) {
						value := strings.TrimPrefix(str, prefix)
						domains = append(domains, splitDomains(value)...)
					}
				}
			}
		}
	case map[string]interface{}:
		for _, key := range envKeys {
			if value, ok := e[key].(string); ok {
				domains = append(domains, splitDomains(value)...)
			}
		}
	}

	return domains
}

// splitDomains splits domain string by comma, newline, or space
func splitDomains(value string) []string {
	var domains []string
	// Replace newlines and commas with space, then split by whitespace
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, ",", " ")
	for _, d := range strings.Fields(value) {
		d = strings.TrimSpace(d)
		if d != "" {
			domains = append(domains, d)
		}
	}
	return domains
}

// uniqueDomains removes duplicate domains while preserving order
func uniqueDomains(domains []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, d := range domains {
		if !seen[d] {
			seen[d] = true
			result = append(result, d)
		}
	}
	return result
}

// ExtractSSLDomains extracts only SSL_DOMAINS from compose file
// Returns all unique SSL domains that need certificates
func ExtractSSLDomains(compose *ComposeFile) []string {
	var allDomains []string

	for _, service := range compose.Services {
		domains := extractSSLDomainsFromEnvironment(service.Environment)
		allDomains = append(allDomains, domains...)
	}

	return uniqueDomains(allDomains)
}

// extractSSLDomainsFromEnvironment extracts only SSL_DOMAINS from environment
func extractSSLDomainsFromEnvironment(env interface{}) []string {
	var domains []string

	switch e := env.(type) {
	case []interface{}:
		for _, item := range e {
			if str, ok := item.(string); ok {
				prefix := "SSL_DOMAINS="
				if strings.HasPrefix(str, prefix) {
					value := strings.TrimPrefix(str, prefix)
					domains = append(domains, splitDomains(value)...)
				}
			}
		}
	case map[string]interface{}:
		if value, ok := e["SSL_DOMAINS"].(string); ok {
			domains = append(domains, splitDomains(value)...)
		}
	}

	return domains
}

func extractDomainsFromLabels(labels interface{}) []string {
	var domains []string

	switch l := labels.(type) {
	case []interface{}:
		for _, item := range l {
			if str, ok := item.(string); ok {
				// Traefik rule
				if strings.Contains(str, "traefik.http.routers") && strings.Contains(str, "Host(") {
					domains = append(domains, extractTraefikHosts(str)...)
				}
			}
		}
	case map[string]interface{}:
		for key, value := range l {
			if strings.Contains(key, "traefik.http.routers") && strings.Contains(key, "rule") {
				if str, ok := value.(string); ok && strings.Contains(str, "Host(") {
					domains = append(domains, extractTraefikHosts(str)...)
				}
			}
		}
	}

	return domains
}

// extractTraefikHosts extracts hosts from Traefik Host() rule
// Supports: Host(`a.com`) || Host(`b.com`) and Host(`a.com`, `b.com`)
func extractTraefikHosts(rule string) []string {
	var hosts []string
	remaining := rule

	for {
		// Find Host( - supports both Host(` and Host(
		start := strings.Index(remaining, "Host(")
		if start == -1 {
			break
		}
		start += 5 // Move past "Host("

		// Find the closing )
		depth := 1
		end := start
		for i := start; i < len(remaining) && depth > 0; i++ {
			if remaining[i] == '(' {
				depth++
			} else if remaining[i] == ')' {
				depth--
			}
			if depth == 0 {
				end = i
			}
		}

		if end <= start {
			break
		}

		// Extract content inside Host()
		content := remaining[start:end]

		// Parse backtick-quoted domains: `domain1`, `domain2`
		for {
			tickStart := strings.Index(content, "`")
			if tickStart == -1 {
				break
			}
			tickEnd := strings.Index(content[tickStart+1:], "`")
			if tickEnd == -1 {
				break
			}
			host := content[tickStart+1 : tickStart+1+tickEnd]
			if host != "" {
				hosts = append(hosts, host)
			}
			content = content[tickStart+1+tickEnd+1:]
		}

		remaining = remaining[end:]
	}

	return hosts
}
