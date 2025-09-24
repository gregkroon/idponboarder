package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

	"harness-onboarder/internal/errors"
	"harness-onboarder/internal/github"
	"harness-onboarder/internal/harness"
	"harness-onboarder/internal/models"
)

var (
	cfgFile     string
	config      models.Config
	githubClient *github.Client
	harnessClient *harness.Client
)

var rootCmd = &cobra.Command{
	Use:   "harness-onboarder",
	Short: "Discover GitHub repositories and onboard them to Harness IDP",
	Long: `A CLI utility that discovers repositories in a GitHub organization,
extracts metadata, and onboards them into Harness IDP using:
- YAML mode (PR generation)
- API mode (direct ingestion) 
- Register mode (register existing catalog-info.yaml files)`,
	RunE: runOnboarder,
}

func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	
	rootCmd.Flags().StringP("org", "o", "", "GitHub organization")
	rootCmd.Flags().StringP("mode", "m", "yaml", "Onboarding mode: yaml, api, or register")
	rootCmd.Flags().IntP("concurrency", "c", 5, "Number of concurrent operations")
	rootCmd.Flags().Bool("dry-run", false, "Dry run mode - don't make actual changes")
	rootCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.Flags().StringSlice("include-repos", []string{}, "Specific repositories to include")
	rootCmd.Flags().StringSlice("exclude-repos", []string{}, "Repositories to exclude")
	
	rootCmd.Flags().String("github-app-id", "", "GitHub App ID")
	rootCmd.Flags().String("github-private-key", "", "GitHub App private key file path")
	rootCmd.Flags().String("github-private-key-b64", "", "GitHub App private key (base64 encoded)")
	rootCmd.Flags().String("github-install-id", "", "GitHub App installation ID")
	
	rootCmd.Flags().String("harness-api-key", "", "Harness API key")
	rootCmd.Flags().String("harness-account-id", "", "Harness account ID")
	rootCmd.Flags().String("harness-org-id", "", "Harness organization ID")
	rootCmd.Flags().String("harness-project-id", "", "Harness project ID")
	rootCmd.Flags().String("harness-base-url", "https://app.harness.io", "Harness base URL")
	
	rootCmd.Flags().String("default-owner", "", "Default owner for components")
	rootCmd.Flags().String("default-type", "service", "Default component type")
	rootCmd.Flags().String("default-lifecycle", "production", "Default lifecycle")
	rootCmd.Flags().String("default-system", "", "Default system")
	rootCmd.Flags().StringToString("default-tags", map[string]string{}, "Default tags (key=value pairs)")
	rootCmd.Flags().StringToString("default-annotations", map[string]string{}, "Default annotations (key=value pairs)")

	rootCmd.Flags().String("harness-connector-ref", "", "Harness connector reference")

	rootCmd.Flags().Duration("rate-limit", 100*time.Millisecond, "Rate limit between API calls")
	rootCmd.Flags().StringSlice("required-files", []string{}, "Required files that must exist in repositories")

	viper.BindPFlags(rootCmd.Flags())
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("HARNESS_ONBOARDER")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	// Bind specific environment variables for clarity
	bindEnvVariables()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling config: %v\n", err)
		os.Exit(1)
	}

	setDefaults()
}

