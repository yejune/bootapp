package hosts

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yejune/bootapp/internal/network"
)

const hostsFile = "/etc/hosts"
const marker = "## bootapp"
const legacyMarker = "## docker-bootapp" // For backward compatibility

// AddEntries adds multiple entries to /etc/hosts (requires root)
// Each container can have multiple domains
// Format: comment line followed by host entry (macOS compatible)
//
//	## bootapp:projectname
//	192.168.1.100	example.local
func AddEntries(containers map[string]network.ContainerInfo, projectName string) error {
	// Remove existing entries for this project first
	if err := RemoveProjectEntries(projectName); err != nil {
		// Ignore error if no entries exist
	}

	// Build all entries (comment line + host entry for each domain)
	var entries []string
	commentLine := fmt.Sprintf("%s:%s", marker, projectName)
	for _, info := range containers {
		if info.IP == "" || len(info.Domains) == 0 {
			continue
		}
		for _, domain := range info.Domains {
			if domain != "" {
				// Add comment line first, then host entry
				entries = append(entries, commentLine)
				entries = append(entries, fmt.Sprintf("%s\t%s", info.IP, domain))
				fmt.Printf("  %s -> %s\n", domain, info.IP)
			}
		}
	}

	if len(entries) == 0 {
		return nil
	}

	// Join all entries with newlines
	content := strings.Join(entries, "\n")

	// Append all entries at once (requires root)
	cmd := exec.Command("sudo", "sh", "-c", fmt.Sprintf("echo '%s' >> %s", content, hostsFile))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// AddEntry adds an entry to /etc/hosts (requires root)
// Format: comment line followed by host entry (macOS compatible)
func AddEntry(ip, domain, projectName string) error {
	// Check if entry already exists
	if exists, _ := EntryExists(domain); exists {
		// Remove existing entry first
		if err := RemoveEntry(domain, projectName); err != nil {
			return err
		}
	}

	// Comment line followed by host entry
	commentLine := fmt.Sprintf("%s:%s", marker, projectName)
	hostEntry := fmt.Sprintf("%s\t%s", ip, domain)
	entry := fmt.Sprintf("%s\n%s", commentLine, hostEntry)

	// Append to /etc/hosts (requires root)
	cmd := exec.Command("sudo", "sh", "-c", fmt.Sprintf("echo '%s' >> %s", entry, hostsFile))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RemoveEntry removes entries for a domain or project from /etc/hosts (requires root)
// For new format: removes the comment line preceding the domain entry and the entry itself
func RemoveEntry(domain, projectName string) error {
	if projectName != "" {
		// Remove by project: use RemoveProjectEntries
		return RemoveProjectEntries(projectName)
	}

	// Remove by domain: need to find and remove the domain line and its preceding comment
	// First, remove lines containing the domain (the host entry)
	domainPattern := fmt.Sprintf("/^[^#].*%s/d", domain)
	cmd := exec.Command("sudo", "sed", "-i", "", domainPattern, hostsFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RemoveProjectEntries removes all entries for a project (requires root)
// Also removes legacy docker-bootapp entries for backward compatibility
// New format: removes comment line and the following host entry line
func RemoveProjectEntries(projectName string) error {
	// Remove current marker entries (comment line + next line)
	// sed pattern: when matching marker line, delete it and the next line
	pattern := fmt.Sprintf("/%s:%s/{N;d;}", marker, projectName)
	cmd := exec.Command("sudo", "sed", "-i", "", pattern, hostsFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Also remove legacy marker entries (docker-bootapp -> bootapp migration)
	// Legacy format had marker on same line, so just delete matching lines
	legacyPattern := fmt.Sprintf("/%s:%s/d", legacyMarker, projectName)
	legacyCmd := exec.Command("sudo", "sed", "-i", "", legacyPattern, hostsFile)
	legacyCmd.Stdout = os.Stdout
	legacyCmd.Stderr = os.Stderr
	if err := legacyCmd.Run(); err != nil {
		return err
	}

	// Also remove old format bootapp entries (marker on same line as host)
	oldFormatPattern := fmt.Sprintf("/%s:%s/d", marker, projectName)
	oldFormatCmd := exec.Command("sudo", "sed", "-i", "", oldFormatPattern, hostsFile)
	oldFormatCmd.Stdout = os.Stdout
	oldFormatCmd.Stderr = os.Stderr

	return oldFormatCmd.Run()
}

// EntryExists checks if a domain entry exists
func EntryExists(domain string) (bool, error) {
	file, err := os.Open(hostsFile)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, domain) && strings.Contains(line, marker) {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// ListEntries returns all bootapp-managed entries
// Format: "IP DOMAIN (project)"
func ListEntries() ([]string, error) {
	file, err := os.Open(hostsFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []string
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.Contains(line, marker) && strings.HasPrefix(strings.TrimSpace(line), "##") {
			// Comment line, next line is host entry
			if i+1 < len(lines) {
				hostLine := lines[i+1]
				fields := strings.Fields(hostLine)
				if len(fields) >= 2 {
					project := strings.TrimPrefix(line, marker+":")
					project = strings.TrimSpace(project)
					entries = append(entries, fmt.Sprintf("%s\t%s\t(%s)", fields[0], fields[1], project))
				}
				i++ // Skip next line
			}
		}
	}

	return entries, scanner.Err()
}

// GetIPForDomain returns the IP for a domain from /etc/hosts
func GetIPForDomain(domain string) (string, error) {
	file, err := os.Open(hostsFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, domain) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[0], nil
			}
		}
	}

	return "", fmt.Errorf("domain %s not found in hosts file", domain)
}
