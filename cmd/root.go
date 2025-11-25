package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

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
	Version: Version,
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
