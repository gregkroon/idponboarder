# Configuration Equivalence Guide

This tool supports three equivalent ways to configure all options:

1. **Config File (YAML)**
2. **Command Line Flags**
3. **Environment Variables**

## Configuration Priority

Settings are applied in this order (later overrides earlier):
1. Default values
2. Config file (`config.yaml`)
3. Environment variables
4. Command line flags

## Complete Configuration Examples

### 1. Config File (`config.yaml`)

```yaml
github:
  organization: "my-org"
  app_id: 123456
  private_key: "/path/to/private-key.pem"
  install_id: 789012

harness:
  api_key: "your-api-key"
  account_id: "account-123"
  base_url: "https://app.harness.io"
  org_id: "org-456"
  project_id: "project-789"
  connector_ref: "my-connector"

defaults:
  owner: "platform-team"
  type: "service"
  lifecycle: "production"
  system: "core-platform"
  tags:
    team: "platform"
    environment: "prod"
  annotations:
    harness.io/managed: "true"
    custom.annotation: "value"

runtime:
  mode: "yaml"
  concurrency: 10
  dry_run: false
  rate_limit: "200ms"
  log_level: "info"
  include_repos:
    - "repo1"
    - "repo2"
  exclude_repos:
    - "archived-repo"
  required_files:
    - "Dockerfile"
    - "package.json"
```

### 2. Command Line Flags

```bash
./harness-onboarder \
  --org "my-org" \
  --github-app-id "123456" \
  --github-private-key "/path/to/private-key.pem" \
  --github-install-id "789012" \
  --harness-api-key "your-api-key" \
  --harness-account-id "account-123" \
  --harness-base-url "https://app.harness.io" \
  --harness-org-id "org-456" \
  --harness-project-id "project-789" \
  --harness-connector-ref "my-connector" \
  --default-owner "platform-team" \
  --default-type "service" \
  --default-lifecycle "production" \
  --default-system "core-platform" \
  --default-tags "team=platform,environment=prod" \
  --default-annotations "harness.io/managed=true,custom.annotation=value" \
  --mode "yaml" \
  --concurrency 10 \
  --rate-limit "200ms" \
  --log-level "info" \
  --include-repos "repo1,repo2" \
  --exclude-repos "archived-repo" \
  --required-files "Dockerfile,package.json"
```

### 3. Environment Variables

```bash
export HARNESS_ONBOARDER_ORG="my-org"
export HARNESS_ONBOARDER_GITHUB_APP_ID="123456"
export HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY="/path/to/private-key.pem"
export HARNESS_ONBOARDER_GITHUB_INSTALL_ID="789012"
export HARNESS_ONBOARDER_HARNESS_API_KEY="your-api-key"
export HARNESS_ONBOARDER_HARNESS_ACCOUNT_ID="account-123"
export HARNESS_ONBOARDER_HARNESS_BASE_URL="https://app.harness.io"
export HARNESS_ONBOARDER_HARNESS_ORG_ID="org-456"
export HARNESS_ONBOARDER_HARNESS_PROJECT_ID="project-789"
export HARNESS_ONBOARDER_HARNESS_CONNECTOR_REF="my-connector"
export HARNESS_ONBOARDER_DEFAULT_OWNER="platform-team"
export HARNESS_ONBOARDER_DEFAULT_TYPE="service"
export HARNESS_ONBOARDER_DEFAULT_LIFECYCLE="production"
export HARNESS_ONBOARDER_DEFAULT_SYSTEM="core-platform"
export HARNESS_ONBOARDER_DEFAULT_TAGS="team=platform,environment=prod"
export HARNESS_ONBOARDER_DEFAULT_ANNOTATIONS="harness.io/managed=true,custom.annotation=value"
export HARNESS_ONBOARDER_MODE="yaml"
export HARNESS_ONBOARDER_CONCURRENCY="10"
export HARNESS_ONBOARDER_RATE_LIMIT="200ms"
export HARNESS_ONBOARDER_LOG_LEVEL="info"
export HARNESS_ONBOARDER_INCLUDE_REPOS="repo1,repo2"
export HARNESS_ONBOARDER_EXCLUDE_REPOS="archived-repo"
export HARNESS_ONBOARDER_REQUIRED_FILES="Dockerfile,package.json"

./harness-onboarder
```

