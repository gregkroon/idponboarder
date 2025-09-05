package models

import "time"

type Config struct {
	GitHub   GitHubConfig   `yaml:"github"`
	Harness  HarnessConfig  `yaml:"harness"`
	Defaults DefaultsConfig `yaml:"defaults"`
	Runtime  RuntimeConfig  `yaml:"runtime"`
}

type GitHubConfig struct {
	Organization string `yaml:"organization"`
	AppID        int64  `yaml:"app_id"`
	PrivateKey   string `yaml:"private_key"`
	InstallID    int64  `yaml:"install_id"`
}

type HarnessConfig struct {
	APIKey        string `yaml:"api_key"`
	AccountID     string `yaml:"account_id"`
	BaseURL       string `yaml:"base_url"`
	OrgID         string `yaml:"org_id"`
	ProjectID     string `yaml:"project_id"`
	ConnectorRef  string `yaml:"connector_ref,omitempty"`
}

type DefaultsConfig struct {
	Owner       string            `yaml:"owner"`
	Type        string            `yaml:"type"`
	Lifecycle   string            `yaml:"lifecycle"`
	System      string            `yaml:"system"`
	Tags        map[string]string `yaml:"tags"`
	Annotations map[string]string `yaml:"annotations"`
}

type RuntimeConfig struct {
	Mode          string        `yaml:"mode"`
	Concurrency   int           `yaml:"concurrency"`
	DryRun        bool          `yaml:"dry_run"`
	StateFile     string        `yaml:"state_file"`
	RateLimit     time.Duration `yaml:"rate_limit"`
	LogLevel      string        `yaml:"log_level"`
	IncludeRepos  []string      `yaml:"include_repos"`
	ExcludeRepos  []string      `yaml:"exclude_repos"`
	RequiredFiles []string      `yaml:"required_files"`
}

type Repository struct {
	ID              int64             `json:"id"`
	Name            string            `json:"name"`
	FullName        string            `json:"full_name"`
	Description     string            `json:"description"`
	HTMLURL         string            `json:"html_url"`
	CloneURL        string            `json:"clone_url"`
	Language        string            `json:"language"`
	Topics          []string          `json:"topics"`
	Private         bool              `json:"private"`
	Archived        bool              `json:"archived"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	PushedAt        time.Time         `json:"pushed_at"`
	CodeOwners      []string          `json:"code_owners"`
	HasDockerfile   bool              `json:"has_dockerfile"`
	HasKubernetes   bool              `json:"has_kubernetes"`
	HasCI           bool              `json:"has_ci"`
	DefaultBranch   string            `json:"default_branch"`
	Stars           int               `json:"stars"`
	Forks           int               `json:"forks"`
	OpenIssues      int               `json:"open_issues"`
	License         string            `json:"license"`
	Metadata        map[string]string `json:"metadata"`
}

type CatalogInfo struct {
	APIVersion        string            `yaml:"apiVersion"`
	Identifier        string            `yaml:"identifier"`
	Name              string            `yaml:"name"`
	Kind              string            `yaml:"kind"`
	Type              string            `yaml:"type"`
	ProjectIdentifier string            `yaml:"projectIdentifier"`
	OrgIdentifier     string            `yaml:"orgIdentifier"`
	Owner             string            `yaml:"owner"`
	Metadata          CatalogMetadata   `yaml:"metadata,omitempty"`
	Spec              CatalogSpec       `yaml:"spec"`
}

type CatalogMetadata struct {
	Description string            `yaml:"description,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`
	Links       []ComponentLink   `yaml:"links,omitempty"`
}

type CatalogSpec struct {
	Lifecycle string `yaml:"lifecycle"`
}

type HarnessComponent struct {
	// IDP 2.0 required fields
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Lifecycle   string `json:"lifecycle"`
	Owner       string `json:"owner"`
	
	// Optional fields
	System      string            `json:"system,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Links       []ComponentLink   `json:"links,omitempty"`
	
	// IDP 2.0 metadata structure
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ComponentLink struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Icon  string `json:"icon,omitempty"`
	Type  string `json:"type,omitempty"`
}

type State struct {
	LastRun     time.Time          `json:"last_run"`
	ProcessedRepos map[string]RepoState `json:"processed_repos"`
}

type RepoState struct {
	LastProcessed time.Time `json:"last_processed"`
	LastCommit    string    `json:"last_commit"`
	Status        string    `json:"status"`
	Error         string    `json:"error,omitempty"`
}