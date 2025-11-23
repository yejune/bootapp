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
func FindComposeFile() (string, error) {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for _, name := range candidates {
		path := filepath.Join(cwd, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no docker-compose file found in %s", cwd)
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

// GetProjectName derives project name from compose file path
func GetProjectName(composePath string) string {
	dir := filepath.Dir(composePath)
	return filepath.Base(dir)
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
// Supports: Host(`a.com`) || Host(`b.com`) or Host(`a.com`, `b.com`)
func extractTraefikHosts(rule string) []string {
	var hosts []string
	remaining := rule

	for {
		start := strings.Index(remaining, "Host(`")
		if start == -1 {
			break
		}
		start += 6
		end := strings.Index(remaining[start:], "`")
		if end == -1 {
			break
		}
		hosts = append(hosts, remaining[start:start+end])
		remaining = remaining[start+end:]
	}

	return hosts
}
