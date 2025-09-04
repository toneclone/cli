# ToneClone CLI - Maintainer Documentation

This document provides guidance for maintainers of the ToneClone CLI project, including release processes, development workflows, and troubleshooting.

## Table of Contents

- [Release Process](#release-process)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Emergency Procedures](#emergency-procedures)

## Release Process

### Overview

The ToneClone CLI uses automated releases powered by GoReleaser and GitHub Actions. Releases are triggered by pushing version tags to the repository.

### Creating a Release

#### 1. Pre-Release Checklist

- [ ] All changes are merged to `main` branch
- [ ] All tests are passing (`make test` or `go test ./...`)
- [ ] Version numbers are updated if needed
- [ ] CHANGELOG or release notes are prepared (optional - auto-generated)
- [ ] No known critical bugs

#### 2. Version Tagging

We follow [Semantic Versioning](https://semver.org/):

- **Patch releases** (v1.0.1, v1.0.2): Bug fixes, minor improvements
- **Minor releases** (v1.1.0, v1.2.0): New features, backward compatible
- **Major releases** (v2.0.0, v3.0.0): Breaking changes

#### 3. Release Steps

```bash
# 1. Ensure you're on the main branch with latest changes
git checkout main
git pull origin main

# 2. Create and push the version tag
git tag v1.0.1
git push origin v1.0.1
```

#### 4. Automated Process

Once the tag is pushed:

1. **GitHub Actions** triggers the release workflow
2. **GoReleaser** builds binaries for all platforms:
   - Linux (amd64, arm64)
   - macOS (amd64, arm64) 
   - Windows (amd64)
3. **Archives** are created (tar.gz for Unix, zip for Windows)
4. **Checksums** are generated for verification
5. **GitHub Release** is created with changelog and assets

#### 5. Post-Release Verification

- [ ] Check that GitHub Actions workflow completed successfully
- [ ] Verify GitHub Release was created with all expected assets
- [ ] Test download and installation of at least one binary
- [ ] Verify `toneclone --version` shows correct version
- [ ] Check that `go install github.com/toneclone/cli@v1.0.1` works

### Release Artifacts

Each release produces:

```
toneclone_1.0.1_Darwin_x86_64.tar.gz
toneclone_1.0.1_Darwin_ARM64.tar.gz
toneclone_1.0.1_Linux_x86_64.tar.gz
toneclone_1.0.1_Linux_ARM64.tar.gz
toneclone_1.0.1_Windows_x86_64.zip
checksums.txt
```

## Development Workflow

### Local Development

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linting and checks
make check
```

### Testing Releases Locally

```bash
# Install GoReleaser locally (if not already installed)
brew install goreleaser/tap/goreleaser

# Test configuration
goreleaser check

# Test build without releasing (creates snapshot)
goreleaser build --snapshot --clean

# Test full release process locally (no publishing)
goreleaser release --snapshot --skip=publish --clean
```

### Branch Strategy

- **main**: Production-ready code, all releases are tagged from here
- **feature branches**: Development work, merged via PRs
- **hotfix branches**: Critical fixes, can be merged directly and released

## Testing

### Automated Testing

- **Unit Tests**: `go test ./...`
- **Integration Tests**: `go test ./test/...` 
- **Build Tests**: GoReleaser snapshot builds
- **CI Tests**: GitHub Actions runs tests on PRs and releases

### Manual Testing Checklist

#### Before Release
- [ ] CLI builds successfully on all target platforms
- [ ] All commands work as expected
- [ ] Authentication flow works
- [ ] Core functionality (write, personas, profiles, training) works
- [ ] Help text is accurate and complete

#### After Release
- [ ] Binaries download correctly from GitHub Releases
- [ ] Installation via `go install` works
- [ ] Version information is correct
- [ ] Cross-platform compatibility verified

### Testing Matrix

| Platform | Architecture | Status |
|----------|-------------|---------|
| Linux    | amd64       | ✅ Automated |
| Linux    | arm64       | ✅ Automated |
| macOS    | amd64       | ✅ Automated |
| macOS    | arm64       | ✅ Automated |
| Windows  | amd64       | ✅ Automated |

## Troubleshooting

### Common Release Issues

#### GoReleaser Configuration Errors

**Problem**: `goreleaser check` fails
**Solution**: 
1. Validate YAML syntax
2. Check that all referenced files exist
3. Verify build targets are correctly configured

#### GitHub Actions Failures

**Problem**: Release workflow fails
**Solutions**:
- Check GitHub Actions logs for specific error
- Verify `GITHUB_TOKEN` has sufficient permissions
- Ensure all required files are committed
- Check that Go modules are properly tidied

#### Missing Binaries in Release

**Problem**: Some platform binaries are missing
**Solutions**:
- Check GoReleaser build configuration
- Verify platform/architecture combinations
- Review build logs for compilation errors

#### Version Information Incorrect

**Problem**: `toneclone --version` shows wrong information
**Solutions**:
- Verify LDFLAGS in GoReleaser configuration
- Check that version variables are correctly defined in `cmd/version.go`
- Ensure build tags are properly formatted

### Debug Commands

```bash
# Check GoReleaser configuration
goreleaser check

# Test build locally
goreleaser build --snapshot --clean

# View generated archives
ls -la dist/

# Test specific binary
./dist/toneclone_linux_amd64_v1/toneclone --version
```

### GitHub Actions Debugging

```bash
# View workflow runs
gh run list --workflow=release.yml

# View specific run details
gh run view <run-id>

# Download artifacts for local testing
gh run download <run-id>
```

## Emergency Procedures

### Rolling Back a Release

If a release has critical issues:

1. **Immediate**: Mark GitHub Release as "pre-release" to discourage downloads
2. **Create hotfix branch** from the problematic tag
3. **Fix issues** and test thoroughly
4. **Create new patch release** (e.g., v1.0.2) with fixes
5. **Update release notes** explaining the issue and fix

### Deleting a Bad Release

```bash
# Delete GitHub release (use with caution)
gh release delete v1.0.1

# Delete Git tag locally and remotely
git tag -d v1.0.1
git push --delete origin v1.0.1
```

### Security Issues

If a release contains security vulnerabilities:

1. **Do not delete** the release immediately (users may have downloaded it)
2. **Create security advisory** on GitHub
3. **Release patched version** as soon as possible
4. **Update documentation** with migration guidance
5. **Consider coordinated disclosure** timeline

## Contact

For questions about releases or maintainer procedures:

- **GitHub Issues**: Technical problems with releases
- **GitHub Discussions**: General questions about development process
- **Direct Contact**: For security-related issues

## Release History

| Version | Date | Type | Notes |
|---------|------|------|-------|
| v1.0.0  | [Initial] | Major | Initial CLI release |
| v1.0.1  | [TBD] | Patch | First automated release |

---

*This document is maintained by the ToneClone CLI maintainers. Please keep it updated as processes evolve.*