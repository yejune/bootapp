package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yejune/docker-bootapp/internal/hosts"
	"github.com/yejune/docker-bootapp/internal/network"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List registered projects",
	Long:  `List all projects with their subnets, domains, and containers.`,
	RunE:  runLs,
}

func init() {
	rootCmd.AddCommand(lsCmd)
}

func runLs(cmd *cobra.Command, args []string) error {
	// Get projects
	projectMgr, err := network.NewProjectManager()
	if err != nil {
		return fmt.Errorf("failed to initialize project manager: %w", err)
	}

	projects := projectMgr.ListProjects()

	// Get hosts entries
	hostEntries, _ := hosts.ListEntries()

	if len(projects) == 0 && len(hostEntries) == 0 {
		fmt.Println("No projects registered")
		return nil
	}

	fmt.Println("Registered Projects:")
	fmt.Println(strings.Repeat("=", 70))

	for name, info := range projects {
		fmt.Printf("\n📦 %s\n", name)
		fmt.Printf("   Path:   %s\n", info.Path)
		fmt.Printf("   Subnet: %s\n", info.Subnet)
		fmt.Printf("   Domain: %s\n", info.Domain)

		// Try to load local config for container details
		localConfig, err := projectMgr.GetLocalConfig(info.Path)
		if err == nil && len(localConfig.Containers) > 0 {
			fmt.Println("   Containers:")
			for containerName, containerInfo := range localConfig.Containers {
				domains := strings.Join(containerInfo.Domains, ", ")
				fmt.Printf("     - %s: %s -> %s\n", containerName, domains, containerInfo.IP)
			}
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("/etc/hosts Entries (docker-bootapp managed):")
	fmt.Println(strings.Repeat("-", 70))

	if len(hostEntries) == 0 {
		fmt.Println("No entries found")
	} else {
		for _, entry := range hostEntries {
			fmt.Println(entry)
		}
	}

	fmt.Println()
	fmt.Println("📁 Config files:")
	homeDir, _ := os.UserHomeDir()
	fmt.Printf("   Global: %s/.docker-bootapp/projects.json\n", homeDir)
	fmt.Println("   Local:  {project}/.docker/network.json")

	return nil
}
