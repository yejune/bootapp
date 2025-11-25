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

// AddEntries adds multiple entries to /etc/hosts (requires root)
// Each container can have multiple domains
func AddEntries(containers map[string]network.ContainerInfo, projectName string) error {
	// Remove existing entries for this project first
	if err := RemoveProjectEntries(projectName); err != nil {
		// Ignore error if no entries exist
	}

	// Build all entries (one entry per domain)
	var entries []string
	for _, info := range containers {
		if info.IP == "" || len(info.Domains) == 0 {
			continue
		}
		for _, domain := range info.Domains {
			if domain != "" {
				entry := fmt.Sprintf("%s\t%s\t%s:%s", info.IP, domain, marker, projectName)
				entries = append(entries, entry)
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
func AddEntry(ip, domain, projectName string) error {
	// Check if entry already exists
	if exists, _ := EntryExists(domain); exists {
		// Remove existing entry first
		if err := RemoveEntry(domain, projectName); err != nil {
			return err
		}
	}

	entry := fmt.Sprintf("%s\t%s\t%s:%s", ip, domain, marker, projectName)

	// Append to /etc/hosts (requires root)
	cmd := exec.Command("sudo", "sh", "-c", fmt.Sprintf("echo '%s' >> %s", entry, hostsFile))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RemoveEntry removes entries for a domain or project from /etc/hosts (requires root)
func RemoveEntry(domain, projectName string) error {
	// Use sed to remove lines matching the domain or project marker
	var pattern string
	if projectName != "" {
		pattern = fmt.Sprintf("/%s:%s/d", marker, projectName)
	} else {
		pattern = fmt.Sprintf("/%s/d", domain)
	}

	cmd := exec.Command("sudo", "sed", "-i", "", pattern, hostsFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RemoveProjectEntries removes all entries for a project (requires root)
func RemoveProjectEntries(projectName string) error {
	pattern := fmt.Sprintf("/%s:%s/d", marker, projectName)

	cmd := exec.Command("sudo", "sed", "-i", "", pattern, hostsFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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
func ListEntries() ([]string, error) {
	file, err := os.Open(hostsFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, marker) {
			entries = append(entries, line)
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
