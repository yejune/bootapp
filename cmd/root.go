package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	composeFile string
	// Version is set at build time via -ldflags
	Version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "bootapp",
	Short: "Bootapp - Multi-project Docker networking made easy",
	Long: `Bootapp automatically manages subnets, /etc/hosts entries,
and routing for multiple Docker Compose projects.

Each project gets a unique subnet, and domains are automatically
registered in /etc/hosts pointing to the container IP.`,
	Version:          Version,
	PersistentPreRun: checkMultipleInstallations,
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

var isPlugin bool

func Execute() {
	// Check if running as Docker CLI plugin
	if len(os.Args) > 1 && os.Args[1] == "docker-cli-plugin-metadata" {
		metadata := pluginMetadata{
			SchemaVersion:    "0.1.0",
			Vendor:           "yejune",
			Version:          Version,
			ShortDescription: "Multi-project Docker networking made easy",
			URL:              "https://github.com/yejune/bootapp",
		}
		jsonBytes, _ := json.Marshal(metadata)
		fmt.Println(string(jsonBytes))
		os.Exit(0)
	}

	// When called as Docker CLI plugin, Docker passes "bootapp" as first arg
	// Strip it so cobra can process subcommands correctly
	if len(os.Args) > 1 && os.Args[1] == "bootapp" {
		isPlugin = true
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}

	// Set custom usage template based on mode
	if isPlugin {
		rootCmd.SetUsageTemplate(dockerPluginUsageTemplate())
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dockerPluginUsageTemplate() string {
	return `Usage:{{if .Runnable}}
  docker bootapp{{end}}{{if .HasAvailableSubCommands}}
  docker bootapp [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "docker bootapp [command] --help" for more information about a command.{{end}}
`
}

// ValidateSudo prompts for sudo password and caches credentials
func ValidateSudo() error {
	// Check if already running as root
	if os.Geteuid() == 0 {
		// Already running with sudo, no need to validate
		return nil
	}

	fmt.Println("Checking sudo credentials...")
	// sudo -v prompts for password if needed and extends the timeout
	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// checkMultipleInstallations detects multiple bootapp installations in PATH
func checkMultipleInstallations(cmd *cobra.Command, args []string) {
	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return
	}

	currentPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		currentPath = exePath
	}

	// Find all installations in PATH
	whichCmd := exec.Command("which", "-a", "bootapp")
	output, err := whichCmd.Output()
	if err != nil {
		return
	}

	paths := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Resolve symlinks and collect unique paths
	var resolvedPaths []string
	seen := make(map[string]bool)
	currentInPath := false

	for _, path := range paths {
		if path == "" {
			continue
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			resolved = path
		}
		if !seen[resolved] {
			resolvedPaths = append(resolvedPaths, resolved)
			seen[resolved] = true
		}
		if resolved == currentPath {
			currentInPath = true
		}
	}

	// Only warn if:
	// 1. Multiple installations in PATH
	// 2. Current executable is in PATH (not running from dev directory)
	if len(resolvedPaths) > 1 && currentInPath {
		fmt.Println("⚠️  Warning: Multiple bootapp installations detected!")
		for _, path := range resolvedPaths {
			installType := "direct"
			if strings.Contains(path, "homebrew") || strings.Contains(path, "/Cellar/") {
				installType = "Homebrew"
			}
			marker := "  "
			if path == currentPath {
				marker = "▸ "
			}
			fmt.Printf("%s %s (%s)\n", marker, path, installType)
		}
		fmt.Println()
		fmt.Println("   Consider removing duplicates to avoid confusion.")
		fmt.Println("   Run 'which bootapp' to see which one is currently active.")
		fmt.Println()
	}
}