func bindEnvVariables() {
	// GitHub configuration
	viper.BindEnv("org", "HARNESS_ONBOARDER_ORG")
	viper.BindEnv("github-app-id", "HARNESS_ONBOARDER_GITHUB_APP_ID")
	viper.BindEnv("github-private-key", "HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY")
	viper.BindEnv("github-private-key-b64", "HARNESS_ONBOARDER_GITHUB_PRIVATE_KEY_B64")
	viper.BindEnv("github-install-id", "HARNESS_ONBOARDER_GITHUB_INSTALL_ID")

	// Harness configuration
	viper.BindEnv("harness-api-key", "HARNESS_ONBOARDER_HARNESS_API_KEY")
	viper.BindEnv("harness-account-id", "HARNESS_ONBOARDER_HARNESS_ACCOUNT_ID")
	viper.BindEnv("harness-org-id", "HARNESS_ONBOARDER_HARNESS_ORG_ID")
	viper.BindEnv("harness-project-id", "HARNESS_ONBOARDER_HARNESS_PROJECT_ID")
	viper.BindEnv("harness-base-url", "HARNESS_ONBOARDER_HARNESS_BASE_URL")
	viper.BindEnv("harness-connector-ref", "HARNESS_ONBOARDER_HARNESS_CONNECTOR_REF")

	// Defaults configuration
	viper.BindEnv("default-owner", "HARNESS_ONBOARDER_DEFAULT_OWNER")
	viper.BindEnv("default-type", "HARNESS_ONBOARDER_DEFAULT_TYPE")
	viper.BindEnv("default-lifecycle", "HARNESS_ONBOARDER_DEFAULT_LIFECYCLE")
	viper.BindEnv("default-system", "HARNESS_ONBOARDER_DEFAULT_SYSTEM")
	viper.BindEnv("default-tags", "HARNESS_ONBOARDER_DEFAULT_TAGS")
	viper.BindEnv("default-annotations", "HARNESS_ONBOARDER_DEFAULT_ANNOTATIONS")

	// Runtime configuration
	viper.BindEnv("mode", "HARNESS_ONBOARDER_MODE")
	viper.BindEnv("concurrency", "HARNESS_ONBOARDER_CONCURRENCY")
	viper.BindEnv("dry-run", "HARNESS_ONBOARDER_DRY_RUN")
	viper.BindEnv("log-level", "HARNESS_ONBOARDER_LOG_LEVEL")
	viper.BindEnv("include-repos", "HARNESS_ONBOARDER_INCLUDE_REPOS")
	viper.BindEnv("exclude-repos", "HARNESS_ONBOARDER_EXCLUDE_REPOS")
	viper.BindEnv("rate-limit", "HARNESS_ONBOARDER_RATE_LIMIT")
	viper.BindEnv("required-files", "HARNESS_ONBOARDER_REQUIRED_FILES")
}

