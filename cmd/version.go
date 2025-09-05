package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

var (
	// Version information - these will be set by build flags
	Version   = "1.0.0"
	GitCommit = "dev"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
	
	// Flag for update checking
	checkUpdates bool
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `Print detailed version information for the ToneClone CLI.

Shows the CLI version, Git commit hash, build date, and Go version.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ToneClone CLI %s\n", Version)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("Go version: %s\n", GoVersion)
		fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		
		// Check for updates if requested
		if checkUpdates {
			fmt.Println() // Add blank line
			checkForVersionUpdates()
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVar(&checkUpdates, "check", false, "Check for available updates")
}

// checkForVersionUpdates checks if a newer version is available (used by version --check)
func checkForVersionUpdates() {
	fmt.Println("Checking for updates...")
	
	latest, found, err := selfupdate.DetectLatest("toneclone/cli")
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		return
	}

	if !found {
		fmt.Println("No release information found")
		return
	}

	currentVersion, err := semver.Parse(strings.TrimPrefix(Version, "v"))
	if err != nil {
		fmt.Printf("Error parsing current version: %v\n", err)
		return
	}

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Printf("Latest version:  %s\n", latest.Version)

	if latest.Version.LTE(currentVersion) {
		fmt.Println("✅ You are running the latest version!")
		return
	}

	fmt.Printf("🆙 A newer version is available: %s → %s\n", currentVersion, latest.Version)
	fmt.Printf("Release URL: %s\n", latest.URL)
	fmt.Println("\nRun 'toneclone update' to upgrade.")
}
