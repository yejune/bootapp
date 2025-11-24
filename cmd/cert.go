package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/docker-bootapp/internal/cert"
	"github.com/yejune/docker-bootapp/internal/compose"
)

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Manage SSL certificates",
	Long:  `Generate and manage self-signed SSL certificates for local development.`,
}

var certListCmd = &cobra.Command{
	Use:   "list",
	Short: "List certificates",
	RunE:  runCertList,
}

var certGenerateCmd = &cobra.Command{
	Use:   "generate [domain...]",
	Short: "Generate certificate for domain(s)",
	RunE:  runCertGenerate,
}

var certInstallCmd = &cobra.Command{
	Use:   "install [domain...]",
	Short: "Install certificate to system trust store",
	RunE:  runCertInstall,
}

var certUninstallCmd = &cobra.Command{
	Use:   "uninstall [domain...]",
	Short: "Remove certificate from system trust store",
	RunE:  runCertUninstall,
}

var certDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect SSL_DOMAINS and generate certificates",
	RunE:  runCertDetect,
}

func init() {
	certCmd.AddCommand(certListCmd)
	certCmd.AddCommand(certGenerateCmd)
	certCmd.AddCommand(certInstallCmd)
	certCmd.AddCommand(certUninstallCmd)
	certCmd.AddCommand(certDetectCmd)
	rootCmd.AddCommand(certCmd)
}

func getCertDir() string {
	return filepath.Join(".", "var", "certs")
}

func runCertList(cmd *cobra.Command, args []string) error {
	certDir := getCertDir()
	domains, err := cert.ListCerts(certDir)
	if err != nil {
		return err
	}

	if len(domains) == 0 {
		fmt.Println("No certificates found in", certDir)
		return nil
	}

	fmt.Printf("Certificates in %s:\n", certDir)
	for _, domain := range domains {
		trusted := ""
		if cert.IsTrusted(domain) {
			trusted = " [trusted]"
		}
		fmt.Printf("  %s%s\n", domain, trusted)
	}
	return nil
}

func runCertGenerate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please specify domain(s)")
	}

	certDir := getCertDir()
	info := cert.DefaultCertInfo()

	for _, domain := range args {
		if cert.CertExists(domain, certDir) {
			fmt.Printf("Certificate already exists: %s\n", domain)
			continue
		}

		fmt.Printf("Generating certificate for %s...\n", domain)
		if err := cert.GenerateCert(domain, certDir, info); err != nil {
			return fmt.Errorf("failed to generate cert for %s: %w", domain, err)
		}
		fmt.Printf("✓ Generated: %s/%s.{crt,key,pem}\n", certDir, domain)
	}
	return nil
}

func runCertInstall(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please specify domain(s)")
	}

	// Validate sudo credentials upfront
	if err := ValidateSudo(); err != nil {
		return fmt.Errorf("sudo authentication failed: %w", err)
	}

	certDir := getCertDir()

	fmt.Println("Installing certificates to system trust store...")

	for _, domain := range args {
		if !cert.CertExists(domain, certDir) {
			fmt.Printf("Certificate not found: %s (run 'cert generate' first)\n", domain)
			continue
		}

		if err := cert.InstallToTrustStore(domain, certDir); err != nil {
			fmt.Printf("Failed to install %s: %v\n", domain, err)
		}
	}
	return nil
}

func runCertUninstall(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please specify domain(s)")
	}

	// Validate sudo credentials upfront
	if err := ValidateSudo(); err != nil {
		return fmt.Errorf("sudo authentication failed: %w", err)
	}

	fmt.Println("Removing certificates from system trust store...")

	for _, domain := range args {
		if err := cert.UninstallFromTrustStore(domain); err != nil {
			fmt.Printf("Failed to uninstall %s: %v\n", domain, err)
		} else {
			fmt.Printf("✓ Removed: %s\n", domain)
		}
	}
	return nil
}

func runCertDetect(cmd *cobra.Command, args []string) error {
	// Find compose file
	var composePath string
	var err error

	if composeFile != "" {
		composePath, err = filepath.Abs(composeFile)
		if err != nil {
			return fmt.Errorf("invalid compose file path: %w", err)
		}
	} else {
		composePath, err = compose.FindComposeFile()
		if err != nil {
			return fmt.Errorf("no compose file found: %w", err)
		}
	}

	// Parse compose file
	composeData, err := compose.ParseComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Extract SSL domains
	serviceDomains := compose.ExtractServiceDomains(composeData)
	if len(serviceDomains) == 0 {
		fmt.Println("No SSL_DOMAINS found in compose file")
		return nil
	}

	// Collect unique domains
	domainSet := make(map[string]bool)
	for _, domains := range serviceDomains {
		for _, d := range domains {
			domainSet[d] = true
		}
	}

	certDir := getCertDir()
	info := cert.DefaultCertInfo()

	fmt.Printf("Detected domains from %s:\n", filepath.Base(composePath))
	for domain := range domainSet {
		fmt.Printf("  %s\n", domain)
	}

	fmt.Println("\nGenerating certificates...")
	for domain := range domainSet {
		if cert.CertExists(domain, certDir) {
			fmt.Printf("  %s [exists]\n", domain)
			continue
		}

		if err := cert.GenerateCert(domain, certDir, info); err != nil {
			fmt.Printf("  %s [failed: %v]\n", domain, err)
		} else {
			fmt.Printf("  %s [generated]\n", domain)
		}
	}

	fmt.Printf("\n✓ Certificates saved to %s\n", certDir)
	fmt.Println("\nTo install to system trust store, run:")
	fmt.Println("  docker bootapp cert install <domain>")

	return nil
}

// GenerateAndInstallCerts generates and optionally installs certs for domains
// Used by 'up' command
func GenerateAndInstallCerts(domains []string, certDir string, install bool) error {
	if len(domains) == 0 {
		return nil
	}

	info := cert.DefaultCertInfo()

	for _, domain := range domains {
		if !cert.CertExists(domain, certDir) {
			fmt.Printf("Generating certificate for %s...\n", domain)
			if err := cert.GenerateCert(domain, certDir, info); err != nil {
				return fmt.Errorf("failed to generate cert for %s: %w", domain, err)
			}
		}

		if install && !cert.IsTrusted(domain) {
			if err := cert.InstallToTrustStore(domain, certDir); err != nil {
				fmt.Printf("Warning: Failed to trust %s: %v\n", domain, err)
			}
		}
	}
	return nil
}
