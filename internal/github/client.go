package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v50/github"

	"harness-onboarder/internal/errors"
	"harness-onboarder/internal/models"
)

type Client struct {
	client *github.Client
	config models.GitHubConfig
}

func NewClient(config models.GitHubConfig) (*Client, error) {
	var transport *ghinstallation.Transport
	var err error

	if strings.HasPrefix(config.PrivateKey, "/") || strings.Contains(config.PrivateKey, ".pem") {
		transport, err = ghinstallation.NewKeyFromFile(
			http.DefaultTransport,
			config.AppID,
			config.InstallID,
			config.PrivateKey,
		)
	} else {
		privateKeyBytes, parseErr := parsePrivateKeyBytes(config.PrivateKey)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", parseErr)
		}
		transport, err = ghinstallation.New(
			http.DefaultTransport,
			config.AppID,
			config.InstallID,
			privateKeyBytes,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub App transport: %w", err)
	}

	client := github.NewClient(&http.Client{Transport: transport})

	return &Client{
		client: client,
		config: config,
	}, nil
}

func parsePrivateKeyBytes(key string) ([]byte, error) {
	var keyBytes []byte
	var err error

	if strings.HasPrefix(key, "-----BEGIN") {
		keyBytes = []byte(key)
	} else {
		keyBytes, err = base64.StdEncoding.DecodeString(key)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 private key: %w", err)
		}
	}

	return keyBytes, nil
}

func parsePrivateKey(key string) (*rsa.PrivateKey, error) {
	var keyBytes []byte
	var err error

	if strings.HasPrefix(key, "-----BEGIN") {
		keyBytes = []byte(key)
	} else if filepath.Ext(key) != "" {
		keyBytes, err = ioutil.ReadFile(key)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}
	} else {
		keyBytes, err = base64.StdEncoding.DecodeString(key)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 private key: %w", err)
		}
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	var parsedKey interface{}
	if block.Type == "PRIVATE KEY" {
		parsedKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	} else if block.Type == "RSA PRIVATE KEY" {
		parsedKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	} else {
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA private key")
	}

	return privateKey, nil
}

func (c *Client) DiscoverRepositories(ctx context.Context, org string) ([]models.Repository, error) {
	return c.DiscoverRepositoriesWithEnrichment(ctx, org, true)
}

func (c *Client) DiscoverRepositoriesWithEnrichment(ctx context.Context, org string, enrich bool) ([]models.Repository, error) {
	return c.DiscoverRepositoriesWithOptions(ctx, org, enrich, nil)
}

