package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

var (
	checkOnly    bool
	forceUpdate  bool
	prerelease   bool
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update ToneClone CLI to the latest version",
	Long: `Update ToneClone CLI to the latest version from GitHub releases.

This command will:
- Check for the latest release on GitHub
- Download and replace the current binary if a newer version is available
- Preserve the installation method when possible
- Show progress during download

Examples:
  toneclone update              # Update to latest version
  toneclone update --check      # Only check for updates
  toneclone update --force      # Force update even if same version
  toneclone update --prerelease # Include pre-release versions`,
	Run: func(cmd *cobra.Command, args []string) {
		if checkOnly {
			checkForUpdates()
			return
		}
		
		performUpdate()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	
	updateCmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates without downloading")
	updateCmd.Flags().BoolVar(&forceUpdate, "force", false, "Force update even if current version is latest")
	updateCmd.Flags().BoolVar(&prerelease, "prerelease", false, "Include pre-release versions")
}

// checkForUpdates checks if a newer version is available
func checkForUpdates() {
	fmt.Println("Checking for updates...")
	
	latest, found, err := selfupdate.DetectLatest("toneclone/cli")
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if !found {
		fmt.Println("No release information found")
		fmt.Printf("Debug: looking for repository: toneclone/cli\n")
		fmt.Printf("Debug: current platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
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

// performUpdate downloads and installs the latest version
func performUpdate() {
	// Detect installation method
	installMethod := detectInstallationMethod()
	
	if installMethod == "homebrew" {
		fmt.Println("🍺 Homebrew installation detected.")
		fmt.Println("Please update using: brew upgrade toneclone")
		fmt.Println("Or: brew update && brew upgrade toneclone")
		return
	}

	fmt.Println("Checking for updates...")
	
	// Parse current version
	currentVersion, err := semver.Parse(strings.TrimPrefix(Version, "v"))
	if err != nil {
		fmt.Printf("Error parsing current version: %v\n", err)
		os.Exit(1)
	}

	// Check for latest version
	latest, found, err := selfupdate.DetectLatest("toneclone/cli")
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if !found {
		fmt.Println("No release information found")
		return
	}

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Printf("Latest version:  %s\n", latest.Version)

	// Check if update is needed
	if latest.Version.LTE(currentVersion) && !forceUpdate {
		fmt.Println("✅ You are already running the latest version!")
		if !forceUpdate {
			return
		}
		fmt.Println("Forcing update due to --force flag...")
	}

	// Perform the update
	fmt.Printf("🔄 Updating from %s to %s...\n", currentVersion, latest.Version)
	
	// Show download progress
	fmt.Println("📥 Downloading update...")
	
	// Perform the update using the simpler API
	release, err := selfupdate.UpdateSelf(currentVersion, "toneclone/cli")
	if err != nil {
		fmt.Printf("❌ Update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully updated to %s!\n", release.Version)
	fmt.Println("🎉 ToneClone CLI has been updated. Restart any running instances to use the new version.")
	
	// Show release notes if available
	if release.ReleaseNotes != "" {
		fmt.Println("\n📋 Release Notes:")
		fmt.Println(release.ReleaseNotes)
	}
}

// detectInstallationMethod tries to determine how ToneClone was installed
func detectInstallationMethod() string {
	exe, err := os.Executable()
	if err != nil {
		return "unknown"
	}

	exePath, err := filepath.EvalSymlinks(exe)
	if err != nil {
		exePath = exe
	}

	// Check for Homebrew installation
	if strings.Contains(exePath, "/usr/local/Cellar/toneclone") ||
		strings.Contains(exePath, "/opt/homebrew/Cellar/toneclone") ||
		strings.Contains(exePath, "/usr/local/bin/toneclone") && isHomebrewManaged(exePath) {
		return "homebrew"
	}

	// Check for go install
	if strings.Contains(exePath, "/go/bin/") {
		return "go-install"
	}

	// Default to manual installation
	return "manual"
}

// isHomebrewManaged checks if a binary is managed by Homebrew
func isHomebrewManaged(path string) bool {
	// Check if it's a symlink to a Homebrew Cellar location
	if link, err := os.Readlink(path); err == nil {
		return strings.Contains(link, "/Cellar/toneclone")
	}
	
	return false
}

