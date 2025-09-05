# Harness Onboarder

A powerful CLI tool that discovers repositories in GitHub organizations and automatically onboards them into Harness Internal Developer Portal (IDP) using multiple integration methods.

## 🚀 Features

- **Multi-Mode Operation**: YAML (PR generation), API (direct creation), and Register (import existing files)
- **Private Repository Support**: Full GitHub App integration with private repository access
- **Smart Updates**: Detects and updates existing catalog-info.yaml files
- **IDP 2.0 Format**: Native support for Harness IDP 2.0 YAML format
- **Identifier Normalization**: Automatically converts hyphens to underscores in identifiers
- **Concurrent Processing**: High-performance parallel repository processing
- **State Management**: Incremental runs with progress tracking
- **Flexible Filtering**: Include/exclude specific repositories
- **Comprehensive Metadata**: Enriched repository information with CI/CD detection

## 📋 Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [GitHub App Setup](#github-app-setup)
- [Usage Modes](#usage-modes)
  - [YAML Mode](#yaml-mode)
  - [API Mode](#api-mode)
  - [Register Mode](#register-mode)
- [Examples](#examples)
- [Configuration Reference](#configuration-reference)
- [Troubleshooting](#troubleshooting)
- [Advanced Usage](#advanced-usage)

## 📦 Installation

### Prerequisites

- Go 1.19 or later
- GitHub App with appropriate permissions
- Harness account with IDP module enabled
- Valid Harness API token

### Build from Source

```bash
git clone https://github.com/your-org/harness-onboarder.git
cd harness-onboarder
go build .
```

### Binary Release

Download the latest binary from the [releases page](https://github.com/your-org/harness-onboarder/releases).

## ⚙️ Configuration

Create a `config.yaml` file in your project directory:

```yaml
# GitHub Configuration
github-app-id: "1894078"
github-install-id: "84249205" 
github-private-key: "/path/to/your/private-key.pem"
org: "your-github-org"

# Harness Configuration  
harness-api-key: "pat.your-api-key"
harness-account-id: "your-account-id"
harness-org-id: "default"
harness-project-id: "your-project-id"
harness-base-url: "https://app.harness.io"
harness-connector-ref: "account.your-github-connector"

# Default Values for Components
defaults:
  owner: "user:account/your.email"
  type: "service"
  lifecycle: "production"
  system: "platform"
  tags:
    managed-by: "harness-onboarder"
  annotations:
    harness: "true"

# Runtime Configuration
runtime:
  mode: "yaml"
  concurrency: 5
  dry_run: false
  state_file: ".harness-onboarder-state.json"
  rate_limit: "100ms"
  log_level: "info"
  
  # Repository Filtering
  include_repos: []
  exclude_repos:
    - "archived-repo"
    - "template-repo"
  
  # Repository Requirements
  required_files: []
```

## 🔧 GitHub App Setup

### 1. Create GitHub App

1. Go to GitHub Settings → Developer settings → GitHub Apps → New GitHub App
2. Fill in basic information:
   - **App name**: `harness-onboarder`
   - **Description**: `Harness IDP onboarding automation`
   - **Homepage URL**: Your organization URL
   - **Webhook URL**: Leave blank or use placeholder

### 2. Set Permissions

**Repository Permissions:**
- **Contents**: `Write` (for YAML mode PR creation)
- **Metadata**: `Read` (for repository information)
- **Pull requests**: `Write` (for creating PRs)
- **Actions**: `Read` (optional, for CI/CD detection)

**Account Permissions:**
- **Email addresses**: `Read` (optional)

### 3. Install the App

1. Click "Install App" → Select your account/organization
2. Choose **"All repositories"** or select specific repositories
3. Note the Installation ID from the URL

### 4. Generate Private Key

1. In your GitHub App settings, scroll to "Private keys"
2. Click "Generate a private key"
3. Download and save the `.pem` file securely

## 🎯 Usage Modes

### YAML Mode

Creates pull requests with `catalog-info.yaml` files in IDP 2.0 format.

**Use Cases:**
- Initial repository onboarding
- Updating existing catalog files
- GitOps workflow integration
- Team review and approval process

**Features:**
- Smart update detection
- Proper commit messages (Add vs Update)
- Skip unchanged files
- IDP 2.0 format generation

**Example Output:**
```yaml
apiVersion: harness.io/v1
identifier: my_service
name: my-service
kind: Component
type: service
projectIdentifier: myproject
orgIdentifier: default
owner: user:account/john.doe
metadata:
  description: A sample microservice
  annotations:
    github.com/project-slug: myorg/my-service
    harness: "true"
    harness.io/source-repo: https://github.com/myorg/my-service
    harness.io/language: Go
  tags:
    - go
    - microservice
  links:
    - url: https://github.com/myorg/my-service
      title: Repository
      icon: github
      type: repository
spec:
  lifecycle: production
```

### API Mode

Directly creates components in Harness IDP via REST API.

**Use Cases:**
- Automated CI/CD pipelines
- Bulk onboarding
- Immediate component creation
- Integration with other tools

**Features:**
- Direct API integration
- No GitHub write permissions required
- Immediate results
- Duplicate detection

### Register Mode

Imports existing `catalog-info.yaml` files from repositories.

**Use Cases:**
- Importing pre-existing catalog files
- Migrating from other platforms
- Registering manually created files
- Batch imports

**Features:**
- Multi-format support (Backstage + IDP 2.0)
- Automatic identifier sanitization
- Smart content parsing
- Duplicate handling

## 📚 Examples

### Basic Usage

```bash
# Run with default config
./harness-onboarder

# Specify custom config file
./harness-onboarder --config /path/to/config.yaml

# Dry run to see what would be processed
./harness-onboarder --dry-run
```

### Mode-Specific Examples

```bash
# YAML Mode - Create PRs for all repositories
./harness-onboarder --mode yaml

# API Mode - Direct component creation
./harness-onboarder --mode api

# Register Mode - Import existing catalog files
./harness-onboarder --mode register
```

### Repository Filtering

```bash
# Process specific repositories only
./harness-onboarder --include-repos "web-app,api-service,database"

# Exclude specific repositories
./harness-onboarder --exclude-repos "archived-project,template-repo"

# Process single repository
./harness-onboarder --include-repos "my-important-service"
```

### Configuration Overrides

```bash
# Override default owner
./harness-onboarder --default-owner "team:platform"

# Set different component type
./harness-onboarder --default-type "library"

# Change lifecycle
./harness-onboarder --default-lifecycle "experimental"

# Adjust concurrency for large organizations
./harness-onboarder --concurrency 10

# Enable debug logging
./harness-onboarder --log-level debug
```

### Advanced Workflows

#### Complete Onboarding Workflow

```bash
# Step 1: Create catalog-info.yaml files via PRs
./harness-onboarder --mode yaml --include-repos "service-a,service-b,service-c"

# Step 2: After PRs are merged, register the entities
./harness-onboarder --mode register --include-repos "service-a,service-b,service-c"

# Step 3: For immediate updates, use API mode
./harness-onboarder --mode api --include-repos "new-service"
```

#### Bulk Migration

```bash
# Migrate 100 repositories in batches
./harness-onboarder --mode yaml --concurrency 3 --rate-limit 200ms

# Monitor progress
tail -f .harness-onboarder-state.json
```

#### Updating Existing Components

```bash
# Update all components with new metadata
./harness-onboarder --mode yaml --default-system "new-platform"

# The tool will detect changes and create update PRs only where needed
```

#### Real-World Example - ecsbg Repository

This example shows the complete lifecycle using the `ecsbg` repository:

```bash
# Step 1: Create PR with catalog-info.yaml
./harness-onboarder --config config.yaml --mode yaml --include-repos ecsbg

# Output: Created PR #2 for gregkroon/ecsbg: https://github.com/gregkroon/ecsbg/pull/2

# Step 2: After PR is merged, register the entity
./harness-onboarder --config config.yaml --mode register --include-repos ecsbg

# Output: Successfully imported entity for repository: gregkroon/ecsbg

# Step 3: Subsequent register attempts show duplicate detection
./harness-onboarder --config config.yaml --mode register --include-repos ecsbg

# Output: DUPLICATE_FILE_IMPORT: The Requested YAML... has already been imported.
```

## 📖 Configuration Reference

### GitHub Configuration

| Field | Description | Required |
|-------|-------------|----------|
| `github-app-id` | Your GitHub App ID | Yes |
| `github-install-id` | Installation ID for your organization | Yes |
| `github-private-key` | Path to private key PEM file | Yes |
| `org` | GitHub organization or username | Yes |

### Harness Configuration

| Field | Description | Required |
|-------|-------------|----------|
| `harness-api-key` | Harness Platform API token | Yes |
| `harness-account-id` | Harness account identifier | Yes |
| `harness-org-id` | Organization identifier | Yes |
| `harness-project-id` | Project identifier | Yes |
| `harness-base-url` | Harness instance URL | No |
| `harness-connector-ref` | GitHub connector reference | No |

### Default Values

| Field | Description | Default |
|-------|-------------|---------|
| `owner` | Default component owner | Required |
| `type` | Component type | `service` |
| `lifecycle` | Component lifecycle | `production` |
| `system` | System/domain grouping | Empty |
| `tags` | Default tags | Empty |
| `annotations` | Default annotations | Empty |

### Runtime Configuration

| Field | Description | Default |
|-------|-------------|---------|
| `mode` | Operation mode | `yaml` |
| `concurrency` | Parallel processing limit | `5` |
| `dry_run` | Preview mode | `false` |
| `state_file` | State persistence file | `.harness-onboarder-state.json` |
| `rate_limit` | API rate limiting | `100ms` |
| `log_level` | Logging verbosity | `info` |
| `include_repos` | Repositories to include | All |
| `exclude_repos` | Repositories to exclude | None |
| `required_files` | Required files filter | None |

## 🔍 Troubleshooting

### Common Issues

#### 1. GitHub App Permissions

**Error:** `403 Resource not accessible by integration`

**Solution:**
- Verify GitHub App has required permissions (Contents: Write, Pull requests: Write)
- Ensure app is installed with "All repositories" access
- Regenerate and update private key if needed

#### 2. Private Repository Access

**Error:** `Found X repositories, expected more`

**Solution:**
- Check GitHub App installation includes private repositories
- Verify installation ID is correct
- Use installation API instead of user API (already implemented)

#### 3. Harness API Issues

**Error:** `HTTP 404 Not Found` or `HTTP 400 Bad Request`

**Solution:**
- Verify Harness API key has correct permissions
- Check account, org, and project identifiers
- Ensure IDP module is enabled in your Harness account

#### 4. Identifier Validation

**Error:** `Entity identifier = my-service is invalid`

**Solution:**
- Tool automatically converts hyphens to underscores
- Ensure identifiers are alphanumeric with underscores only
- Check for special characters in repository names

### Debug Mode

Enable detailed logging for troubleshooting:

```bash
./harness-onboarder --log-level debug --dry-run
```

### State Management

Reset processing state if needed:

```bash
# Remove state file to reprocess all repositories
rm .harness-onboarder-state.json

# Or backup and restore specific state
cp .harness-onboarder-state.json backup-state.json
```

## 🚀 Advanced Usage

### Custom Templates

Modify the `buildCatalogInfo` function in `internal/cmd/root.go` to customize generated YAML:

```go
// Add custom annotations
annotations["custom.io/team"] = "platform"
annotations["custom.io/tier"] = "1"
```

### Integration with CI/CD

```yaml
# .github/workflows/onboard.yml
name: Onboard to Harness IDP
on:
  push:
    branches: [main]
    paths: ['.harness-onboard']

jobs:
  onboard:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Harness Onboarder
        run: |
          ./harness-onboarder --mode api --include-repos ${{ github.repository }}
        env:
          HARNESS_API_KEY: ${{ secrets.HARNESS_API_KEY }}
```

### Batch Processing

```bash
# Process repositories in alphabetical batches
./harness-onboarder --include-repos "$(echo {a..f}*)"
./harness-onboarder --include-repos "$(echo {g..m}*)"
./harness-onboarder --include-repos "$(echo {n..z}*)"
```

### Monitoring and Metrics

```bash
# Extract metrics from state file
jq '.processed_repos | length' .harness-onboarder-state.json

# Count successful vs failed
jq '[.processed_repos[] | select(.status == "success")] | length' .harness-onboarder-state.json
```

### Environment Variables

All configuration options can be overridden via environment variables with the prefix `HARNESS_ONBOARDER_`:

```bash
export HARNESS_ONBOARDER_ORG="my-org"
export HARNESS_ONBOARDER_GITHUB_APP_ID="123456"
export HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY="/path/to/key.pem"
export HARNESS_ONBOARDER_HARNESS_API_KEY="your-api-key"
```

### Scheduled Execution with Harness CI

```yaml
pipeline:
  name: Repository Onboarding
  identifier: repo_onboarding
  projectIdentifier: your_project
  orgIdentifier: your_org
  stages:
  - stage:
      name: Onboard Repositories
      identifier: onboard_repos
      type: CI
      spec:
        cloneCodebase: false
        execution:
          steps:
          - step:
              type: Run
              name: Run Onboarder
              identifier: run_onboarder
              spec:
                shell: Bash
                command: |
                  # Download and run onboarder
                  wget https://github.com/your-org/harness-onboarder/releases/latest/download/harness-onboarder
                  chmod +x harness-onboarder
                  
                  # Run with environment variables
                  ./harness-onboarder \
                    --org="${GITHUB_ORG}" \
                    --mode=api \
                    --concurrency=3 \
                    --log-level=info
                envVariables:
                  HARNESS_ONBOARDER_GITHUB_APP_ID: <+secrets.getValue("github_app_id")>
                  HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY: <+secrets.getValue("github_private_key")>
                  HARNESS_ONBOARDER_GITHUB_INSTALL_ID: <+secrets.getValue("github_install_id")>
                  HARNESS_ONBOARDER_HARNESS_API_KEY: <+secrets.getValue("harness_api_key")>
                  HARNESS_ONBOARDER_HARNESS_ACCOUNT_ID: <+account.identifier>
  triggers:
  - trigger:
      name: Daily Onboarding
      identifier: daily_onboarding
      type: Scheduled
      spec:
        type: Cron
        spec:
          expression: "0 9 * * *"  # Daily at 9 AM
```

## 🏗️ Architecture

### Core Components

- **Repository Discovery**: GitHub App integration for organization scanning
- **Metadata Enrichment**: Extracts CODEOWNERS, languages, topics, and CI/CD signals
- **Multi-Mode Processing**: YAML (PRs), API (direct), Register (import)
- **State Management**: Tracks processing history and errors
- **Rate Limiting**: Respects GitHub and Harness API limits

### Key Features

1. **Private Repository Support**: Uses GitHub App Installation API for full access
2. **Identifier Normalization**: Converts `my-service` → `my_service` for API compatibility
3. **Smart Updates**: Detects content changes and creates update PRs only when needed
4. **IDP 2.0 Format**: Native Harness format with backward compatibility
5. **Concurrent Processing**: Configurable parallelism with rate limiting
6. **Error Recovery**: Retry mechanisms and comprehensive error reporting

## 🔄 Workflow Patterns

### GitOps Pattern

```bash
# 1. Generate catalog files via PRs
./harness-onboarder --mode yaml --include-repos "team-a-*"

# 2. Team reviews and merges PRs
# 3. Register merged files
./harness-onboarder --mode register --include-repos "team-a-*"
```

### Direct Integration Pattern

```bash
# Direct component creation (no PRs needed)
./harness-onboarder --mode api --include-repos "service-*"
```

### Migration Pattern

```bash
# Import existing Backstage catalog files
./harness-onboarder --mode register

# Update to IDP 2.0 format via PRs
./harness-onboarder --mode yaml
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/your-org/harness-onboarder/issues)
- **Documentation**: [Harness Developer Hub](https://developer.harness.io/docs/internal-developer-portal/)
- **Community**: [Harness Community Slack](https://harnesscommunity.slack.com)

---

**Made with ❤️ for the Harness Developer Community**