// DiscoverRepositoriesWithOptions discovers repositories with optional filtering for specific repo names
// If specificRepos is provided, it will directly fetch those repositories instead of scanning all repos
func (c *Client) DiscoverRepositoriesWithOptions(ctx context.Context, org string, enrich bool, specificRepos []string) ([]models.Repository, error) {
	var allRepos []models.Repository
	
	// If specific repositories are requested, fetch them directly
	if len(specificRepos) > 0 {
		log.Printf("DEBUG: Directly fetching %d specific repositories for: %s", len(specificRepos), org)
		return c.fetchSpecificRepositories(ctx, org, specificRepos, enrich)
	}
	
	log.Printf("DEBUG: Starting full repository discovery for: %s", org)
	
	// First try to get the user/org to determine if it's a user or organization
	user, _, err := c.client.Users.Get(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("failed to get user/org info: %w", err)
	}
	
	isOrg := user.GetType() == "Organization"
	log.Printf("DEBUG: %s is organization: %v", org, isOrg)
	
	if isOrg {
		// Use organization endpoint
		opts := &github.RepositoryListByOrgOptions{
			Type: "all",
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}

		log.Printf("DEBUG: Fetching organization repositories...")
		for {
			repos, resp, err := c.client.Repositories.ListByOrg(ctx, org, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to list repositories: %w", err)
			}

			log.Printf("DEBUG: Retrieved %d repositories from API", len(repos))
			for _, repo := range repos {
				if repo == nil {
					continue
				}

				var modelRepo models.Repository
				var err error
				
				if enrich {
					log.Printf("DEBUG: Enriching repository: %s", repo.GetFullName())
					modelRepo, err = c.enrichRepository(ctx, repo)
					if err != nil {
						log.Printf("Warning: failed to enrich repository %s: %v", repo.GetFullName(), err)
						continue
					}
					log.Printf("DEBUG: Successfully enriched repository: %s", repo.GetFullName())
				} else {
					// Create minimal repository model without enrichment
					modelRepo = models.Repository{
						ID:            repo.GetID(),
						Name:          repo.GetName(),
						FullName:      repo.GetFullName(),
						Description:   repo.GetDescription(),
						HTMLURL:       repo.GetHTMLURL(),
						CloneURL:      repo.GetCloneURL(),
						Language:      repo.GetLanguage(),
						Topics:        repo.Topics,
						Private:       repo.GetPrivate(),
						Archived:      repo.GetArchived(),
						CreatedAt:     repo.GetCreatedAt().Time,
						UpdatedAt:     repo.GetUpdatedAt().Time,
						PushedAt:      repo.GetPushedAt().Time,
						DefaultBranch: repo.GetDefaultBranch(),
						Stars:         repo.GetStargazersCount(),
						Forks:         repo.GetForksCount(),
						OpenIssues:    repo.GetOpenIssuesCount(),
						Metadata:      make(map[string]string),
					}
					if repo.GetLicense() != nil {
						modelRepo.License = repo.GetLicense().GetName()
					}
				}

				allRepos = append(allRepos, modelRepo)
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	} else {
		// Use GitHub App Installation API for user accounts to access private repos
		opts := &github.ListOptions{
			PerPage: 100,
		}

		for {
			installationRepos, resp, err := c.client.Apps.ListRepos(ctx, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to list repositories: %w", err)
			}

			for _, repo := range installationRepos.Repositories {
				if repo == nil {
					continue
				}

				var modelRepo models.Repository
				var err error
				
				if enrich {
					log.Printf("DEBUG: Enriching repository: %s", repo.GetFullName())
					modelRepo, err = c.enrichRepository(ctx, repo)
					if err != nil {
						log.Printf("Warning: failed to enrich repository %s: %v", repo.GetFullName(), err)
						continue
					}
					log.Printf("DEBUG: Successfully enriched repository: %s", repo.GetFullName())
				} else {
					// Create minimal repository model without enrichment
					modelRepo = models.Repository{
						ID:            repo.GetID(),
						Name:          repo.GetName(),
						FullName:      repo.GetFullName(),
						Description:   repo.GetDescription(),
						HTMLURL:       repo.GetHTMLURL(),
						CloneURL:      repo.GetCloneURL(),
						Language:      repo.GetLanguage(),
						Topics:        repo.Topics,
						Private:       repo.GetPrivate(),
						Archived:      repo.GetArchived(),
						CreatedAt:     repo.GetCreatedAt().Time,
						UpdatedAt:     repo.GetUpdatedAt().Time,
						PushedAt:      repo.GetPushedAt().Time,
						DefaultBranch: repo.GetDefaultBranch(),
						Stars:         repo.GetStargazersCount(),
						Forks:         repo.GetForksCount(),
						OpenIssues:    repo.GetOpenIssuesCount(),
						Metadata:      make(map[string]string),
					}
					if repo.GetLicense() != nil {
						modelRepo.License = repo.GetLicense().GetName()
					}
				}

				allRepos = append(allRepos, modelRepo)
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}

	return allRepos, nil
}

// fetchSpecificRepositories directly fetches specific repositories by name
func (c *Client) fetchSpecificRepositories(ctx context.Context, org string, repoNames []string, enrich bool) ([]models.Repository, error) {
	var allRepos []models.Repository
	
	for _, repoName := range repoNames {
		log.Printf("DEBUG: Fetching repository: %s/%s", org, repoName)
		
		repo, _, err := c.client.Repositories.Get(ctx, org, repoName)
		if err != nil {
			// Categorize the error but don't fail the entire operation
			procErr := errors.CategorizeError(err, fmt.Sprintf("%s/%s", org, repoName))
			log.Printf("Warning: %s", procErr.GetUserFriendlyMessage())
			continue
		}
		
		if repo == nil {
			log.Printf("Warning: repository %s/%s not found", org, repoName)
			continue
		}
		
		var modelRepo models.Repository
		
		if enrich {
			log.Printf("DEBUG: Enriching repository: %s", repo.GetFullName())
			modelRepo, err = c.enrichRepository(ctx, repo)
			if err != nil {
				log.Printf("Warning: failed to enrich repository %s: %v", repo.GetFullName(), err)
				continue
			}
			log.Printf("DEBUG: Successfully enriched repository: %s", repo.GetFullName())
		} else {
			// Create minimal repository model without enrichment
			modelRepo = models.Repository{
				ID:            repo.GetID(),
				Name:          repo.GetName(),
				FullName:      repo.GetFullName(),
				Description:   repo.GetDescription(),
				HTMLURL:       repo.GetHTMLURL(),
				CloneURL:      repo.GetCloneURL(),
				Language:      repo.GetLanguage(),
				Topics:        repo.Topics,
				Private:       repo.GetPrivate(),
				Archived:      repo.GetArchived(),
				CreatedAt:     repo.GetCreatedAt().Time,
				UpdatedAt:     repo.GetUpdatedAt().Time,
				PushedAt:      repo.GetPushedAt().Time,
				DefaultBranch: repo.GetDefaultBranch(),
				Stars:         repo.GetStargazersCount(),
				Forks:         repo.GetForksCount(),
				OpenIssues:    repo.GetOpenIssuesCount(),
				Metadata:      make(map[string]string),
			}
			if repo.GetLicense() != nil {
				modelRepo.License = repo.GetLicense().GetName()
			}
		}
		
		allRepos = append(allRepos, modelRepo)
	}
	
	log.Printf("DEBUG: Successfully fetched %d specific repositories", len(allRepos))
	return allRepos, nil
}

func (c *Client) enrichRepository(ctx context.Context, repo *github.Repository) (models.Repository, error) {
	modelRepo := models.Repository{
		ID:            repo.GetID(),
		Name:          repo.GetName(),
		FullName:      repo.GetFullName(),
		Description:   repo.GetDescription(),
		HTMLURL:       repo.GetHTMLURL(),
		CloneURL:      repo.GetCloneURL(),
		Language:      repo.GetLanguage(),
		Topics:        repo.Topics,
		Private:       repo.GetPrivate(),
		Archived:      repo.GetArchived(),
		CreatedAt:     repo.GetCreatedAt().Time,
		UpdatedAt:     repo.GetUpdatedAt().Time,
		PushedAt:      repo.GetPushedAt().Time,
		DefaultBranch: repo.GetDefaultBranch(),
		Stars:         repo.GetStargazersCount(),
		Forks:         repo.GetForksCount(),
		OpenIssues:    repo.GetOpenIssuesCount(),
		Metadata:      make(map[string]string),
	}

	if repo.GetLicense() != nil {
		modelRepo.License = repo.GetLicense().GetName()
	}

	codeOwners, err := c.getCodeOwners(ctx, repo)
	if err != nil {
		log.Printf("Warning: failed to get CODEOWNERS for %s: %v", repo.GetFullName(), err)
	} else {
		modelRepo.CodeOwners = codeOwners
	}

	signals, err := c.detectRepositorySignals(ctx, repo)
	if err != nil {
		log.Printf("Warning: failed to detect signals for %s: %v", repo.GetFullName(), err)
	} else {
		modelRepo.HasDockerfile = signals.HasDockerfile
		modelRepo.HasKubernetes = signals.HasKubernetes
		modelRepo.HasCI = signals.HasCI
	}

	return modelRepo, nil
}

func (c *Client) getCodeOwners(ctx context.Context, repo *github.Repository) ([]string, error) {
	paths := []string{
		"CODEOWNERS",
		".github/CODEOWNERS",
		"docs/CODEOWNERS",
	}

	for _, path := range paths {
		content, _, resp, err := c.client.Repositories.GetContents(
			ctx,
			repo.GetOwner().GetLogin(),
			repo.GetName(),
			path,
			nil,
		)

		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				continue
			}
			return nil, err
		}

		if content == nil {
			continue
		}

		contentStr, err := content.GetContent()
		if err != nil {
			return nil, err
		}

		return parseCodeOwners(contentStr), nil
	}

	return []string{}, nil
}

func parseCodeOwners(content string) []string {
	var owners []string
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		for i := 1; i < len(parts); i++ {
			owner := strings.TrimPrefix(parts[i], "@")
			if !contains(owners, owner) {
				owners = append(owners, owner)
			}
		}
	}

	return owners
}

type repositorySignals struct {
	HasDockerfile bool
	HasKubernetes bool
	HasCI         bool
}

func (c *Client) detectRepositorySignals(ctx context.Context, repo *github.Repository) (*repositorySignals, error) {
	signals := &repositorySignals{}

	files := []struct {
		path string
		flag *bool
	}{
		{"Dockerfile", &signals.HasDockerfile},
		{"docker-compose.yml", &signals.HasDockerfile},
		{"docker-compose.yaml", &signals.HasDockerfile},
	}

	k8sFiles := []string{
		"k8s/", "kubernetes/", "deploy/", "deployment/",
		"*.yaml", "*.yml",
	}

	ciFiles := []string{
		".github/workflows/", ".gitlab-ci.yml", ".circleci/",
		"Jenkinsfile", ".travis.yml", "azure-pipelines.yml",
		".harness/", "bitbucket-pipelines.yml",
	}

	for _, file := range files {
		exists, err := c.fileExists(ctx, repo, file.path)
		if err != nil {
			log.Printf("Warning: error checking %s in %s: %v", file.path, repo.GetFullName(), err)
			continue
		}
		*file.flag = exists
		if exists {
			break
		}
	}

	signals.HasKubernetes = c.checkPathsExist(ctx, repo, k8sFiles)
	signals.HasCI = c.checkPathsExist(ctx, repo, ciFiles)

	return signals, nil
}

func (c *Client) fileExists(ctx context.Context, repo *github.Repository, path string) (bool, error) {
	_, _, resp, err := c.client.Repositories.GetContents(
		ctx,
		repo.GetOwner().GetLogin(),
		repo.GetName(),
		path,
		nil,
	)

	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (c *Client) checkPathsExist(ctx context.Context, repo *github.Repository, paths []string) bool {
	for _, path := range paths {
		if strings.Contains(path, "*") {
			if c.checkGlobPattern(ctx, repo, path) {
				return true
			}
			continue
		}

		exists, err := c.fileExists(ctx, repo, path)
		if err != nil {
			continue
		}
		if exists {
			return true
		}
	}
	return false
}

func (c *Client) checkGlobPattern(ctx context.Context, repo *github.Repository, pattern string) bool {
	tree, _, err := c.client.Git.GetTree(
		ctx,
		repo.GetOwner().GetLogin(),
		repo.GetName(),
		repo.GetDefaultBranch(),
		true,
	)

	if err != nil {
		return false
	}

	re, err := regexp.Compile(strings.ReplaceAll(pattern, "*", ".*"))
	if err != nil {
		return false
	}

	for _, entry := range tree.Entries {
		if entry.GetPath() != "" && re.MatchString(entry.GetPath()) {
			return true
		}
	}

	return false
}

func (c *Client) CreatePR(ctx context.Context, repo models.Repository, yamlContent string) error {
	owner, repoName, err := parseFullName(repo.FullName)
	if err != nil {
		return err
	}

	branchName := fmt.Sprintf("harness-onboarding-%d", time.Now().Unix())
	
	baseBranch, _, err := c.client.Repositories.GetBranch(ctx, owner, repoName, repo.DefaultBranch, true)
	if err != nil {
		return fmt.Errorf("failed to get base branch: %w", err)
	}

	newRef := &github.Reference{
		Ref: github.String(fmt.Sprintf("refs/heads/%s", branchName)),
		Object: &github.GitObject{
			SHA: baseBranch.Commit.SHA,
		},
	}

	_, _, err = c.client.Git.CreateRef(ctx, owner, repoName, newRef)
	if err != nil {
		// Check if branch already exists (usually indicates existing PR)
		if strings.Contains(strings.ToLower(err.Error()), "reference already exists") {
			return errors.NewPRExistsError(repo.FullName, 0, err)
		}
		return fmt.Errorf("failed to create branch: %w", err)
	}

	catalogPath := "catalog-info.yaml"
	
	// Check if catalog-info.yaml already exists
	existingFile, _, resp, err := c.client.Repositories.GetContents(ctx, owner, repoName, catalogPath, nil)
	var isUpdate bool
	var message string
	var content *github.RepositoryContentFileOptions
	
	if err == nil && existingFile != nil {
		// File exists - check if content is different
		existingContent, err := existingFile.GetContent()
		if err != nil {
			return fmt.Errorf("failed to get existing content: %w", err)
		}
		
		if strings.TrimSpace(existingContent) == strings.TrimSpace(yamlContent) {
			log.Printf("Catalog-info.yaml in %s is already up to date, skipping", repo.FullName)
			return nil
		}
		
		// Content is different - prepare for update
		isUpdate = true
		message = "Update Harness IDP catalog-info.yaml"
		content = &github.RepositoryContentFileOptions{
			Message: &message,
			Content: []byte(yamlContent),
			Branch:  &branchName,
			SHA:     existingFile.SHA, // Required for updates
		}
	} else if resp != nil && resp.StatusCode == 404 {
		// File doesn't exist - prepare for creation
		isUpdate = false
		message = "Add Harness IDP catalog-info.yaml"
		content = &github.RepositoryContentFileOptions{
			Message: &message,
			Content: []byte(yamlContent),
			Branch:  &branchName,
		}
	} else {
		return fmt.Errorf("failed to check existing file: %w", err)
	}

	// Create or update the file
	if isUpdate {
		_, _, err = c.client.Repositories.UpdateFile(ctx, owner, repoName, catalogPath, content)
		if err != nil {
			return fmt.Errorf("failed to update file: %w", err)
		}
	} else {
		_, _, err = c.client.Repositories.CreateFile(ctx, owner, repoName, catalogPath, content)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
	}

	// Set PR title and body based on whether it's an add or update
	var prTitle string
	var prBody string
	
	if isUpdate {
		prTitle = "Update Harness IDP Integration"
		prBody = `This PR updates the catalog-info.yaml file to sync this repository with Harness IDP.

The updated file contains:
- Component metadata
- Owner information  
- Lifecycle and type configuration
- Repository annotations

This ensures the repository information stays current in Harness IDP.

Auto-generated by harness-onboarder tool.`
	} else {
		prTitle = "Add Harness IDP Integration"  
		prBody = `This PR adds a catalog-info.yaml file to integrate this repository with Harness IDP.

The file contains:
- Component metadata
- Owner information
- Lifecycle and type configuration
- Repository annotations

This enables the repository to be discovered and managed through Harness IDP.

Auto-generated by harness-onboarder tool.`
	}

	newPR := &github.NewPullRequest{
		Title: &prTitle,
		Head:  &branchName,
		Base:  &repo.DefaultBranch,
		Body:  &prBody,
	}

	pr, _, err := c.client.PullRequests.Create(ctx, owner, repoName, newPR)
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	log.Printf("Created PR #%d for %s: %s", pr.GetNumber(), repo.FullName, pr.GetHTMLURL())
	return nil
}

func parseFullName(fullName string) (string, string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository full name: %s", fullName)
	}
	return parts[0], parts[1], nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetCatalogInfo retrieves the catalog-info.yaml file content from a repository
func (c *Client) GetCatalogInfo(ctx context.Context, repo models.Repository) (string, error) {
	owner, repoName, err := parseFullName(repo.FullName)
	if err != nil {
		return "", err
	}

	catalogPaths := []string{
		"catalog-info.yaml",
		"catalog-info.yml",
		".harness/catalog-info.yaml", 
		".harness/catalog-info.yml",
	}

	for _, path := range catalogPaths {
		content, _, resp, err := c.client.Repositories.GetContents(
			ctx,
			owner,
			repoName,
			path,
			nil,
		)

		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				continue // Try next path
			}
			return "", fmt.Errorf("error checking %s: %w", path, err)
		}

		if content == nil {
			continue
		}

		contentStr, err := content.GetContent()
		if err != nil {
			return "", fmt.Errorf("error decoding content from %s: %w", path, err)
		}

		log.Printf("Found catalog file in %s at path: %s", repo.FullName, path)
		return contentStr, nil
	}

	return "", fmt.Errorf("no catalog-info.yaml file found in %s", repo.FullName)
}

