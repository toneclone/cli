package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for ToneClone CLI.

The completion script must be evaluated in your shell to provide completions.
This can be done by including it in your shell's configuration file.

Examples:
  # Generate bash completion
  toneclone completion bash > /etc/bash_completion.d/toneclone
  
  # Generate zsh completion
  toneclone completion zsh > "${fpath[1]}/_toneclone"
  
  # Generate fish completion
  toneclone completion fish > ~/.config/fish/completions/toneclone.fish
  
  # Generate powershell completion
  toneclone completion powershell > toneclone.ps1

Installation:
  # Bash (Linux):
  sudo toneclone completion bash > /etc/bash_completion.d/toneclone
  
  # Bash (macOS with Homebrew):
  toneclone completion bash > /usr/local/etc/bash_completion.d/toneclone
  
  # Zsh:
  toneclone completion zsh > ~/.zsh/completions/_toneclone
  # Add to ~/.zshrc: fpath=(~/.zsh/completions $fpath)
  
  # Fish:
  toneclone completion fish > ~/.config/fish/completions/toneclone.fish
  
  # PowerShell:
  toneclone completion powershell > toneclone.ps1
  # Add to PowerShell profile: . .\toneclone.ps1

Quick Setup:
  # Test completion without installing
  source <(toneclone completion bash)     # Bash
  toneclone completion fish | source      # Fish
  toneclone completion zsh | source /dev/stdin  # Zsh`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
