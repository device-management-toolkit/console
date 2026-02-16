# Redfish Release Process

This document describes how to create Redfish preview releases using the GitHub Actions workflow.

**Workflow File**: `.github/workflows/release.yml`

Redfish releases are handled by the main release workflow, which automatically detects redfish tags and processes them differently from semantic releases.

**Trigger**: The workflow runs automatically when a tag matching `v*-redfish.preview.*` is pushed to GitHub.

> **Note**: The separate `release-redfish.yml` workflow file is now deprecated. All releases (both semantic and redfish) are handled by the unified `release.yml` workflow.

## Important: Release Validations

The workflow includes **strict validations** to ensure release integrity:

- ✅ **Tag format validation** - Tags must match `v<version>-redfish.preview.<iteration>`
- ✅ **No duplicate releases** - Each version can only be released once
- ✅ **Tag commit verification** - Ensures tag points to the expected commit

These checks prevent accidental overwrites and ensure each release is unique and traceable.

## Version Format

Redfish releases use the following version format:

```bash
v<console-version>-redfish.preview.<iteration>
```

**The phase is strictly `preview` for all Redfish releases.**

Examples:

- `v1.19.1-redfish.preview.1`
- `v1.19.1-redfish.preview.2`
- `v2.0.0-redfish.preview.1`

## Release Process

### 1. Prepare Your Code

Make sure all your changes are merged into the `redfish` branch via a Pull Request:

```bash
# Create a feature branch from redfish
git checkout redfish
git pull origin redfish
git checkout -b feature/my-redfish-changes

# Make your changes
git add .
git commit -m "feat: add redfish support"
git push origin feature/my-redfish-changes

# Create a PR against the redfish branch on GitHub
# After review and approval, merge the PR
```

### 2. Create and Push the Version Tag

Once your changes are merged into the redfish branch, create a git tag with the appropriate version format and push it to trigger the workflow:

```bash
# Switch to redfish branch and pull latest
git checkout redfish
git pull origin redfish

# Create the version tag
git tag v1.19.1-redfish.preview.1

# Push the tag to trigger the workflow
git push origin v1.19.1-redfish.preview.1
```

**That's it!** Pushing the tag automatically triggers the workflow.

### 3. Monitor the Workflow

1. Go to <https://github.com/device-management-toolkit/console/actions>
2. Look for the "Release CI" workflow run
3. Monitor the build progress

The workflow will:

1. Detect this is a Redfish preview release (based on tag pattern)
2. Validate the tag format
3. Check that no release already exists for this tag
4. Build **all** platform binaries (8 total: 4 full + 4 headless):
   - Linux x64 Console (Full + Headless)
   - Linux ARM64 Console (Full + Headless)
   - macOS ARM64 Console (Full + Headless)
   - Windows x64 Console (Full + Headless)
5. Create a GitHub pre-release with all binaries
6. Skip Docker image publishing (only for production releases)
7. Skip OpenAPI/SwaggerHub updates (only for production releases)

### 4. Verify the Release

1. Go to <https://github.com/device-management-toolkit/console/releases>
2. Find your new pre-release (marked as "Pre-release")
3. Verify all 4 binaries are attached
4. Download and test the binaries

## Built Binaries

For Redfish releases, the following binaries are created:

| Platform | Architecture | Type | File Name |
|----------|--------------|------|-----------||
| Linux | x64 | Full | `console_linux_x64` |
| Linux | x64 | Headless | `console_linux_x64_headless` |
| Linux | ARM64 | Full | `console_linux_arm64` |
| Linux | ARM64 | Headless | `console_linux_arm64_headless` |
| Windows | x64 | Full | `console_windows_x64.exe` |
| Windows | x64 | Headless | `console_windows_x64_headless.exe` |
| macOS | ARM64 | Full | `console_mac_arm64` |
| macOS | ARM64 | Headless | `console_mac_arm64_headless` |

Plus `licenses.zip` containing all dependencies' licenses.

## Creating Subsequent Releases

For the next iteration, increment the version properly:

```bash
# Second preview
git tag v1.19.1-redfish.preview.2
git push origin v1.19.1-redfish.preview.2
```

## Troubleshooting

### "Invalid tag format" Error

If you see this error, it means the tag doesn't match the expected pattern.

**Expected format**: `v<version>-redfish.preview.<iteration>`

**Valid examples**:

- `v1.19.1-redfish.preview.1`
- `v1.20.0-redfish.preview.1`

**Invalid examples**:

- `v1.19.1-redfish` (missing phase and iteration)
- `v1.19.1-redfish.alpha.1` (wrong phase - must be 'preview')
- `v1.19.1-redfish.beta.1` (wrong phase - must be 'preview')
- `redfish-v1.19.1` (wrong format)
- `v1.19.1-preview.1` (missing 'redfish' identifier)

**Solution**: Create a tag with the correct format:

```bash
git tag v1.19.1-redfish.preview.1
git push origin v1.19.1-redfish.preview.1
```

### "A release already exists for tag" Error

**This is now a STRICT requirement** - the workflow will fail if a release already exists for the tag.

This prevents:

- Duplicate releases
- Overwriting existing releases
- Version confusion

If you see this error:

```text
❌ Error: A release already exists for tag v1.19.1-redfish.preview.1
```

**Solution**: Use a different version number:

```bash
# Increment to next version
git tag v1.19.1-redfish.preview.2
git push origin v1.19.1-redfish.preview.2
```

**Note**: You cannot re-release the same version. Each release must have a unique version number.

## Differences from Main/Beta Releases

| Feature | Main/Beta | Redfish |
|---------|-----------|---------|
| Workflow File | `release.yml` | `release.yml` (same workflow, different trigger) |
| Trigger | Branch push (main, beta) | Tag push (v*-redfish.preview.*) |
| Versioning | Semantic Release (automatic) | Manual git tags |
| Binaries | 8 binaries (4 full + 4 headless) | 8 binaries (4 full + 4 headless) |
| Release Type | Production release | Pre-release |
| Docker Push | Yes | No |
| OpenAPI/SwaggerHub | Yes | No |

## Notes

- Redfish releases are always marked as **pre-releases** on GitHub
- Docker images are **not** built for Redfish releases
- OpenAPI specs are **not** updated for Redfish releases
- The workflow uses the same build process as production, ensuring consistency
- **Strict validation**: Tags must point to the commit being released - moved tags will cause the workflow to fail
- **Duplicate prevention**: Attempting to re-release an existing version will fail - each version must be unique
- These validations ensure release integrity and prevent accidental overwrites