## Configuration Mapping

| Config File | Command Line Flag | Environment Variable |
|-------------|-------------------|---------------------|
| `github.organization` | `--org` | `HARNESS_ONBOARDER_ORG` |
| `github.app_id` | `--github-app-id` | `HARNESS_ONBOARDER_GITHUB_APP_ID` |
| `github.private_key` | `--github-private-key` | `HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY` |
| `github.install_id` | `--github-install-id` | `HARNESS_ONBOARDER_GITHUB_INSTALL_ID` |
| `harness.api_key` | `--harness-api-key` | `HARNESS_ONBOARDER_HARNESS_API_KEY` |
| `harness.account_id` | `--harness-account-id` | `HARNESS_ONBOARDER_HARNESS_ACCOUNT_ID` |
| `harness.base_url` | `--harness-base-url` | `HARNESS_ONBOARDER_HARNESS_BASE_URL` |
| `harness.org_id` | `--harness-org-id` | `HARNESS_ONBOARDER_HARNESS_ORG_ID` |
| `harness.project_id` | `--harness-project-id` | `HARNESS_ONBOARDER_HARNESS_PROJECT_ID` |
| `harness.connector_ref` | `--harness-connector-ref` | `HARNESS_ONBOARDER_HARNESS_CONNECTOR_REF` |
| `defaults.owner` | `--default-owner` | `HARNESS_ONBOARDER_DEFAULT_OWNER` |
| `defaults.type` | `--default-type` | `HARNESS_ONBOARDER_DEFAULT_TYPE` |
| `defaults.lifecycle` | `--default-lifecycle` | `HARNESS_ONBOARDER_DEFAULT_LIFECYCLE` |
| `defaults.system` | `--default-system` | `HARNESS_ONBOARDER_DEFAULT_SYSTEM` |
| `defaults.tags` | `--default-tags` | `HARNESS_ONBOARDER_DEFAULT_TAGS` |
| `defaults.annotations` | `--default-annotations` | `HARNESS_ONBOARDER_DEFAULT_ANNOTATIONS` |
| `runtime.mode` | `--mode` | `HARNESS_ONBOARDER_MODE` |
| `runtime.concurrency` | `--concurrency` | `HARNESS_ONBOARDER_CONCURRENCY` |
| `runtime.dry_run` | `--dry-run` | `HARNESS_ONBOARDER_DRY_RUN` |
| `runtime.rate_limit` | `--rate-limit` | `HARNESS_ONBOARDER_RATE_LIMIT` |
| `runtime.log_level` | `--log-level` | `HARNESS_ONBOARDER_LOG_LEVEL` |
| `runtime.include_repos` | `--include-repos` | `HARNESS_ONBOARDER_INCLUDE_REPOS` |
| `runtime.exclude_repos` | `--exclude-repos` | `HARNESS_ONBOARDER_EXCLUDE_REPOS` |
| `runtime.required_files` | `--required-files` | `HARNESS_ONBOARDER_REQUIRED_FILES` |

## Special Notes

### Key-Value Pairs (Tags & Annotations)
- **Config file**: Use YAML map syntax
- **Command line**: Use `key=value,key2=value2` format
- **Environment**: Use `key=value,key2=value2` format

### Arrays/Lists
- **Config file**: Use YAML array syntax
- **Command line**: Use comma-separated values
- **Environment**: Use comma-separated values

### Duration Values
- **Config file**: Use string format like `"200ms"` or `"1s"`
- **Command line**: Use duration format like `200ms` or `1s`
- **Environment**: Use string format like `"200ms"` or `"1s"`

### Boolean Values
- **Config file**: Use YAML boolean (`true`/`false`)
- **Command line**: Use flag presence (e.g., `--dry-run` means `true`)
- **Environment**: Use string `"true"` or `"false"`