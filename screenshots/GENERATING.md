# Generating the ToneClone CLI Demo GIF

This guide explains how to generate the demo GIF for the ToneClone CLI.

## Prerequisites

1. **VHS** - Terminal recording tool by Charm
   ```bash
   brew install vhs
   ```

2. **jq** - JSON processor (for cleanup script)
   ```bash
   brew install jq
   ```

3. **ToneClone CLI** - Installed and available in PATH
   ```bash
   which toneclone
   ```

4. **Writing Samples** - Located at `~/projects/writing-samples/jon/emails/`

## Quick Start

### First Time Setup

1. Navigate to the screenshots directory:
   ```bash
   cd screenshots
   ```

2. Run the setup script to configure the demo account:
   ```bash
   ./setup-demo.sh
   ```

3. Generate the GIF:
   ```bash
   vhs demo.tape
   ```

4. The output will be `demo.gif` in the current directory

### Regenerating the GIF

If the demo account is already set up:

```bash
cd screenshots
vhs demo.tape
```

## Demo Account Details

- **API Key**: `tc_live_6GKKFXTXIF3O5U7WFLTNKDIALSUQKGDMB6YD2HYRAA3L56IAWR4Q`
- **Profile Name**: `demo`

### Personas Created

1. **Developer** - Engineering communication, commit messages, technical docs
2. **Product Marketer** - Marketing copy, announcements, launches
3. **Technical Writer** - API docs, guides, release notes

### Knowledge Cards Created

1. **Git Commits** - Conventional commit format, clear descriptions
2. **Product Launch** - Marketing copy for product announcements
3. **Release Notes** - Changelog format, feature descriptions

### Training Data

4 sample emails are uploaded and associated with the "Developer" persona to demonstrate the training capability.

### Sample Git Diff

A `sample.diff` file shows a simple, realistic code change (adding a REST API endpoint for user profile retrieval) to demonstrate commit message generation by piping git diff output into the CLI.

## Demo Flow

The GIF demonstrates the following features (~50-60 seconds):

1. **Help Command** (~4s) - Shows available commands
2. **Commit Message Generation** (~10s) - Shows `git diff |` being typed, but actually pipes `sample.diff` to demonstrate realistic workflow
3. **Marketing Copy** (~12s) - Using Product Marketer persona + Product Launch knowledge
4. **List Personas** (~6s) - Shows all available personas
5. **List Knowledge** (~6s) - Shows all knowledge cards

> **Note**: The demo uses a bash function to override `git` command, making `git diff` output the contents of `sample.diff`. This allows the demo to show realistic `git diff |` usage while using a controlled sample file for consistent results.

## Customization

### Modifying the Demo Flow

Edit `demo.tape` to change:
- Commands demonstrated
- Typing speed (`Set TypingSpeed`)
- Sleep durations between commands
- Terminal size and theme
- Prompts and personas used

### Changing Visual Style

In `demo.tape`, modify:
```
Set FontSize 16
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
```

Available themes: https://github.com/charmbracelet/vhs#themes

### Adding More Training Data

Edit `setup-demo.sh` to add more files:
```bash
toneclone training add --file="$SAMPLES_DIR/filename.txt" --persona="Developer"
```

## Cleanup

To remove all demo data from the account:

```bash
./cleanup-demo.sh
```

This will delete:
- All user-created personas
- All knowledge cards
- All training files

Note: The demo profile will remain in `~/.toneclone.yaml`. To remove it:

```bash
toneclone auth logout
```

## Troubleshooting

### VHS Not Found

Install VHS:
```bash
brew install vhs
```

### API Authentication Errors

Re-run the setup script:
```bash
./setup-demo.sh
```

Or manually login:
```bash
echo "tc_live_6GKKFXTXIF3O5U7WFLTNKDIALSUQKGDMB6YD2HYRAA3L56IAWR4Q" | toneclone auth login --name="demo" --from-stdin --force
```

### Writing Samples Not Found

Update the `SAMPLES_DIR` path in `setup-demo.sh`:
```bash
SAMPLES_DIR="/path/to/your/writing/samples"
```

### GIF Too Large

Reduce the dimensions or duration in `demo.tape`:
```
Set Width 1000
Set Height 500
```

Or reduce sleep times between commands.

## Tips

- Keep the demo under 60 seconds for better engagement
- Use realistic typing speed for authenticity
- Pause between commands so viewers can read
- Show practical use cases that resonate with developers
- Test the tape file before committing to ensure timing is right

## Files

- `demo.tape` - VHS recording script
- `setup-demo.sh` - Automated setup script
- `cleanup-demo.sh` - Cleanup script
- `sample.diff` - Sample git diff for commit message demo
- `demo.gif` - Generated output (not committed to git)
- `GENERATING.md` - This file
