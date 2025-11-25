package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/bootapp/internal/compose"
	"github.com/yejune/bootapp/internal/hosts"
	"github.com/yejune/bootapp/internal/network"
	"github.com/yejune/bootapp/internal/route"
)

var (
	removeVolumes bool
	removeOrphans bool
	keepHosts     bool
	removeConfig  bool
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop and remove containers",
	Long: `Stop containers using docker-compose and optionally:
- Remove /etc/hosts entries
- Remove routing (macOS only)
- Remove project from global config`,
	RunE: runDown,
}

func init() {
	downCmd.Flags().BoolVarP(&removeVolumes, "volumes", "v", false, "Remove volumes")
	downCmd.Flags().BoolVar(&removeOrphans, "remove-orphans", false, "Remove orphan containers")
	downCmd.Flags().BoolVar(&keepHosts, "keep-hosts", false, "Keep /etc/hosts entries")
	downCmd.Flags().BoolVar(&removeConfig, "remove-config", false, "Remove project from global config")
	rootCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, args []string) error {
	// Validate sudo credentials upfront (required for /etc/hosts modification)
	if !keepHosts {
		if err := ValidateSudo(); err != nil {
			fmt.Println("\nOr use --keep-hosts to skip /etc/hosts modification:")
			fmt.Println("   docker bootapp down --keep-hosts")
			return fmt.Errorf("sudo authentication failed: %w", err)
		}
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

	// Get project info
	projectName := compose.GetProjectName(composePath)
	fmt.Printf("Project: %s\n", projectName)

	// Initialize project manager
	projectMgr, err := network.NewProjectManager()
	if err != nil {
		return fmt.Errorf("failed to initialize project manager: %w", err)
	}

	// Get project info for subnet
	projectInfo, hasProject := projectMgr.GetProject(projectName)

	// Run docker-compose down
	fmt.Println("\nStopping containers...")
	if err := runDockerComposeDown(composePath, projectName); err != nil {
		return err
	}

	// Remove /etc/hosts entries
	if !keepHosts {
		fmt.Println("\nCleaning up /etc/hosts...")
		if err := hosts.RemoveProjectEntries(projectName); err != nil {
			fmt.Printf("Warning: Failed to clean /etc/hosts: %v\n", err)
		} else {
			fmt.Println("Removed /etc/hosts entries")
		}
	}

	// Remove route (macOS only) - use subnet from global config
	if hasProject {
		fmt.Println("\nCleaning up routing...")
		if err := route.RemoveRoute(projectInfo.Subnet); err != nil {
			fmt.Printf("Warning: Failed to remove route: %v\n", err)
		}
	}

	// Remove config if requested
	if removeConfig {
		fmt.Println("\nRemoving project configuration...")
		if err := projectMgr.RemoveProject(projectName); err != nil {
			fmt.Printf("Warning: Failed to remove project config: %v\n", err)
		} else {
			fmt.Println("Removed from global config")
		}
	}

	fmt.Println("\n✅ Containers stopped")
	fmt.Println("\n📁 Configuration: ~/.bootapp/projects.json")

	return nil
}

func runDockerComposeDown(composePath, projectName string) error {
	// Use "docker compose" (V2) instead of "docker-compose"
	args := []string{"compose", "-f", composePath, "-p", projectName, "down"}

	if removeVolumes {
		args = append(args, "-v")
	}
	if removeOrphans {
		args = append(args, "--remove-orphans")
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = filepath.Dir(composePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
