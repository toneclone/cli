# ToneClone CLI

A powerful command-line interface for ToneClone, the AI-powered writing assistance platform. Generate text, manage personas, configure profiles, and streamline your writing workflow from the terminal.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Authentication](#authentication)
- [Core Commands](#core-commands)
- [Configuration](#configuration)
- [Shell Completion](#shell-completion)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/toneclone/cli.git
cd cli

# Build the CLI
make build

# Install to your PATH (optional)
sudo cp bin/toneclone /usr/local/bin/
```

### Verify Installation

```bash
toneclone --version
toneclone --help
```

## Quick Start

1. **Login with your API key**:
   ```bash
   toneclone auth login
   ```

2. **Check your account**:
   ```bash
   toneclone user whoami
   ```

3. **Generate your first text**:
   ```bash
   toneclone generate "Write a professional email about project updates"
   ```

4. **List available personas**:
   ```bash
   toneclone personas list
   ```

## Authentication

### Login with API Key

```bash
# Interactive login
toneclone auth login

# Or provide API key directly
toneclone auth login --key="your-api-key" --name="work"
```

### Manage API Keys

```bash
# List configured keys
toneclone auth list

# Switch between keys
toneclone auth use work

# Check current key
toneclone auth current

# Logout (remove key)
toneclone auth logout
```

### Environment Variables

```bash
# Set API key via environment variable
export TONECLONE_API_KEY="your-api-key"
export TONECLONE_BASE_URL="https://api.toneclone.ai"

# Use with CLI
toneclone generate "Hello world"
```

## Core Commands

### Text Generation

```bash
# Basic text generation
toneclone generate "Write a blog post about AI"

# With persona
toneclone write "Draft a proposal" --persona="Professional"

# With profile
toneclone write "Write an email" --profile="Email Template"

# Interactive mode
toneclone write --persona=casual

# Streaming output
toneclone write "Long story"

# Save to file
toneclone write "Content" --output="result.txt"
```

### Persona Management

```bash
# List all personas
toneclone personas list

# Get persona details
toneclone personas get persona-id

# Create new persona
toneclone personas create --name="Technical Writer"

# Interactive creation
toneclone personas create --interactive

# Update persona
toneclone personas update persona-id --name="New Name"

# Delete persona
toneclone personas delete persona-id --confirm
```

### Profile Management

```bash
# List profiles
toneclone profiles list

# Create profile
toneclone profiles create --name="Email Template" --instructions="Write professional emails"

# Update profile
toneclone profiles update profile-id --name="New Name"

# Append to instructions
toneclone profiles update profile-id --append=" Include examples."

# Associate with persona
toneclone profiles associate --profile-id=123 --persona="Professional"

# Delete profile
toneclone profiles delete profile-id --confirm
```

### Training Data Management

```bash
# List training files
toneclone training list

# Upload text content
toneclone training add --text="Sample content" --filename="sample.txt" --persona="Writer"

# Upload file
toneclone training add --file="document.pdf" --persona="Professional"

# Bulk upload directory
toneclone training add --directory="./docs" --recursive --persona="Technical"

# Associate file with persona
toneclone training associate --file-id=123 --persona="Professional"

# Remove training file
toneclone training remove --file-id=123 --confirm
```

## Configuration

### Configuration Management

```bash
# Show current configuration
toneclone config show

# List API keys
toneclone config list

# Validate configuration
toneclone config validate

# Show config file path
toneclone config path

# Initialize new config
toneclone config init
```

### Configuration File

The CLI uses a YAML configuration file located at `~/.toneclone.yaml`:

```yaml
default_key: "work"
keys:
  work:
    key: "tc_live_..."
    base_url: "https://api.toneclone.ai"
  personal:
    key: "tc_live_..."
    base_url: "https://api.toneclone.ai"
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TONECLONE_API_KEY` | API key for authentication | - |
| `TONECLONE_BASE_URL` | Base URL for API | `https://api.toneclone.ai` |
| `TONECLONE_PROFILE` | Profile/key name to use | `default` |

## Shell Completion

### Install Completion

```bash
# Bash
toneclone completion bash > /etc/bash_completion.d/toneclone

# Zsh
toneclone completion zsh > "${fpath[1]}/_toneclone"

# Fish
toneclone completion fish > ~/.config/fish/completions/toneclone.fish

# PowerShell
toneclone completion powershell > toneclone.ps1
```

### Quick Setup

```bash
# Test completion without installing
source <(toneclone completion bash)     # Bash
toneclone completion fish | source      # Fish
toneclone completion zsh | source /dev/stdin  # Zsh
```

## User Management

```bash
# Check current user
toneclone user whoami

# Detailed user info
toneclone user info

# View user settings
toneclone user settings
```

## Health & Diagnostics

```bash
# Quick ping test
toneclone ping

# Basic health check
toneclone health

# Comprehensive status
toneclone health status

# JSON output
toneclone health --format=json
```

## Examples

### Daily Workflow

```bash
# Morning routine
toneclone auth current
toneclone health
toneclone personas list

# Generate content
toneclone write "Write a project status update" --persona="Professional" --profile="Email"

# Save to file
toneclone write "Weekly report content" --output="report.md"
```

### Content Creation

```bash
# Blog post
toneclone write "AI trends in 2024" --persona="Tech Writer" --profile="Blog Post"

# Email campaigns
toneclone write "Newsletter content" --persona="Marketing" --profile="Email Campaign"

# Documentation
toneclone write "API documentation for users endpoint" --persona="Technical" --profile="Documentation"
```

### Persona Setup

```bash
# Create specialized personas
toneclone personas create --name="Social Media Manager"

toneclone personas create --name="Technical Writer"

# Create matching profiles
toneclone profiles create \
  --name="Twitter Post" \
  --instructions="Write engaging Twitter posts under 280 characters"

toneclone profiles create \
  --name="Documentation" \
  --instructions="Write clear, concise technical documentation with examples"
```

### Training Data Management

```bash
# Upload training materials
toneclone training add --file="brand-guidelines.pdf" --persona="Marketing"
toneclone training add --directory="./docs" --recursive --persona="Technical"

# Manage file associations
toneclone training associate --file-id=456 --persona="Professional"
```

## Output Formats

Most commands support multiple output formats:

```bash
# Table format (default)
toneclone personas list

# JSON format
toneclone personas list --format=json
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Config file path |
| `--profile` | Profile/key name to use |
| `--verbose` | Verbose output |
| `--debug` | Debug output |
| `--help` | Show help |

## Troubleshooting

### Common Issues

**Authentication errors:**
```bash
# Check current authentication
toneclone auth current

# Validate API key
toneclone auth status

# Re-login if needed
toneclone auth login
```

**Connection issues:**
```bash
# Test connectivity
toneclone ping

# Check service health
toneclone health status

# Validate configuration
toneclone config validate
```

**Configuration problems:**
```bash
# Show current config
toneclone config show

# Check config file location
toneclone config path

# Reinitialize config
toneclone config init
```

### Debug Mode

```bash
# Enable debug output
toneclone --debug write "test"

# Verbose mode
toneclone --verbose personas list
```

### Getting Help

```bash
# General help
toneclone --help

# Command-specific help
toneclone write --help
toneclone personas create --help

# List all commands
toneclone --help
```

## Advanced Usage

### Scripting

```bash
#!/bin/bash
# Batch content generation

# Check authentication
if ! toneclone auth current &>/dev/null; then
    echo "Please login: toneclone auth login"
    exit 1
fi

# Generate multiple pieces
toneclone write "Blog post about AI" --output="blog-ai.md"
toneclone write "Social media post" --output="social-ai.txt"
toneclone write "Email newsletter" --output="newsletter.html"

echo "Content generation complete!"
```

### CI/CD Integration

```bash
# Use environment variables for CI/CD
export TONECLONE_API_KEY="$SECRET_API_KEY"

# Generate content in pipeline
toneclone write "Release notes for v1.2.0" --output="release-notes.md"

# Validate configuration
toneclone config validate
```

### Profile Management

```bash
# Create profiles for different use cases
toneclone profiles create --name="Email" --instructions="Professional email format"
toneclone profiles create --name="Blog" --instructions="Engaging blog post style"
toneclone profiles create --name="Documentation" --instructions="Clear technical writing"

# Associate profiles with personas
toneclone profiles associate --profile-id=email-123 --persona="Professional"
toneclone profiles associate --profile-id=blog-456 --persona="Creative"
```

## Support

- **Documentation**: [docs.toneclone.com](https://docs.toneclone.com)
- **GitHub Issues**: [github.com/toneclone/cli/issues](https://github.com/toneclone/cli/issues)
- **Support**: [support@toneclone.com](mailto:support@toneclone.com)

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.