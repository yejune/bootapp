package cert

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// InstallToTrustStore adds certificate to system trust store
func InstallToTrustStore(domain, certDir string) error {
	certPath := filepath.Join(certDir, domain+".crt")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return fmt.Errorf("certificate not found: %s", certPath)
	}

	switch runtime.GOOS {
	case "darwin":
		return installDarwin(certPath, domain)
	case "linux":
		return installLinux(certPath, domain)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// UninstallFromTrustStore removes certificate from system trust store
func UninstallFromTrustStore(domain string) error {
	switch runtime.GOOS {
	case "darwin":
		return uninstallDarwin(domain)
	case "linux":
		return uninstallLinux(domain)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// IsTrusted checks if certificate is in system trust store
func IsTrusted(domain string) bool {
	switch runtime.GOOS {
	case "darwin":
		return isTrustedDarwin(domain)
	case "linux":
		return isTrustedLinux(domain)
	default:
		return false
	}
}

// macOS implementation (requires root)
func installDarwin(certPath, domain string) error {
	// Remove existing certificate first
	uninstallDarwin(domain)

	// Copy cert to temp file (avoids path issues)
	tmpCertFile := fmt.Sprintf("/tmp/bootapp-cert-%s.crt", domain)
	cpCmd := exec.Command("cp", certPath, tmpCertFile)
	if err := cpCmd.Run(); err != nil {
		return fmt.Errorf("failed to copy certificate: %w", err)
	}
	defer os.Remove(tmpCertFile)

	// Add to system keychain as trusted for SSL
	cmd := exec.Command("sudo", "security", "add-trusted-cert",
		"-d", "-r", "trustRoot",
		"-p", "ssl",
		"-k", "/Library/Keychains/System.keychain",
		tmpCertFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add certificate to keychain: %w", err)
	}

	fmt.Printf("✓ Certificate trusted: %s\n", domain)
	return nil
}

func uninstallDarwin(domain string) error {
	// Find certificate hash
	cmd := exec.Command("security", "find-certificate", "-a", "-Z", "-c", domain)
	output, err := cmd.Output()
	if err != nil {
		return nil // Not found, nothing to remove
	}

	// Extract SHA-1 hash
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "SHA-1 hash:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				hash := parts[2]
				// Delete by hash (requires root)
				delCmd := exec.Command("sudo", "security", "delete-certificate",
					"-Z", hash,
					"/Library/Keychains/System.keychain")
				delCmd.Run() // ignore errors
			}
		}
	}

	return nil
}

func isTrustedDarwin(domain string) bool {
	// Check if certificate has actual trust settings (not just existence)
	cmd := exec.Command("security", "dump-trust-settings", "-d")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Look for exact domain match with "Cert X: <domain>" format
	// Then check "Number of trust settings : N" where N > 0
	lines := strings.Split(string(output), "\n")
	foundDomain := false
	for _, line := range lines {
		// Check for exact domain match: "Cert X: domain"
		if strings.HasSuffix(strings.TrimSpace(line), ": "+domain) {
			foundDomain = true
			continue
		}
		// After finding domain, check trust settings count
		if foundDomain && strings.Contains(line, "Number of trust settings :") {
			// "Number of trust settings : 0" means NOT trusted
			// "Number of trust settings : 1" or more means trusted
			if strings.Contains(line, ": 0") {
				return false
			}
			return true
		}
		// Reset if we hit another cert entry without finding trust settings
		if foundDomain && strings.HasPrefix(strings.TrimSpace(line), "Cert ") {
			foundDomain = false
		}
	}
	return false
}

// Linux implementation (requires root)
func installLinux(certPath, domain string) error {
	// Try Debian/Ubuntu style
	if _, err := exec.LookPath("update-ca-certificates"); err == nil {
		destPath := fmt.Sprintf("/usr/local/share/ca-certificates/%s.crt", domain)

		cpCmd := exec.Command("sudo", "cp", certPath, destPath)
		if err := cpCmd.Run(); err != nil {
			return fmt.Errorf("failed to copy certificate: %w", err)
		}

		updateCmd := exec.Command("sudo", "update-ca-certificates")
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr
		if err := updateCmd.Run(); err != nil {
			return fmt.Errorf("failed to update certificates: %w", err)
		}

		fmt.Printf("✓ Certificate trusted: %s\n", domain)
		return nil
	}

	// Try RHEL/CentOS style
	if _, err := exec.LookPath("update-ca-trust"); err == nil {
		destPath := fmt.Sprintf("/etc/pki/ca-trust/source/anchors/%s.crt", domain)

		cpCmd := exec.Command("sudo", "cp", certPath, destPath)
		if err := cpCmd.Run(); err != nil {
			return fmt.Errorf("failed to copy certificate: %w", err)
		}

		updateCmd := exec.Command("sudo", "update-ca-trust", "extract")
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr
		if err := updateCmd.Run(); err != nil {
			return fmt.Errorf("failed to update certificates: %w", err)
		}

		fmt.Printf("✓ Certificate trusted: %s\n", domain)
		return nil
	}

	return fmt.Errorf("no supported certificate trust mechanism found")
}

func uninstallLinux(domain string) error {
	// Try Debian/Ubuntu style
	path1 := fmt.Sprintf("/usr/local/share/ca-certificates/%s.crt", domain)
	if _, err := os.Stat(path1); err == nil {
		exec.Command("sudo", "rm", path1).Run()
		exec.Command("sudo", "update-ca-certificates", "--fresh").Run()
		return nil
	}

	// Try RHEL/CentOS style
	path2 := fmt.Sprintf("/etc/pki/ca-trust/source/anchors/%s.crt", domain)
	if _, err := os.Stat(path2); err == nil {
		exec.Command("sudo", "rm", path2).Run()
		exec.Command("sudo", "update-ca-trust", "extract").Run()
		return nil
	}

	return nil
}

func isTrustedLinux(domain string) bool {
	path1 := fmt.Sprintf("/usr/local/share/ca-certificates/%s.crt", domain)
	path2 := fmt.Sprintf("/etc/pki/ca-trust/source/anchors/%s.crt", domain)

	if _, err := os.Stat(path1); err == nil {
		return true
	}
	if _, err := os.Stat(path2); err == nil {
		return true
	}
	return false
}
