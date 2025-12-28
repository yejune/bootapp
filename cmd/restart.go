package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/bootapp/internal/compose"
)

var restartCmd = &cobra.Command{
	Use:   "restart [service...]",
	Short: "Restart containers",
	Long: `Restart containers using docker-compose restart.

This command restarts containers without recreating them, so:
- Container IPs are preserved
- /etc/hosts entries remain valid
- No network reconfiguration needed

Examples:
  bootapp restart           # Restart all services
  bootapp restart web       # Restart only 'web' service
  bootapp restart web api   # Restart 'web' and 'api' services`,
	RunE: runRestart,
}

func init() {
	rootCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	// Find or use specified docker-compose file
	var composePath string
	var err error

	if composeFile != "" {
		composePath, err = filepath.Abs(composeFile)
		if err != nil {
			return fmt.Errorf("invalid compose file path: %w", err)
		}
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			return fmt.Errorf("compose file not found: %s", composePath)
		}
	} else {
		composePath, err = compose.FindComposeFile()
		if err != nil {
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

	// Parse compose file for project name
	composeData, err := compose.ParseComposeFile(composePath)
	if err != nil {
		return fmt.Errorf("failed to parse compose file: %w", err)
	}

	projectName := compose.GetProjectName(composePath, composeData)
	fmt.Printf("Project: %s\n", projectName)

	// Build docker compose restart command
	dockerArgs := []string{"compose", "-f", composePath, "-p", projectName, "restart"}

	// Add specific services if provided
	if len(args) > 0 {
		dockerArgs = append(dockerArgs, args...)
		fmt.Printf("\nRestarting services: %v\n", args)
	} else {
		fmt.Println("\nRestarting all services...")
	}

	dockerCmd := exec.Command("docker", dockerArgs...)
	dockerCmd.Dir = filepath.Dir(composePath)
	dockerCmd.Stdin = os.Stdin
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	if err := dockerCmd.Run(); err != nil {
		return fmt.Errorf("restart failed: %w", err)
	}

	fmt.Println("\n✅ Restart complete (IPs preserved)")

	return nil
}