func setDefaults() {
	// Map command line flags to config fields
	if viper.IsSet("github-app-id") {
		if appIDStr := viper.GetString("github-app-id"); appIDStr != "" {
			if appID, err := strconv.ParseInt(appIDStr, 10, 64); err == nil {
				config.GitHub.AppID = appID
			}
		}
	}
	if viper.IsSet("github-install-id") {
		if installIDStr := viper.GetString("github-install-id"); installIDStr != "" {
			if installID, err := strconv.ParseInt(installIDStr, 10, 64); err == nil {
				config.GitHub.InstallID = installID
			}
		}
	}
	if viper.IsSet("github-private-key") {
		config.GitHub.PrivateKey = viper.GetString("github-private-key")
	}

	// Handle base64-encoded private key for container deployments
	if viper.IsSet("github-private-key-b64") {
		keyB64 := viper.GetString("github-private-key-b64")
		if keyB64 != "" {
			keyBytes, err := base64.StdEncoding.DecodeString(keyB64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error decoding base64 private key: %v\n", err)
				os.Exit(1)
			}

			// Create temporary file for the decoded key
			tmpFile, err := os.CreateTemp("", "github-key-*.pem")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating temporary key file: %v\n", err)
				os.Exit(1)
			}

			// Write decoded key to temporary file
			if _, err := tmpFile.Write(keyBytes); err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				fmt.Fprintf(os.Stderr, "Error writing temporary key file: %v\n", err)
				os.Exit(1)
			}

			tmpFile.Close()

			// Set file permissions to 600 for security
			if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
				os.Remove(tmpFile.Name())
				fmt.Fprintf(os.Stderr, "Error setting key file permissions: %v\n", err)
				os.Exit(1)
			}

			config.GitHub.PrivateKey = tmpFile.Name()
		}
	}
	if viper.IsSet("org") {
		config.GitHub.Organization = viper.GetString("org")
	}

	// Map other command line flags
	if viper.IsSet("harness-api-key") {
		config.Harness.APIKey = viper.GetString("harness-api-key")
	}
	if viper.IsSet("harness-account-id") {
		config.Harness.AccountID = viper.GetString("harness-account-id")
	}
	if viper.IsSet("harness-org-id") {
		config.Harness.OrgID = viper.GetString("harness-org-id")
	}
	if viper.IsSet("harness-project-id") {
		config.Harness.ProjectID = viper.GetString("harness-project-id")
	}
	if viper.IsSet("harness-base-url") {
		config.Harness.BaseURL = viper.GetString("harness-base-url")
	}
	if viper.IsSet("harness-connector-ref") {
		config.Harness.ConnectorRef = viper.GetString("harness-connector-ref")
	}

	if viper.IsSet("default-owner") {
		config.Defaults.Owner = viper.GetString("default-owner")
	}
	if viper.IsSet("default-type") {
		config.Defaults.Type = viper.GetString("default-type")
	}
	if viper.IsSet("default-lifecycle") {
		config.Defaults.Lifecycle = viper.GetString("default-lifecycle")
	}
	if viper.IsSet("default-system") {
		config.Defaults.System = viper.GetString("default-system")
	}
	if viper.IsSet("default-tags") {
		config.Defaults.Tags = viper.GetStringMapString("default-tags")
	}
	if viper.IsSet("default-annotations") {
		config.Defaults.Annotations = viper.GetStringMapString("default-annotations")
	}

	if viper.IsSet("mode") {
		config.Runtime.Mode = viper.GetString("mode")
	}
	if viper.IsSet("concurrency") {
		config.Runtime.Concurrency = viper.GetInt("concurrency")
	}
	if viper.IsSet("dry-run") {
		config.Runtime.DryRun = viper.GetBool("dry-run")
	}
	if viper.IsSet("log-level") {
		config.Runtime.LogLevel = viper.GetString("log-level")
	}
	if viper.IsSet("include-repos") {
		config.Runtime.IncludeRepos = viper.GetStringSlice("include-repos")
	}
	if viper.IsSet("exclude-repos") {
		config.Runtime.ExcludeRepos = viper.GetStringSlice("exclude-repos")
	}
	if viper.IsSet("rate-limit") {
		config.Runtime.RateLimit = viper.GetDuration("rate-limit")
	}
	if viper.IsSet("required-files") {
		config.Runtime.RequiredFiles = viper.GetStringSlice("required-files")
	}

	// Set defaults for unset values
	if config.Runtime.Concurrency == 0 {
		config.Runtime.Concurrency = 5
	}
	if config.Runtime.RateLimit == 0 {
		config.Runtime.RateLimit = time.Millisecond * 100
	}
	if config.Runtime.LogLevel == "" {
		config.Runtime.LogLevel = "info"
	}
	if config.Runtime.Mode == "" {
		config.Runtime.Mode = "yaml"
	}
	if config.Defaults.Type == "" {
		config.Defaults.Type = "service"
	}
	if config.Defaults.Lifecycle == "" {
		config.Defaults.Lifecycle = "production"
	}
	if config.Harness.BaseURL == "" {
		config.Harness.BaseURL = "https://app.harness.io"
	}
}

