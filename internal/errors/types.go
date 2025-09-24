package errors

import (
	"fmt"
	"strings"
)

// ErrorCategory represents different types of errors that can occur
type ErrorCategory string

const (
	ErrorCategoryRepository    ErrorCategory = "REPOSITORY"
	ErrorCategoryEntity       ErrorCategory = "ENTITY"
	ErrorCategoryAuthentication ErrorCategory = "AUTHENTICATION"
	ErrorCategoryValidation   ErrorCategory = "VALIDATION"
	ErrorCategoryNetwork      ErrorCategory = "NETWORK"
	ErrorCategoryPR           ErrorCategory = "PULL_REQUEST"
	ErrorCategoryUnknown      ErrorCategory = "UNKNOWN"
)

// ErrorType represents specific error types within categories
type ErrorType string

const (
	// Repository errors
	ErrorTypeRepositoryNotFound     ErrorType = "REPOSITORY_NOT_FOUND"
	ErrorTypeRepositoryAccessDenied ErrorType = "REPOSITORY_ACCESS_DENIED"
	ErrorTypeCatalogFileNotFound    ErrorType = "CATALOG_FILE_NOT_FOUND"
	ErrorTypeCatalogFileInvalid     ErrorType = "CATALOG_FILE_INVALID"
	
	// Entity errors
	ErrorTypeEntityExists           ErrorType = "ENTITY_EXISTS"
	ErrorTypeEntityAlreadyRegistered ErrorType = "ENTITY_ALREADY_REGISTERED"
	ErrorTypeEntityNotFound         ErrorType = "ENTITY_NOT_FOUND"
	ErrorTypeEntityValidationFailed ErrorType = "ENTITY_VALIDATION_FAILED"
	
	// Authentication errors
	ErrorTypeUnauthorized   ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden      ErrorType = "FORBIDDEN"
	ErrorTypeAPIKeyInvalid  ErrorType = "API_KEY_INVALID"
	
	// Validation errors
	ErrorTypeInvalidIdentifier ErrorType = "INVALID_IDENTIFIER"
	ErrorTypeMissingField      ErrorType = "MISSING_FIELD"
	ErrorTypeInvalidValue      ErrorType = "INVALID_VALUE"
	
	// Network errors
	ErrorTypeRateLimit     ErrorType = "RATE_LIMIT"
	ErrorTypeTimeout       ErrorType = "TIMEOUT"
	ErrorTypeConnectionFailed ErrorType = "CONNECTION_FAILED"
	
	// Pull Request errors
	ErrorTypePRExists      ErrorType = "PR_EXISTS"
	ErrorTypePRConflict    ErrorType = "PR_CONFLICT"
	ErrorTypePRCreateFailed ErrorType = "PR_CREATE_FAILED"
	
	// Unknown errors
	ErrorTypeUnknown ErrorType = "UNKNOWN"
)

// ProcessingError represents a structured error with category, type, and context
type ProcessingError struct {
	Category   ErrorCategory
	Type       ErrorType
	Message    string
	Repository string
	Cause      error
	Recoverable bool
	UserFriendly string
}

func (e *ProcessingError) Error() string {
	if e.Repository != "" {
		return fmt.Sprintf("[%s:%s] %s (repo: %s)", e.Category, e.Type, e.Message, e.Repository)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Category, e.Type, e.Message)
}

func (e *ProcessingError) Unwrap() error {
	return e.Cause
}

// IsRecoverable returns true if the error is likely recoverable with retry
func (e *ProcessingError) IsRecoverable() bool {
	return e.Recoverable
}

// GetUserFriendlyMessage returns a user-friendly error message
func (e *ProcessingError) GetUserFriendlyMessage() string {
	if e.UserFriendly != "" {
		return e.UserFriendly
	}
	return e.Message
}

// NewRepositoryNotFoundError creates an error for when a repository is not found
func NewRepositoryNotFoundError(repo string, cause error) *ProcessingError {
	return &ProcessingError{
		Category:     ErrorCategoryRepository,
		Type:         ErrorTypeRepositoryNotFound,
		Message:      "repository not found or inaccessible",
		Repository:   repo,
		Cause:        cause,
		Recoverable:  false,
		UserFriendly: fmt.Sprintf("Repository '%s' was not found or you don't have access to it. Please check the repository name and your permissions.", repo),
	}
}

