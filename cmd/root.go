package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var composeFile string

var rootCmd = &cobra.Command{
	Use:   "bootapp",
	Short: "Docker Bootapp - Multi-project Docker networking made easy",
	Long: `Docker Bootapp automatically manages subnets, /etc/hosts entries,
and routing for multiple Docker Compose projects.

Each project gets a unique subnet, and domains are automatically
registered in /etc/hosts pointing to the container IP.`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&composeFile, "file", "f", "", "Compose file (default: auto-detect)")
}

// Docker CLI Plugin metadata
type pluginMetadata struct {
	SchemaVersion    string `json:"SchemaVersion"`
	Vendor           string `json:"Vendor"`
	Version          string `json:"Version"`
	ShortDescription string `json:"ShortDescription"`
	URL              string `json:"URL,omitempty"`
}

func Execute() {
	// Check if running as Docker CLI plugin
	if len(os.Args) > 1 && os.Args[1] == "docker-cli-plugin-metadata" {
		metadata := pluginMetadata{
			SchemaVersion:    "0.1.0",
			Vendor:           "yejune",
			Version:          "1.0.0",
			ShortDescription: "Multi-project Docker networking made easy",
			URL:              "https://github.com/yejune/docker-bootapp",
		}
		jsonBytes, _ := json.Marshal(metadata)
		fmt.Println(string(jsonBytes))
		os.Exit(0)
	}

	// When called as Docker CLI plugin, Docker passes "bootapp" as first arg
	// Strip it so cobra can process subcommands correctly
	if len(os.Args) > 1 && os.Args[1] == "bootapp" {
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ValidateSudo prompts for sudo password and caches credentials
func ValidateSudo() error {
	fmt.Println("Checking sudo credentials...")
	// sudo -v prompts for password if needed and extends the timeout
	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