func runOnboarder(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	if err := validateConfig(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if config.Runtime.DryRun {
		log.Println("Running in dry-run mode - no changes will be made")
	}

	var err error
	githubClient, err = github.NewClient(config.GitHub)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	harnessClient, err = harness.NewClient(config.Harness)
	if err != nil {
		return fmt.Errorf("failed to create Harness client: %w", err)
	}


	log.Printf("Starting onboarding process for organization: %s", config.GitHub.Organization)
	log.Printf("Mode: %s, Concurrency: %d, Dry Run: %t", 
		config.Runtime.Mode, config.Runtime.Concurrency, config.Runtime.DryRun)

	// Skip enrichment for register and api modes since we only need basic repo info
	// Only yaml mode needs full enrichment for PR creation
	enrich := config.Runtime.Mode == "yaml"
	
	// Use optimized discovery when specific repositories are requested
	var repos []models.Repository
	if len(config.Runtime.IncludeRepos) > 0 {
		log.Printf("Using optimized discovery for %d specific repositories", len(config.Runtime.IncludeRepos))
		repos, err = githubClient.DiscoverRepositoriesWithOptions(ctx, config.GitHub.Organization, enrich, config.Runtime.IncludeRepos)
	} else {
		repos, err = githubClient.DiscoverRepositoriesWithEnrichment(ctx, config.GitHub.Organization, enrich)
	}
	if err != nil {
		return fmt.Errorf("failed to discover repositories: %w", err)
	}

	// Apply filtering - when using optimized discovery, most filtering is already done
	filteredRepos := filterRepositories(repos, len(config.Runtime.IncludeRepos) > 0)
	log.Printf("Found %d repositories, %d after filtering", len(repos), len(filteredRepos))

	if config.Runtime.DryRun {
		log.Printf("Would process %d repositories:", len(filteredRepos))
		for _, repo := range filteredRepos {
			log.Printf("  - %s", repo.FullName)
		}
		return nil
	}

	switch config.Runtime.Mode {
	case "yaml":
		return processYAMLMode(ctx, filteredRepos)
	case "api":
		return processAPIMode(ctx, filteredRepos)
	case "register":
		log.Printf("DEBUG: About to process %d filtered repositories in register mode", len(filteredRepos))
		return processRegisterMode(ctx, filteredRepos)
	default:
		return fmt.Errorf("unsupported mode: %s (supported: yaml, api, register)", config.Runtime.Mode)
	}
}

func validateConfig() error {
	if config.GitHub.Organization == "" {
		return fmt.Errorf("GitHub organization is required")
	}
	if config.GitHub.AppID == 0 {
		return fmt.Errorf("GitHub App ID is required")
	}
	if config.GitHub.PrivateKey == "" {
		return fmt.Errorf("GitHub private key is required")
	}
	if config.GitHub.InstallID == 0 {
		return fmt.Errorf("GitHub installation ID is required")
	}
	
	if config.Harness.APIKey == "" {
		return fmt.Errorf("Harness API key is required")
	}
	if config.Harness.AccountID == "" {
		return fmt.Errorf("Harness account ID is required")
	}
	if config.Harness.OrgID == "" {
		return fmt.Errorf("Harness organization ID is required")
	}
	if config.Harness.ProjectID == "" {
		return fmt.Errorf("Harness project ID is required")
	}
	
	if config.Defaults.Owner == "" {
		return fmt.Errorf("default owner is required")
	}
	
	return nil
}

func filterRepositories(repos []models.Repository, optimizedDiscovery bool) []models.Repository {
	var filtered []models.Repository
	
	// If we used optimized discovery, we already have the specific repos we want
	// Only need to check for archived repos and exclude list
	if optimizedDiscovery {
		excludeMap := make(map[string]bool)
		for _, repo := range config.Runtime.ExcludeRepos {
			excludeMap[repo] = true
		}
		
		for _, repo := range repos {
			if repo.Archived {
				continue
			}
			
			if excludeMap[repo.Name] {
				continue
			}
			
			filtered = append(filtered, repo)
		}
		
		return filtered
	}
	
	// Original filtering logic for full discovery
	includeMap := make(map[string]bool)
	for _, repo := range config.Runtime.IncludeRepos {
		includeMap[repo] = true
	}
	
	excludeMap := make(map[string]bool)
	for _, repo := range config.Runtime.ExcludeRepos {
		excludeMap[repo] = true
	}
	
	for _, repo := range repos {
		if repo.Archived {
			continue
		}
		
		if len(includeMap) > 0 && !includeMap[repo.Name] {
			continue
		}
		
		if excludeMap[repo.Name] {
			continue
		}
		
		filtered = append(filtered, repo)
	}
	
	return filtered
}

func processYAMLMode(ctx context.Context, repos []models.Repository) error {
	log.Printf("Processing %d repositories in YAML mode", len(repos))
	
	semaphore := make(chan struct{}, config.Runtime.Concurrency)
	results := make(chan errors.ProcessingResult, len(repos))
	
	for _, repo := range repos {
		go func(r models.Repository) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			time.Sleep(config.Runtime.RateLimit)
			result := processRepositoryYAMLWithResult(ctx, r)
			results <- result
		}(repo)
	}
	
	// Collect results and build summary
	summary := errors.NewErrorSummary()
	for i := 0; i < len(repos); i++ {
		result := <-results
		summary.AddResult(result)
	}
	
	// Print detailed summary
	summary.PrintSummary()
	
	if summary.Total > 0 {
		return fmt.Errorf("encountered %d errors during YAML processing", summary.Total)
	}
	
	return nil
}

