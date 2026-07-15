package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	debug   bool
	profile string
)

// docsURL is the canonical docs/help link surfaced in structured errors and the
// prime guide.
const docsURL = "https://toneclone.ai"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "toneclone",
	Short: "ToneClone CLI - AI-powered writing assistance from the command line",
	Long: `ToneClone CLI provides command-line access to ToneClone's AI writing capabilities.

Generate text, manage personas, handle training data, and more - all from your terminal.
Perfect for automation, scripting, and integration with other tools.

For agents: run 'toneclone prime' first for an operating manual (no auth required).
Use --json for structured output; failures are emitted as a structured JSON error
when --json is set.

Examples:
  toneclone prime --json
  toneclone write --persona=professional --prompt="Write a product description"
  toneclone personas list
  toneclone training add --file=data.txt --persona=writer

Get started by configuring your API key:
  toneclone auth login

For more help on any command, use:
  toneclone [command] --help`,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	// Reset per-execution JSON mode before Cobra parses flags. Cobra will set
	// jsonOutput again for the real --json flag; this pre-scan exists only for
	// the hidden write --output=json compatibility alias so centralized error
	// rendering can emit JSON even for early validation failures.
	jsonOutput = writeOutputAliasRequestsJSON(os.Args[1:])

	// Silence cobra's built-in error and usage printing so we can render errors
	// ourselves (a structured envelope in --json mode) and keep runtime failures
	// clean for scripts and agents. Argument/flag mistakes still print a clear
	// message via renderCommandError; run `--help` for full usage.
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	err := rootCmd.Execute()
	if err != nil {
		renderCommandError(err)
	}
	return err
}

func writeOutputAliasRequestsJSON(args []string) bool {
	writeIndex := -1
	for i, arg := range args {
		if arg == "write" {
			writeIndex = i
			break
		}
	}
	if writeIndex == -1 {
		return false
	}
	for i := writeIndex + 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--output" && i+1 < len(args):
			return strings.EqualFold(args[i+1], "json")
		case strings.HasPrefix(arg, "--output="):
			return strings.EqualFold(strings.TrimPrefix(arg, "--output="), "json")
		}
	}
	return false
}

func init() {
	cobra.OnInitialize(initConfig)

	// Cobra defaults successful command output to stderr unless an output writer
	// is configured. Keep successful, pipeable output on stdout while errors and
	// diagnostics continue to use stderr.
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.toneclone.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "debug output (includes verbose)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "configuration profile to use")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in structured JSON (including errors)")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".toneclone" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".toneclone")
	}

	// Environment variables
	viper.SetEnvPrefix("TONECLONE")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && (verbose || debug) {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}
