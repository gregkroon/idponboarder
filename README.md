# Harness Onboarder (BETA)

Automatically discover GitHub repositories and onboard them to Harness Internal Developer Portal (IDP).

## Quick Start

```bash
# 1. Build
go build -o harness-onboarder .

# 2. Set environment variables
export HARNESS_ONBOARDER_ORG="your-org"
export HARNESS_ONBOARDER_GITHUB_APP_ID="123456"
export HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY="/path/to/key.pem"
export HARNESS_ONBOARDER_GITHUB_INSTALL_ID="789012"
export HARNESS_ONBOARDER_HARNESS_API_KEY="pat.your-api-key"
export HARNESS_ONBOARDER_HARNESS_ACCOUNT_ID="your-account-id"
export HARNESS_ONBOARDER_HARNESS_ORG_ID="default"
export HARNESS_ONBOARDER_HARNESS_PROJECT_ID="onboarder"
export HARNESS_ONBOARDER_DEFAULT_OWNER="user:account/your.name"

# 3. Run
./harness-onboarder --mode api --include-repos "my-repo"
```

## Configuration

Set these environment variables:

```bash
# GitHub Configuration
export HARNESS_ONBOARDER_ORG="your-github-org"
export HARNESS_ONBOARDER_GITHUB_APP_ID="123456"
export HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY="/path/to/key.pem"
export HARNESS_ONBOARDER_GITHUB_INSTALL_ID="789012"

# Harness Configuration
export HARNESS_ONBOARDER_HARNESS_API_KEY="pat.your-api-key"
export HARNESS_ONBOARDER_HARNESS_ACCOUNT_ID="account-id"
export HARNESS_ONBOARDER_HARNESS_ORG_ID="default"
export HARNESS_ONBOARDER_HARNESS_PROJECT_ID="onboarder"
export HARNESS_ONBOARDER_DEFAULT_OWNER="user:account/your.name"
```

## Workflows

### Workflow 1: YAML → Register (GitOps)

**Step 1: Create PRs with catalog-info.yaml files**

```bash
./harness-onboarder --mode yaml --include-repos "service-a,service-b"
```

**Step 2: After PRs are merged, register the entities**

```bash
./harness-onboarder --mode register --include-repos "service-a,service-b"
```

#### Harness Pipeline for YAML Mode

```yaml
pipeline:
  name: IDP YAML Mode
  identifier: idp_yaml
  projectIdentifier: onboarder
  orgIdentifier: default
  stages:
    - stage:
        name: Create Catalog PRs
        identifier: yaml_stage
        type: CI
        spec:
          cloneCodebase: false
          execution:
            steps:
              - step:
                  type: Run
                  name: Create catalog-info.yaml PRs
                  identifier: create_prs
                  spec:
                    connectorRef: account.dockerhub_connector
                    image: munkys123/harness-onboarder:latest
                    shell: Sh
                    command: |-
                      cd /app
                      echo '<+secrets.getValue("github_private_key")>' > github-key.pem
                      ./harness-onboarder --mode yaml --include-repos "service-a,service-b"
                    envVariables:
                      HARNESS_ONBOARDER_ORG: your-org
                      HARNESS_ONBOARDER_GITHUB_APP_ID: "123456"
                      HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY: /app/github-key.pem
                      HARNESS_ONBOARDER_GITHUB_INSTALL_ID: "789012"
                      HARNESS_ONBOARDER_HARNESS_API_KEY: <+secrets.getValue("harness_api_key")>
                      HARNESS_ONBOARDER_HARNESS_ACCOUNT_ID: <+account.identifier>
                      HARNESS_ONBOARDER_HARNESS_ORG_ID: default
                      HARNESS_ONBOARDER_HARNESS_PROJECT_ID: <+project.identifier>
                      HARNESS_ONBOARDER_DEFAULT_OWNER: user:account/your.name
                    runAsUser: "0"
          platform:
            os: Linux
            arch: Amd64
          runtime:
            type: Cloud
            spec: {}
```

#### Harness Pipeline for Register Mode

```yaml
pipeline:
  name: IDP Register Mode
  identifier: idp_register
  projectIdentifier: onboarder
  orgIdentifier: default
  stages:
    - stage:
        name: Register Entities
        identifier: register_stage
        type: CI
        spec:
          cloneCodebase: false
          execution:
            steps:
              - step:
                  type: Run
                  name: Register catalog-info.yaml files
                  identifier: register_entities
                  spec:
                    connectorRef: account.dockerhub_connector
                    image: munkys123/harness-onboarder:latest
                    shell: Sh
                    command: |-
                      cd /app
                      echo '<+secrets.getValue("github_private_key")>' > github-key.pem
                      ./harness-onboarder --mode register --include-repos "service-a,service-b"
                    envVariables:
                      HARNESS_ONBOARDER_ORG: your-org
                      HARNESS_ONBOARDER_GITHUB_APP_ID: "123456"
                      HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY: /app/github-key.pem
                      HARNESS_ONBOARDER_GITHUB_INSTALL_ID: "789012"
                      HARNESS_ONBOARDER_HARNESS_API_KEY: <+secrets.getValue("harness_api_key")>
                      HARNESS_ONBOARDER_HARNESS_ACCOUNT_ID: <+account.identifier>
                      HARNESS_ONBOARDER_HARNESS_ORG_ID: default
                      HARNESS_ONBOARDER_HARNESS_PROJECT_ID: <+project.identifier>
                      HARNESS_ONBOARDER_DEFAULT_OWNER: user:account/your.name
                    runAsUser: "0"
          platform:
            os: Linux
            arch: Amd64
          runtime:
            type: Cloud
            spec: {}
```