func processAPIMode(ctx context.Context, repos []models.Repository) error {
	log.Printf("Processing %d repositories in API mode", len(repos))
	
	semaphore := make(chan struct{}, config.Runtime.Concurrency)
	results := make(chan errors.ProcessingResult, len(repos))
	
	for _, repo := range repos {
		go func(r models.Repository) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			time.Sleep(config.Runtime.RateLimit)
			result := processRepositoryAPIWithResult(ctx, r)
			results <- result
		}(repo)
	}
	
	// Collect results and build summary
	summary := errors.NewErrorSummary()
	for i := 0; i < len(repos); i++ {
		result := <-results
		summary.AddResult(result)
	}
	
	// Print detailed summary
	summary.PrintSummary()
	
	if summary.Total > 0 {
		return fmt.Errorf("encountered %d errors during API processing", summary.Total)
	}
	
	return nil
}

func processRepositoryYAML(ctx context.Context, repo models.Repository) error {
	result := processRepositoryYAMLWithResult(ctx, repo)
	return result.Error
}

func processRepositoryYAMLWithResult(ctx context.Context, repo models.Repository) errors.ProcessingResult {
	log.Printf("Processing repository %s in YAML mode", repo.FullName)
	
	// First check if there are any existing open PRs for Harness onboarding
	log.Printf("DEBUG: Checking for existing open Harness onboarding PRs in %s", repo.FullName)
	existingPR, err := githubClient.CheckForExistingOnboardingPR(ctx, repo)
	if err != nil {
		log.Printf("DEBUG: Error checking for existing PRs in %s: %v", repo.FullName, err)
	}
	if existingPR != nil {
		log.Printf("Repository %s already has an open Harness onboarding PR #%d", repo.FullName, existingPR.GetNumber())
		return errors.ProcessingResult{
			Repository: repo.FullName,
			Success:    true,
			Error:      nil,
			Message:    fmt.Sprintf("Open PR #%d already exists (%s)", existingPR.GetNumber(), existingPR.GetTitle()),
			Skipped:    true,
			Action:     "skipped",
		}
	}
	
	// Check if catalog-info.yaml already exists in the repository
	log.Printf("DEBUG: Checking for existing catalog-info.yaml in %s", repo.FullName)
	existingCatalog, err := githubClient.GetCatalogInfo(ctx, repo)
	if err != nil {
		log.Printf("DEBUG: No existing catalog file found in %s: %v", repo.FullName, err)
	}
	if err == nil && existingCatalog != "" {
		log.Printf("Repository %s already has catalog-info.yaml file", repo.FullName)
		
		// Check if the component is already registered in Harness IDP
		catalogInfo := buildCatalogInfo(repo)
		component, err := harnessClient.GetComponent(ctx, catalogInfo.Identifier)
		if err == nil && component != nil {
			log.Printf("Component %s already exists in Harness IDP and has catalog-info.yaml file", catalogInfo.Identifier)
			return errors.ProcessingResult{
				Repository: repo.FullName,
				Success:    true,
				Error:      nil,
				Message:    "Already onboarded (file exists in repo, component exists in IDP)",
				Skipped:    true,
				Action:     "skipped",
			}
		} else {
			log.Printf("Catalog file exists but component not found in IDP - may need registration")
			return errors.ProcessingResult{
				Repository: repo.FullName,
				Success:    true,
				Error:      nil,
				Message:    "Catalog file exists, but component not in IDP (use register mode)",
				Skipped:    true,
				Action:     "skipped",
			}
		}
	}
	
	// Generate the catalog info and YAML content
	catalogInfo := buildCatalogInfo(repo)
	yamlContent, err := yaml.Marshal(catalogInfo)
	if err != nil {
		procErr := &errors.ProcessingError{
			Category:     errors.ErrorCategoryValidation,
			Type:         errors.ErrorTypeCatalogFileInvalid,
			Message:      fmt.Sprintf("failed to marshal catalog-info.yaml: %s", err.Error()),
			Repository:   repo.FullName,
			Cause:        err,
			Recoverable:  false,
			UserFriendly: fmt.Sprintf("Failed to generate catalog-info.yaml for '%s'. This might be due to invalid repository metadata.", repo.FullName),
		}
		return errors.ProcessingResult{
			Repository: repo.FullName,
			Success:    false,
			Error:      procErr,
			Message:    "YAML generation failed",
			Action:     "failed",
		}
	}
	
	err = githubClient.CreatePR(ctx, repo, string(yamlContent))
	if err != nil {
		procErr := errors.CategorizeError(err, repo.FullName)
		
		// Handle specific PR-related scenarios
		if procErr.Type == errors.ErrorTypePRExists {
			return errors.ProcessingResult{
				Repository: repo.FullName,
				Success:    false,
				Error:      procErr,
				Message:    "PR already exists",
				Skipped:    true,
				Action:     "skipped",
			}
		}
		
		return errors.ProcessingResult{
			Repository: repo.FullName,
			Success:    false,
			Error:      procErr,
			Message:    "PR creation failed",
			Action:     "failed",
		}
	}
	
	log.Printf("Successfully created PR for repository: %s", repo.FullName)
	return errors.ProcessingResult{
		Repository: repo.FullName,
		Success:    true,
		Error:      nil,
		Message:    "PR created successfully",
		Action:     "created",
	}
}

