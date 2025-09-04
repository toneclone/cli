package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	debug   bool
	profile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "toneclone",
	Short: "ToneClone CLI - AI-powered writing assistance from the command line",
	Long: `ToneClone CLI provides command-line access to ToneClone's AI writing capabilities.

Generate text, manage personas, handle training data, and more - all from your terminal.
Perfect for automation, scripting, and integration with other tools.

Examples:
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
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.toneclone.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "debug output (includes verbose)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "configuration profile to use")

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