// NewEntityExistsError creates an error for when an entity already exists
func NewEntityExistsError(repo string, identifier string, cause error) *ProcessingError {
	return &ProcessingError{
		Category:     ErrorCategoryEntity,
		Type:         ErrorTypeEntityExists,
		Message:      fmt.Sprintf("entity with identifier '%s' already exists", identifier),
		Repository:   repo,
		Cause:        cause,
		Recoverable:  false,
		UserFriendly: fmt.Sprintf("Component '%s' already exists in Harness IDP. Use update mode or remove the existing component first.", identifier),
	}
}

// NewEntityAlreadyRegisteredError creates an error for when an entity is already registered
func NewEntityAlreadyRegisteredError(repo string, cause error) *ProcessingError {
	return &ProcessingError{
		Category:     ErrorCategoryEntity,
		Type:         ErrorTypeEntityAlreadyRegistered,
		Message:      "entity already registered",
		Repository:   repo,
		Cause:        cause,
		Recoverable:  false,
		UserFriendly: fmt.Sprintf("Repository '%s' has already been imported into Harness IDP. No action needed.", repo),
	}
}

// NewCatalogFileNotFoundError creates an error for when catalog-info.yaml is missing
func NewCatalogFileNotFoundError(repo string, cause error) *ProcessingError {
	return &ProcessingError{
		Category:     ErrorCategoryRepository,
		Type:         ErrorTypeCatalogFileNotFound,
		Message:      "catalog-info.yaml file not found",
		Repository:   repo,
		Cause:        cause,
		Recoverable:  false,
		UserFriendly: fmt.Sprintf("Repository '%s' doesn't have a catalog-info.yaml file. Create one first or use YAML mode to generate it.", repo),
	}
}

// NewPRExistsError creates an error for when a PR already exists
func NewPRExistsError(repo string, prNumber int, cause error) *ProcessingError {
	return &ProcessingError{
		Category:     ErrorCategoryPR,
		Type:         ErrorTypePRExists,
		Message:      fmt.Sprintf("pull request already exists (PR #%d)", prNumber),
		Repository:   repo,
		Cause:        cause,
		Recoverable:  false,
		UserFriendly: fmt.Sprintf("Repository '%s' already has an open pull request for Harness onboarding (PR #%d). Please review and merge it first.", repo, prNumber),
	}
}

// NewPRExistsErrorWithTitle creates an error for when a PR already exists with title
func NewPRExistsErrorWithTitle(repo string, prNumber int, title string, cause error) *ProcessingError {
	return &ProcessingError{
		Category:     ErrorCategoryPR,
		Type:         ErrorTypePRExists,
		Message:      fmt.Sprintf("open PR #%d already exists (%s)", prNumber, title),
		Repository:   repo,
		Cause:        cause,
		Recoverable:  false,
		UserFriendly: fmt.Sprintf("Repository '%s' already has an open Harness onboarding PR #%d ('%s'). Please review and merge it first.", repo, prNumber, title),
	}
}

// NewUnauthorizedError creates an error for authentication issues
func NewUnauthorizedError(message string, cause error) *ProcessingError {
	return &ProcessingError{
		Category:     ErrorCategoryAuthentication,
		Type:         ErrorTypeUnauthorized,
		Message:      message,
		Cause:        cause,
		Recoverable:  false,
		UserFriendly: "Authentication failed. Please check your API keys and permissions.",
	}
}

// NewRateLimitError creates an error for rate limiting
func NewRateLimitError(cause error) *ProcessingError {
	return &ProcessingError{
		Category:     ErrorCategoryNetwork,
		Type:         ErrorTypeRateLimit,
		Message:      "rate limit exceeded",
		Cause:        cause,
		Recoverable:  true,
		UserFriendly: "API rate limit exceeded. The tool will retry automatically after a delay.",
	}
}