func processRepositoryAPI(ctx context.Context, repo models.Repository) error {
	result := processRepositoryAPIWithResult(ctx, repo)
	return result.Error
}

func processRepositoryAPIWithResult(ctx context.Context, repo models.Repository) errors.ProcessingResult {
	log.Printf("Processing repository %s in API mode", repo.FullName)
	
	component := buildHarnessComponent(repo)
	
	err := harnessClient.CreateComponent(ctx, component)
	if err != nil {
		procErr := errors.CategorizeError(err, repo.FullName)
		
		// Handle specific entity-related scenarios
		if procErr.Type == errors.ErrorTypeEntityExists {
			return errors.ProcessingResult{
				Repository: repo.FullName,
				Success:    false,
				Error:      procErr,
				Message:    "Component already exists",
				Skipped:    true,
				Action:     "skipped",
			}
		}
		
		return errors.ProcessingResult{
			Repository: repo.FullName,
			Success:    false,
			Error:      procErr,
			Message:    "Component creation failed",
			Action:     "failed",
		}
	}
	
	log.Printf("Successfully created component for repository: %s", repo.FullName)
	return errors.ProcessingResult{
		Repository: repo.FullName,
		Success:    true,
		Error:      nil,
		Message:    "Component created successfully",
		Action:     "created",
	}
}

func processRegisterMode(ctx context.Context, repos []models.Repository) error {
	log.Printf("Processing %d repositories in REGISTER mode", len(repos))
	
	semaphore := make(chan struct{}, config.Runtime.Concurrency)
	results := make(chan errors.ProcessingResult, len(repos))
	
	for _, repo := range repos {
		go func(r models.Repository) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			time.Sleep(config.Runtime.RateLimit)
			result := processRepositoryRegisterWithResult(ctx, r)
			results <- result
		}(repo)
	}
	
	// Collect results and build summary
	summary := errors.NewErrorSummary()
	for i := 0; i < len(repos); i++ {
		result := <-results
		summary.AddResult(result)
	}
	
	// Print detailed summary
	summary.PrintSummary()
	
	if summary.Total > 0 {
		return fmt.Errorf("encountered %d errors during REGISTER processing", summary.Total)
	}
	
	return nil
}

func processRepositoryRegister(ctx context.Context, repo models.Repository) error {
	result := processRepositoryRegisterWithResult(ctx, repo)
	return result.Error
}