// CheckForExistingOnboardingPR checks if there are any open PRs related to Harness onboarding
func (c *Client) CheckForExistingOnboardingPR(ctx context.Context, repo models.Repository) (*github.PullRequest, error) {
	owner, repoName, err := parseFullName(repo.FullName)
	if err != nil {
		return nil, err
	}

	// List open pull requests
	opts := &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 50, // Should be enough to find recent onboarding PRs
		},
	}

	prs, _, err := c.client.PullRequests.List(ctx, owner, repoName, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	// Look for PRs that appear to be Harness onboarding related
	for _, pr := range prs {
		if pr == nil {
			continue
		}

		title := strings.ToLower(pr.GetTitle())
		body := strings.ToLower(pr.GetBody())
		
		// Check if PR is related to Harness onboarding
		if isHarnessOnboardingPR(title, body) {
			log.Printf("Found existing Harness onboarding PR #%d: %s", pr.GetNumber(), pr.GetTitle())
			return pr, nil
		}
	}

	return nil, nil
}

// isHarnessOnboardingPR determines if a PR is related to Harness onboarding
func isHarnessOnboardingPR(title, body string) bool {
	harnessKeywords := []string{
		"harness",
		"catalog-info.yaml",
		"catalog-info",
		"idp",
		"harness-onboarder",
		"harness onboarding",
		"add harness",
		"update harness",
	}

	text := title + " " + body
	
	for _, keyword := range harnessKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}

// GetClient returns the underlying GitHub client for direct API access
func (c *Client) GetClient() *github.Client {
	return c.client
}