### Workflow 2: Direct API (Fast)

**Single step: Create components directly in Harness IDP**

```bash
./harness-onboarder --mode api --include-repos "service-a,service-b"
```

#### Harness Pipeline for API Mode

```yaml
pipeline:
  name: IDP API Mode
  identifier: idp_api
  projectIdentifier: onboarder
  orgIdentifier: default
  stages:
    - stage:
        name: Direct API Onboarding
        identifier: api_stage
        type: CI
        spec:
          cloneCodebase: false
          execution:
            steps:
              - step:
                  type: Run
                  name: Create components via API
                  identifier: api_onboard
                  spec:
                    connectorRef: account.dockerhub_connector
                    image: munkys123/harness-onboarder:latest
                    shell: Sh
                    command: |-
                      cd /app
                      echo '<+secrets.getValue("github_private_key")>' > github-key.pem
                      ./harness-onboarder --mode api --include-repos "service-a,service-b"
                    envVariables:
                      HARNESS_ONBOARDER_ORG: your-org
                      HARNESS_ONBOARDER_GITHUB_APP_ID: "123456"
                      HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY: /app/github-key.pem
                      HARNESS_ONBOARDER_GITHUB_INSTALL_ID: "789012"
                      HARNESS_ONBOARDER_HARNESS_API_KEY: <+secrets.getValue("harness_api_key")>
                      HARNESS_ONBOARDER_HARNESS_ACCOUNT_ID: <+account.identifier>
                      HARNESS_ONBOARDER_HARNESS_ORG_ID: default
                      HARNESS_ONBOARDER_HARNESS_PROJECT_ID: <+project.identifier>
                      HARNESS_ONBOARDER_DEFAULT_OWNER: user:account/your.name
                    runAsUser: "0"
          platform:
            os: Linux
            arch: Amd64
          runtime:
            type: Cloud
            spec: {}
```

## Common Examples

```bash
# Dry run to preview changes
./harness-onboarder --dry-run

# Process all repositories
./harness-onboarder --mode api

# Exclude archived repositories
./harness-onboarder --exclude-repos "old-service,archived-repo"

# Debug mode
./harness-onboarder --log-level debug
```

## Building Docker Image

### Using the build script (recommended)

```bash
# Build multi-platform image and push to registry
./build.sh harness-onboarder latest

# Build with custom registry
DEFAULT_NS=myregistry ./build.sh harness-onboarder v1.0.0

# Build for specific platforms
PLATFORMS=linux/amd64 ./build.sh harness-onboarder latest
```

### Manual build

```bash
# Single platform build
docker build -t harness-onboarder:latest .

# Multi-platform build
docker buildx build --platform linux/amd64,linux/arm64 \
  -t your-registry/harness-onboarder:latest --push .
```

## Container Usage

```bash
docker run --rm \
  -e HARNESS_ONBOARDER_ORG="your-org" \
  -e HARNESS_ONBOARDER_GITHUB_APP_ID="123456" \
  -e HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY_B64="$(base64 -i key.pem)" \
  -e HARNESS_ONBOARDER_HARNESS_API_KEY="pat.your-key" \
  -e HARNESS_ONBOARDER_HARNESS_ACCOUNT_ID="your-account-id" \
  -e HARNESS_ONBOARDER_HARNESS_PROJECT_ID="onboarder" \
  -e HARNESS_ONBOARDER_DEFAULT_OWNER="user:account/your.name" \
  munkys123/harness-onboarder:latest \
  --mode api --include-repos "my-repo"
```

## Generated Output

Creates IDP 2.0 format `catalog-info.yaml` files:

```yaml
apiVersion: harness.io/v1
identifier: my_service
name: my-service
kind: Component
type: service
projectIdentifier: onboarder
orgIdentifier: default
owner: user:account/your.name
metadata:
  description: Service description
  annotations:
    github.com/project-slug: your-org/my-service
    harness.io/source-repo: https://github.com/your-org/my-service
    harness.io/language: Go
  tags:
    - go
  links:
    - url: https://github.com/your-org/my-service
      title: Repository
      icon: github
spec:
  lifecycle: production
```

## GitHub App Setup

1. **Create GitHub App**: `https://github.com/settings/apps/new`
2. **Permissions**: Contents (Read & Write), Metadata (Read), Pull requests (Read & Write)
3. **Install**: Choose "All repositories" in your organization
4. **Get Values**: App ID, Installation ID (from URL), Private Key (download .pem)

## License

MIT License