func processRepositoryRegisterWithResult(ctx context.Context, repo models.Repository) errors.ProcessingResult {
	log.Printf("Processing repository %s in REGISTER mode", repo.FullName)
	
	// Check if catalog-info.yaml exists in the repository and get the path and content
	catalogPath, catalogContent, err := getCatalogInfoPathAndContent(ctx, repo)
	if err != nil {
		// Missing catalog files are expected - skip gracefully
		log.Printf("Skipping %s: %v", repo.FullName, err)
		return errors.ProcessingResult{
			Repository: repo.FullName,
			Success:    true,
			Error:      nil,
			Message:    "No catalog-info.yaml found",
			Skipped:    true,
			Action:     "skipped",
		}
	}
	
	log.Printf("Registering repository for entity import: %s (branch: %s, file: %s)", repo.FullName, repo.DefaultBranch, catalogPath)
	
	// Sanitize the catalog content to ensure identifiers don't have hyphens
	sanitizedContent := sanitizeYAMLIdentifiers(catalogContent)
	
	// Register the repository for entity import with Harness IDP
	err = harnessClient.RegisterCatalogLocation(ctx, repo.FullName, repo.DefaultBranch, catalogPath, sanitizedContent)
	if err != nil {
		procErr := errors.CategorizeError(err, repo.FullName)
		
		// Handle specific registration scenarios
		if procErr.Type == errors.ErrorTypeEntityAlreadyRegistered {
			return errors.ProcessingResult{
				Repository: repo.FullName,
				Success:    false,
				Error:      procErr,
				Message:    "Entity already registered",
				Skipped:    true,
				Action:     "skipped",
			}
		}
		
		return errors.ProcessingResult{
			Repository: repo.FullName,
			Success:    false,
			Error:      procErr,
			Message:    "Registration failed",
			Action:     "failed",
		}
	}
	
	log.Printf("Successfully registered entity for repository: %s", repo.FullName)
	return errors.ProcessingResult{
		Repository: repo.FullName,
		Success:    true,
		Error:      nil,
		Message:    "Entity registered successfully",
		Action:     "registered",
	}
}

// getCatalogInfoPath checks if catalog-info.yaml exists and returns the path
func getCatalogInfoPath(ctx context.Context, repo models.Repository) (string, error) {
	catalogPaths := []string{
		"catalog-info.yaml",
		"catalog-info.yml",
		".harness/catalog-info.yaml", 
		".harness/catalog-info.yml",
	}
	
	owner := strings.Split(repo.FullName, "/")[0]
	repoName := strings.Split(repo.FullName, "/")[1]

	for _, path := range catalogPaths {
		_, _, resp, err := githubClient.GetClient().Repositories.GetContents(
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

		log.Printf("Found catalog file in %s at path: %s", repo.FullName, path)
		return path, nil
	}

	return "", fmt.Errorf("no catalog-info.yaml file found in %s", repo.FullName)
}

// getCatalogInfoPathAndContent checks if catalog-info.yaml exists and returns both the path and content
func getCatalogInfoPathAndContent(ctx context.Context, repo models.Repository) (string, string, error) {
	catalogPaths := []string{
		"catalog-info.yaml",
		"catalog-info.yml",
		".harness/catalog-info.yaml", 
		".harness/catalog-info.yml",
	}
	
	owner := strings.Split(repo.FullName, "/")[0]
	repoName := strings.Split(repo.FullName, "/")[1]

	for _, path := range catalogPaths {
		content, _, resp, err := githubClient.GetClient().Repositories.GetContents(
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
			return "", "", fmt.Errorf("error checking %s: %w", path, err)
		}

		if content == nil {
			continue
		}

		contentStr, err := content.GetContent()
		if err != nil {
			return "", "", fmt.Errorf("error decoding content from %s: %w", path, err)
		}

		log.Printf("Found catalog file in %s at path: %s", repo.FullName, path)
		return path, contentStr, nil
	}

	return "", "", fmt.Errorf("no catalog-info.yaml file found in %s", repo.FullName)
}

// sanitizeYAMLIdentifiers replaces hyphens with underscores in YAML identifier fields
// This ensures compatibility with Harness IDP API requirements
func sanitizeYAMLIdentifiers(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	for i, line := range lines {
		// Look for identifier field and replace hyphens with underscores in the value
		if strings.HasPrefix(strings.TrimSpace(line), "identifier:") {
			// Split on ":" to separate field and value
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				fieldPart := parts[0]
				valuePart := strings.TrimSpace(parts[1])
				// Replace hyphens with underscores in the identifier value
				sanitizedValue := strings.ReplaceAll(valuePart, "-", "_")
				lines[i] = fieldPart + ": " + sanitizedValue
			}
		}
	}
	return strings.Join(lines, "\n")
}

