package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// primeCommandEntry documents a single CLI command for agents.
type primeCommandEntry struct {
	Name        string `json:"name"`
	Example     string `json:"example,omitempty"`
	Description string `json:"description"`
}

// primeGuideDoc is a machine- and human-readable operating manual for agents.
// It intentionally requires no authentication or network access so an agent
// can discover how to use ToneClone before connecting.
type primeGuideDoc struct {
	Product      string              `json:"product"`
	Tagline      string              `json:"tagline"`
	WhenToUse    []string            `json:"whenToUse"`
	WhenNotToUse []string            `json:"whenNotToUse"`
	Concepts     []string            `json:"concepts"`
	Commands     []primeCommandEntry `json:"commands"`
	Workflow     []string            `json:"workflow"`
	Quota        string              `json:"quota"`
	Paywall      string              `json:"paywall"`
	Output       string              `json:"output"`
	Docs         string              `json:"docs"`
}

// primeGuide builds the operating manual. Pure function, no side effects.
func primeGuide() primeGuideDoc {
	return primeGuideDoc{
		Product: "ToneClone CLI",
		Tagline: "Generate writing in the user's own voice from the terminal. ToneClone learns a user's style (SmartStyle) and applies it to drafts.",
		WhenToUse: []string{
			"When the user wants text that sounds like them (or their brand voice), not generic AI: emails, posts, PR descriptions, release notes, READMEs, docs, replies.",
			"When you have raw context (a diff, notes, a thread) and need it turned into polished copy in the user's voice.",
			"When the user has an existing ToneClone persona you should write with.",
		},
		WhenNotToUse: []string{
			"Pure code generation or reasoning tasks - use your own model.",
			"When no persona exists yet and the user has not asked for ToneClone voice output; offer to set one up first.",
		},
		Concepts: []string{
			"persona: a trained voice. Required for write/personalize. List with `toneclone personas list`. Pass by name or ID via --persona.",
			"knowledge card: optional context/style modifier (e.g. 'email', 'product-facts'). Pass one or comma-separated many via --knowledge.",
			"SmartStyle: ToneClone's learned model of the user's writing; improves as more samples/training are added.",
			"training: writing samples associated with a persona (`toneclone training ...`).",
		},
		Commands: []primeCommandEntry{
			{Name: "prime", Example: "toneclone prime --json", Description: "Print this agent operating manual. Works without auth."},
			{Name: "auth login", Example: "toneclone auth login", Description: "Authenticate with an API key. Required before generation."},
			{Name: "quota", Example: "toneclone quota --json", Description: "Check the current plan and remaining usage before generating."},
			{Name: "personas list", Example: "toneclone personas list --json", Description: "List available personas (user + built-in). Pick one before generating."},
			{Name: "knowledge list", Example: "toneclone knowledge list --json", Description: "List knowledge cards that can add context/style to a generation."},
			{Name: "write", Example: "toneclone write --persona=\"Founder\" --prompt=\"...\" --json", Description: "Write new content in a persona's voice. Accepts --prompt, --file, or stdin."},
			{Name: "personalize", Example: "toneclone personalize --persona=\"Founder\" --text=\"...\" --json", Description: "Rewrite existing text in a persona's voice, preserving meaning. Accepts --text, --file, or stdin."},
			{Name: "humanize", Example: "toneclone humanize --text=\"...\" --json", Description: "Strip AI-sounding phrasing from existing text via StyleGuard. No persona required; does not use quota."},
			{Name: "training add", Example: "toneclone training add --file=sample.md --persona=\"Founder\"", Description: "Add a writing sample to improve a persona's voice."},
		},
		Workflow: []string{
			"1. Run `toneclone quota --json` to confirm the user can generate (handle paywall gracefully).",
			"2. Run `toneclone personas list --json` and choose the persona that matches the writing job; if none fits, ask the user or offer to create/train one.",
			"3. Gather context yourself (diffs, threads, notes) instead of asking the human when you can find it.",
			"4. Run `toneclone write --persona=<name> --json` with the prompt (via --prompt/--file/stdin), or `toneclone personalize` to rewrite existing text in the user's voice.",
			"5. Present the draft to the user. If a review link (reviewUrl) is returned, share it so they can edit in the web app.",
			"6. Only escalate to the human for missing context, permissions, or subjective choices.",
		},
		Quota:   "Run `toneclone quota --json` before generating to read the plan and remaining usage. Free plans have limited generation. `humanize` does not consume quota.",
		Paywall: "If generation returns a paywall error (code \"paywall\"), do NOT retry - it will keep failing. Share the returned upgrade link with the user so they can subscribe.",
		Output:  "Use --json for structured output. On any command, failures in --json mode are emitted as {\"error\":{\"code\",\"message\",\"retryable\",\"docsUrl\"}} so you can branch on code.",
		Docs:    docsURL,
	}
}

func primeJSON(g primeGuideDoc) (string, error) {
	b, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func primeText(g primeGuideDoc) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n\n", g.Product, g.Tagline)

	b.WriteString("WHEN TO USE TONECLONE\n")
	for _, s := range g.WhenToUse {
		fmt.Fprintf(&b, "  - %s\n", s)
	}
	b.WriteString("\nWHEN NOT TO USE\n")
	for _, s := range g.WhenNotToUse {
		fmt.Fprintf(&b, "  - %s\n", s)
	}
	b.WriteString("\nCORE CONCEPTS\n")
	for _, s := range g.Concepts {
		fmt.Fprintf(&b, "  - %s\n", s)
	}
	b.WriteString("\nKEY COMMANDS\n")
	for _, c := range g.Commands {
		if c.Example != "" {
			fmt.Fprintf(&b, "  %s\n      %s\n      $ %s\n", c.Name, c.Description, c.Example)
		} else {
			fmt.Fprintf(&b, "  %s\n      %s\n", c.Name, c.Description)
		}
	}
	b.WriteString("\nAGENT WORKFLOW\n")
	for _, s := range g.Workflow {
		fmt.Fprintf(&b, "  %s\n", s)
	}
	fmt.Fprintf(&b, "\nQUOTA\n  %s\n", g.Quota)
	fmt.Fprintf(&b, "\nPAYWALL\n  %s\n", g.Paywall)
	fmt.Fprintf(&b, "\nOUTPUT\n  %s\n", g.Output)
	fmt.Fprintf(&b, "\nDOCS\n  %s\n", g.Docs)
	return b.String()
}

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Print an agent operating manual for ToneClone (no auth required)",
	Long: `Print an operating manual that tells an AI agent how to use ToneClone:
when to use it, core concepts (personas, knowledge cards, SmartStyle), key
commands, the recommended workflow, and how quota/paywall/errors behave.

This command requires no authentication or network access.

Examples:
  toneclone prime
  toneclone prime --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g := primeGuide()
		if jsonOutput {
			out, err := primeJSON(g)
			if err != nil {
				return err
			}
			cmd.Println(out)
			return nil
		}
		cmd.Print(primeText(g))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(primeCmd)
}