// CategorizeError analyzes an error and returns a structured ProcessingError
func CategorizeError(err error, repo string) *ProcessingError {
	if err == nil {
		return nil
	}
	
	// If already a ProcessingError, return as-is
	if procErr, ok := err.(*ProcessingError); ok {
		if procErr.Repository == "" {
			procErr.Repository = repo
		}
		return procErr
	}
	
	errMsg := strings.ToLower(err.Error())
	
	// GitHub API errors
	if strings.Contains(errMsg, "404") && strings.Contains(errMsg, "not found") {
		return NewRepositoryNotFoundError(repo, err)
	}
	if strings.Contains(errMsg, "403") && strings.Contains(errMsg, "forbidden") {
		return &ProcessingError{
			Category:     ErrorCategoryAuthentication,
			Type:         ErrorTypeForbidden,
			Message:      "access forbidden",
			Repository:   repo,
			Cause:        err,
			Recoverable:  false,
			UserFriendly: fmt.Sprintf("Access to repository '%s' is forbidden. Check your GitHub App permissions.", repo),
		}
	}
	if strings.Contains(errMsg, "401") && strings.Contains(errMsg, "unauthorized") {
		return NewUnauthorizedError("GitHub authentication failed", err)
	}
	if strings.Contains(errMsg, "429") || strings.Contains(errMsg, "rate limit") {
		return NewRateLimitError(err)
	}
	
	// Harness API errors
	if strings.Contains(errMsg, "duplicate_file_import") || strings.Contains(errMsg, "already been imported") {
		return NewEntityAlreadyRegisteredError(repo, err)
	}
	if strings.Contains(errMsg, "already exists") || strings.Contains(errMsg, "duplicate") {
		return NewEntityExistsError(repo, "unknown", err)
	}
	
	// Catalog file errors
	if strings.Contains(errMsg, "catalog-info.yaml") && strings.Contains(errMsg, "not found") {
		return NewCatalogFileNotFoundError(repo, err)
	}
	
	// PR errors
	if strings.Contains(errMsg, "pull request") && strings.Contains(errMsg, "already") {
		return NewPRExistsError(repo, 0, err)
	}
	
	// Default to unknown error
	return &ProcessingError{
		Category:     ErrorCategoryUnknown,
		Type:         ErrorTypeUnknown,
		Message:      err.Error(),
		Repository:   repo,
		Cause:        err,
		Recoverable:  false,
		UserFriendly: fmt.Sprintf("An unexpected error occurred while processing '%s': %s", repo, err.Error()),
	}
}

// ProcessingResult represents the result of processing a repository
type ProcessingResult struct {
	Repository string
	Success    bool
	Error      *ProcessingError
	Message    string
	Skipped    bool
	Action     string // "created", "updated", "skipped", "failed"
}

// ErrorSummary provides a summary of all errors encountered
type ErrorSummary struct {
	Total     int
	ByCategory map[ErrorCategory]int
	ByType     map[ErrorType]int
	Recoverable int
	Results    []ProcessingResult
}

// NewErrorSummary creates a new error summary
func NewErrorSummary() *ErrorSummary {
	return &ErrorSummary{
		ByCategory: make(map[ErrorCategory]int),
		ByType:     make(map[ErrorType]int),
		Results:    make([]ProcessingResult, 0),
	}
}

// AddResult adds a processing result to the summary
func (s *ErrorSummary) AddResult(result ProcessingResult) {
	s.Results = append(s.Results, result)
	
	if result.Error != nil {
		s.Total++
		s.ByCategory[result.Error.Category]++
		s.ByType[result.Error.Type]++
		
		if result.Error.Recoverable {
			s.Recoverable++
		}
	}
}

// PrintSummary prints a formatted summary of all errors
func (s *ErrorSummary) PrintSummary() {
	if s.Total == 0 {
		fmt.Println("✅ All repositories processed successfully!")
		return
	}
	
	fmt.Printf("\n📊 Processing Summary:\n")
	fmt.Printf("   Total repositories: %d\n", len(s.Results))
	fmt.Printf("   Successful: %d\n", len(s.Results)-s.Total)
	fmt.Printf("   Failed: %d\n", s.Total)
	fmt.Printf("   Recoverable errors: %d\n", s.Recoverable)
	
	if len(s.ByCategory) > 0 {
		fmt.Printf("\n🏷️  Error Categories:\n")
		for category, count := range s.ByCategory {
			fmt.Printf("   %s: %d\n", category, count)
		}
	}
	
	fmt.Printf("\n📝 Detailed Results:\n")
	for _, result := range s.Results {
		status := "✅"
		if result.Error != nil {
			if result.Error.Recoverable {
				status = "⚠️ "
			} else {
				status = "❌"
			}
		} else if result.Skipped {
			status = "⏭️ "
		}
		
		fmt.Printf("   %s %s - %s\n", status, result.Repository, result.Message)
		if result.Error != nil {
			fmt.Printf("      └─ %s\n", result.Error.GetUserFriendlyMessage())
		}
	}
}