func buildCatalogInfo(repo models.Repository) models.CatalogInfo {
	name := sanitizeName(repo.Name)
	// Normalize identifier by replacing hyphens with underscores
	identifier := strings.ReplaceAll(name, "-", "_")
	
	annotations := make(map[string]string)
	for k, v := range config.Defaults.Annotations {
		// Transform hyphenated annotation keys back to dot notation
		if k == "harness-io-managed" {
			annotations["harness.io/managed"] = v
		} else {
			annotations[k] = v
		}
	}
	annotations["github.com/project-slug"] = repo.FullName
	annotations["harness.io/source-repo"] = repo.HTMLURL
	
	if repo.Language != "" {
		annotations["harness.io/language"] = repo.Language
	}
	
	tags := repo.Topics
	if repo.Language != "" && !contains(tags, strings.ToLower(repo.Language)) {
		tags = append(tags, strings.ToLower(repo.Language))
	}
	
	// Build links for IDP 2.0 format
	links := []models.ComponentLink{
		{
			URL:   repo.HTMLURL,
			Title: "Repository",
			Icon:  "github",
			Type:  "repository",
		},
	}
	
	return models.CatalogInfo{
		APIVersion:        "harness.io/v1",
		Identifier:        identifier,
		Name:              repo.Name,
		Kind:              "Component",
		Type:              config.Defaults.Type,
		ProjectIdentifier: config.Harness.ProjectID,
		OrgIdentifier:     config.Harness.OrgID,
		Owner:             getOwner(repo),
		Metadata: models.CatalogMetadata{
			Description: repo.Description,
			Tags:        tags,
			Annotations: annotations,
			Links:       links,
		},
		Spec: models.CatalogSpec{
			Lifecycle: config.Defaults.Lifecycle,
		},
	}
}

func buildHarnessComponent(repo models.Repository) models.HarnessComponent {
	name := sanitizeName(repo.Name)
	// Normalize identifier by replacing hyphens with underscores
	identifier := strings.ReplaceAll(name, "-", "_")
	
	annotations := make(map[string]string)
	for k, v := range config.Defaults.Annotations {
		// Transform hyphenated annotation keys back to dot notation
		if k == "harness-io-managed" {
			annotations["harness.io/managed"] = v
		} else {
			annotations[k] = v
		}
	}
	annotations["github.com/project-slug"] = repo.FullName
	annotations["harness.io/source-repo"] = repo.HTMLURL
	
	if repo.Language != "" {
		annotations["harness.io/language"] = repo.Language
	}
	
	tags := repo.Topics
	if repo.Language != "" && !contains(tags, strings.ToLower(repo.Language)) {
		tags = append(tags, strings.ToLower(repo.Language))
	}
	
	links := []models.ComponentLink{
		{
			URL:   repo.HTMLURL,
			Title: "Repository",
			Icon:  "github",
		},
	}
	
	metadata := make(map[string]interface{})
	metadata["stars"] = repo.Stars
	metadata["forks"] = repo.Forks
	metadata["language"] = repo.Language
	metadata["created_at"] = repo.CreatedAt
	metadata["updated_at"] = repo.UpdatedAt
	
	return models.HarnessComponent{
		Identifier:  identifier,  // IDP 2.0 requires identifier field
		Name:        repo.Name,     // Keep original repo name with hyphens
		Type:        config.Defaults.Type,
		Lifecycle:   config.Defaults.Lifecycle,
		Owner:       getOwner(repo),
		System:      config.Defaults.System,
		Description: repo.Description,
		Tags:        tags,
		Annotations: annotations,
		Links:       links,
		Metadata:    metadata,
	}
}

func getOwner(repo models.Repository) string {
	if len(repo.CodeOwners) > 0 {
		return repo.CodeOwners[0]
	}
	return config.Defaults.Owner
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}