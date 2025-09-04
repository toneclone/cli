# ToneClone CLI Release and Installation Plan

## Project Overview

This document outlines the plan for implementing a comprehensive release and installation system for the ToneClone CLI. The goal is to provide multiple installation methods and automated release processes that match the standards of popular Go CLI tools.

## Current State Analysis

### Existing Infrastructure
- ✅ Makefile with multi-platform build support (`make build-all`)
- ✅ Version management with build-time variables (`Version`, `GitCommit`, `BuildDate`)
- ✅ Basic GitHub repository structure
- ✅ Go module properly configured (`github.com/toneclone/cli`)
- ✅ CLI built with Cobra framework (supports `go install`)

### What We Need to Add
- Automated release system (GoReleaser + GitHub Actions)
- Multiple installation methods (Homebrew, curl script, go install)
- Self-update functionality within the CLI
- Enhanced version management and update checking

## Installation Methods (Target State)

### 1. Go Install (Easiest for Go developers)
```bash
go install github.com/toneclone/cli@latest
go install github.com/toneclone/cli@v1.1.0  # specific version
```

### 2. Homebrew Tap (Best for macOS/Linux users)
```bash
brew tap toneclone/toneclone
brew install toneclone
```

### 3. Curl One-liner (Universal)
```bash
curl -sSL https://install.toneclone.ai | bash
# or with GitHub raw URL as fallback
curl -sSL https://raw.githubusercontent.com/toneclone/cli/main/scripts/install.sh | bash
```

### 4. Direct Binary Download
- GitHub Releases page with binaries for all platforms
- Checksums and signatures for verification

### 5. Self-Update (Built into CLI)
```bash
toneclone update              # update to latest
toneclone update --check      # check for updates
toneclone version --check     # version with update check
```

## Technical Implementation Plan

### Phase 1: Automated Releases (Foundation)
**Goal**: Set up automated release pipeline that generates binaries and releases

**Components**:
1. **GoReleaser Configuration** (`.goreleaser.yml`)
   - Multi-platform builds (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
   - GitHub Releases integration with changelog generation
   - Binary archives (tar.gz for Unix, zip for Windows)
   - Checksums and signing

2. **GitHub Actions Workflow** (`.github/workflows/release.yml`)
   - Trigger on version tags (`v*`)
   - Use GoReleaser Action v6
   - Automated changelog from commits/PRs
   - Release artifact publishing

**Success Criteria**:
- Push `v1.1.0` tag → automated release with binaries
- All platforms build successfully
- GitHub Release created with changelog

### Phase 2: Homebrew Integration
**Goal**: Enable `brew install toneclone` installation method

**Components**:
1. **Homebrew Tap Repository** (`github.com/toneclone/homebrew-toneclone`)
   - Separate repository for Homebrew formulas
   - Automated formula updates via GoReleaser

2. **GoReleaser Homebrew Config**
   - Configure formula generation in `.goreleaser.yml`
   - Automatic formula updates on releases

**Success Criteria**:
- `brew tap toneclone/toneclone && brew install toneclone` works
- Homebrew formula automatically updates on new releases

### Phase 3: Universal Install Script
**Goal**: Provide curl-based installation for all users

**Components**:
1. **Install Script** (`scripts/install.sh`)
   - Detect OS and architecture
   - Download appropriate binary from GitHub Releases
   - Install to `/usr/local/bin` or `$HOME/.local/bin`
   - Handle permissions and PATH setup

2. **Script Hosting Options**:
   - GitHub raw URL (primary)
   - Custom domain (optional): `install.toneclone.ai`

**Success Criteria**:
- Works on Linux, macOS, and WSL
- Handles different architectures (amd64, arm64)
- Graceful error handling and clear messages

### Phase 4: Self-Update System
**Goal**: Built-in update capability within the CLI

**Components**:
1. **Update Command** (`cmd/update.go`)
   - Use `github.com/rhysd/go-github-selfupdate` library
   - Check GitHub Releases API for latest version
   - Download and replace binary in-place
   - Handle different installation methods

2. **Enhanced Version Command** (`cmd/version.go`)
   - Optional update checking (`--check` flag)
   - Show installation method (Homebrew, go install, manual)
   - JSON output for programmatic use

3. **Update Features**:
   - Progress bars during download
   - Confirmation prompts
   - Backup and rollback on failure
   - Respect installation method (warn for Homebrew)

**Success Criteria**:
- `toneclone update` successfully updates binary
- `toneclone version --check` shows update availability
- Works for manually installed binaries
- Gracefully handles Homebrew installations

### Phase 5: Documentation and Polish
**Goal**: Complete user-facing documentation and testing

**Components**:
1. **README Updates**
   - Comprehensive installation instructions
   - Update and maintenance instructions
   - Troubleshooting section

2. **Release Documentation**
   - Maintainer guide for creating releases
   - Testing procedures for release automation
   - Rollback procedures

**Success Criteria**:
- All installation methods documented clearly
- Release process is documented for maintainers
- Troubleshooting covers common scenarios

## Implementation Dependencies

### External Libraries Needed
- `github.com/rhysd/go-github-selfupdate` - Self-update functionality
- GoReleaser tool (CI/CD only, not a Go dependency)

### External Resources Required
- **Homebrew Tap Repository**: `github.com/toneclone/homebrew-toneclone`
- **GitHub Tokens**: For automated releases and Homebrew updates
- **Optional**: Custom domain for install script (`install.toneclone.ai`)

### Security Considerations
- Binary signing and checksum verification
- GitHub token permissions (minimal required scope)
- Secure update mechanism with verification
- Protection against downgrade attacks

## Testing Strategy

### Automated Testing
- GitHub Actions to test release process
- Matrix testing across platforms
- Integration tests for update functionality

### Manual Testing Checklist
- [ ] All installation methods work on clean systems
- [ ] Updates work from each installation method
- [ ] Rollback works if update fails
- [ ] Cross-platform compatibility verified
- [ ] Error messages are clear and helpful

## Risk Assessment

### Low Risk
- Go install method (already works)
- Basic GoReleaser setup
- GitHub Actions automation

### Medium Risk  
- Homebrew tap creation and automation
- Self-update implementation complexity
- Cross-platform install script compatibility

### High Risk
- Self-update security (binary replacement)
- Handling different installation methods in update logic
- Maintaining backward compatibility during updates

## Success Metrics

### User Experience
- Installation time < 30 seconds for any method
- Clear error messages for common issues
- Update process is obvious and safe

### Maintainer Experience
- Release process is fully automated (tag push → release)
- Documentation is comprehensive
- Testing catches issues before release

### Adoption Metrics
- Installation method usage distribution
- Update adoption rates
- Support ticket reduction related to installation

## Next Steps

1. **Start with Phase 1**: Set up GoReleaser and GitHub Actions
2. **Test thoroughly**: Verify automated releases work end-to-end
3. **Iterate quickly**: Get basic automation working before adding complexity
4. **Gather feedback**: Test with real users before finalizing

This plan provides a roadmap for creating a professional-grade release and installation system while managing complexity through phased implementation.