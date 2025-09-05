package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	"harness-onboarder/internal/models"
)

type Client struct {
	httpClient *http.Client
	config     models.HarnessConfig
	baseURL    *url.URL
}

type ComponentCreateRequest struct {
	Component models.HarnessComponent `json:"component"`
}

type ComponentResponse struct {
	Status    string                  `json:"status"`
	Component models.HarnessComponent `json:"component,omitempty"`
	Error     string                  `json:"error,omitempty"`
	Message   string                  `json:"message,omitempty"`
}

type ListComponentsResponse struct {
	Status     string                    `json:"status"`
	Components []models.HarnessComponent `json:"components,omitempty"`
	Total      int                       `json:"total"`
	Error      string                    `json:"error,omitempty"`
}

type EntityImportRequest struct {
	BranchName        string `json:"branch_name"`
	ConnectorRef      string `json:"connector_ref"`
	RepoName          string `json:"repo_name"`
	IsHarnessCodeRepo bool   `json:"is_harness_code_repo"`
	FilePath          string `json:"file_path"`
	Identifier        string `json:"identifier"`
	AccountIdentifier string `json:"accountIdentifier"`
	OrgIdentifier     string `json:"orgIdentifier"`
	ProjectIdentifier string `json:"projectIdentifier"`
}

type CatalogLocationResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type CatalogEntity struct {
	APIVersion        string `yaml:"apiVersion"`
	Identifier        string `yaml:"identifier"`
	Name              string `yaml:"name"`
	Kind              string `yaml:"kind"`
	Type              string `yaml:"type"`
	ProjectIdentifier string `yaml:"projectIdentifier"`
	OrgIdentifier     string `yaml:"orgIdentifier"`
	Owner             string `yaml:"owner"`
	Metadata          struct {
		Description string            `yaml:"description,omitempty"`
		Annotations map[string]string `yaml:"annotations,omitempty"`
		Tags        []string          `yaml:"tags,omitempty"`
		Links       []struct {
			URL   string `yaml:"url"`
			Title string `yaml:"title"`
			Icon  string `yaml:"icon,omitempty"`
			Type  string `yaml:"type,omitempty"`
		} `yaml:"links,omitempty"`
	} `yaml:"metadata,omitempty"`
	Spec struct {
		Lifecycle string `yaml:"lifecycle"`
	} `yaml:"spec"`
}

func NewClient(config models.HarnessConfig) (*Client, error) {
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
		},
	}

	return &Client{
		httpClient: httpClient,
		config:     config,
		baseURL:    baseURL,
	}, nil
}

func (c *Client) CreateComponent(ctx context.Context, component models.HarnessComponent) error {
	if err := c.validateComponent(component); err != nil {
		return fmt.Errorf("component validation failed: %w", err)
	}

	existing, err := c.GetComponent(ctx, component.Identifier)
	if err == nil && existing != nil {
		log.Printf("Component %s (identifier: %s) already exists, updating instead", component.Name, component.Identifier)
		return c.UpdateComponent(ctx, component)
	}

	// Convert component to YAML string for the new API format
	yamlData, err := c.componentToYAML(component)
	if err != nil {
		return fmt.Errorf("failed to convert component to YAML: %w", err)
	}

	// Create request body with YAML string
	reqBody := map[string]interface{}{
		"yaml": yamlData,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	log.Printf("DEBUG: Creating component with YAML payload: %s", string(jsonData))

	// Use the correct API endpoint
	endpoint := fmt.Sprintf("/gateway/v1/entities?convert=false&dry_run=false&accountIdentifier=%s&orgIdentifier=%s&projectIdentifier=%s",
		c.config.AccountID, c.config.OrgID, c.config.ProjectID)

	log.Printf("DEBUG: POST %s", endpoint)

	req, err := c.newRequest(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add required headers for entity creation API
	req.Header.Set("harness-account", c.config.AccountID)
	req.Header.Set("harness-org", c.config.OrgID)
	req.Header.Set("harness-project", c.config.ProjectID)

	// The new entity creation API returns a different response format
	var resp interface{} // Use generic interface to handle any response format
	if err := c.doRequest(req, &resp); err != nil {
		return fmt.Errorf("failed to create component: %w", err)
	}
	
	// For the entity creation API, success is indicated by HTTP 200/201 status
	// The response format may vary, so we don't need to parse specific fields

	log.Printf("Successfully created component: %s (identifier: %s)", component.Name, component.Identifier)
	return nil
}

// componentToYAML converts a HarnessComponent to IDP 2.0 YAML format
func (c *Client) componentToYAML(component models.HarnessComponent) (string, error) {
	yamlComponent := CatalogEntity{
		APIVersion:        "harness.io/v1",
		Kind:              "Component",
		Identifier:        component.Identifier,
		Name:              component.Name,
		Type:              component.Type,
		ProjectIdentifier: c.config.ProjectID,
		OrgIdentifier:     c.config.OrgID,
		Owner:             component.Owner,
		Metadata: struct {
			Description string            `yaml:"description,omitempty"`
			Annotations map[string]string `yaml:"annotations,omitempty"`
			Tags        []string          `yaml:"tags,omitempty"`
			Links       []struct {
				URL   string `yaml:"url"`
				Title string `yaml:"title"`
				Icon  string `yaml:"icon,omitempty"`
				Type  string `yaml:"type,omitempty"`
			} `yaml:"links,omitempty"`
		}{
			Description: component.Description,
			Annotations: component.Annotations,
			Tags:        component.Tags,
		},
		Spec: struct {
			Lifecycle string `yaml:"lifecycle"`
		}{
			Lifecycle: component.Lifecycle,
		},
	}

	// Convert component links
	for _, link := range component.Links {
		yamlComponent.Metadata.Links = append(yamlComponent.Metadata.Links, struct {
			URL   string `yaml:"url"`
			Title string `yaml:"title"`
			Icon  string `yaml:"icon,omitempty"`
			Type  string `yaml:"type,omitempty"`
		}{
			URL:   link.URL,
			Title: link.Title,
			Icon:  link.Icon,
			Type:  link.Type,
		})
	}

	yamlBytes, err := yaml.Marshal(yamlComponent)
	if err != nil {
		return "", fmt.Errorf("failed to marshal component to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

func (c *Client) UpdateComponent(ctx context.Context, component models.HarnessComponent) error {
	if err := c.validateComponent(component); err != nil {
		return fmt.Errorf("component validation failed: %w", err)
	}

	reqBody := ComponentCreateRequest{
		Component: component,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal component: %w", err)
	}

	endpoint := fmt.Sprintf("/gateway/idp/api/v1/accounts/%s/orgs/%s/projects/%s/catalog/components/%s",
		c.config.AccountID, c.config.OrgID, c.config.ProjectID, component.Identifier)

	req, err := c.newRequest(ctx, "PUT", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	var resp ComponentResponse
	if err := c.doRequest(req, &resp); err != nil {
		return fmt.Errorf("failed to update component: %w", err)
	}

	if resp.Status != "success" && resp.Status != "SUCCESS" {
		return fmt.Errorf("component update failed: %s - %s", resp.Status, resp.Error)
	}

	log.Printf("Successfully updated component: %s (identifier: %s)", component.Name, component.Identifier)
	return nil
}

func (c *Client) GetComponent(ctx context.Context, name string) (*models.HarnessComponent, error) {
	endpoint := fmt.Sprintf("/gateway/idp/api/v1/accounts/%s/orgs/%s/projects/%s/catalog/components/%s",
		c.config.AccountID, c.config.OrgID, c.config.ProjectID, name)

	req, err := c.newRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var resp ComponentResponse
	if err := c.doRequest(req, &resp); err != nil {
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get component: %w", err)
	}

	if resp.Status != "success" && resp.Status != "SUCCESS" {
		if resp.Status == "not_found" || resp.Status == "NOT_FOUND" {
			return nil, nil
		}
		return nil, fmt.Errorf("get component failed: %s - %s", resp.Status, resp.Error)
	}

	return &resp.Component, nil
}

func (c *Client) ListComponents(ctx context.Context) ([]models.HarnessComponent, error) {
	endpoint := fmt.Sprintf("/gateway/idp/api/v1/accounts/%s/orgs/%s/projects/%s/catalog/components",
		c.config.AccountID, c.config.OrgID, c.config.ProjectID)

	req, err := c.newRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var resp ListComponentsResponse
	if err := c.doRequest(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to list components: %w", err)
	}

	if resp.Status != "success" && resp.Status != "SUCCESS" {
		return nil, fmt.Errorf("list components failed: %s - %s", resp.Status, resp.Error)
	}

	return resp.Components, nil
}

func (c *Client) DeleteComponent(ctx context.Context, name string) error {
	endpoint := fmt.Sprintf("/gateway/idp/api/v1/accounts/%s/orgs/%s/projects/%s/catalog/components/%s",
		c.config.AccountID, c.config.OrgID, c.config.ProjectID, name)

	req, err := c.newRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	var resp ComponentResponse
	if err := c.doRequest(req, &resp); err != nil {
		return fmt.Errorf("failed to delete component: %w", err)
	}

	if resp.Status != "success" && resp.Status != "SUCCESS" {
		return fmt.Errorf("component deletion failed: %s - %s", resp.Status, resp.Error)
	}

	log.Printf("Successfully deleted component: %s", name)
	return nil
}

// RegisterCatalogLocation registers a repository for entity import with Harness IDP
func (c *Client) RegisterCatalogLocation(ctx context.Context, repoFullName, branchName, filePath, catalogContent string) error {
	// Extract just the repository name from the full name (owner/repo -> repo)
	repoName := strings.Split(repoFullName, "/")[1]
	
	// Parse catalog content to extract entity identifier for IDP 2.0
	entityIdentifier, err := c.extractEntityIdentifier(catalogContent)
	if err != nil {
		return fmt.Errorf("failed to extract entity identifier from catalog: %w", err)
	}
	
	// Sanitize the identifier - replace hyphens with underscores for API compatibility
	entityIdentifier = strings.ReplaceAll(entityIdentifier, "-", "_")
	
	connectorRef := c.config.ConnectorRef
	if connectorRef == "" {
		connectorRef = "account.Gihubapp" // Default fallback
	}

	reqBody := EntityImportRequest{
		BranchName:        branchName,
		ConnectorRef:      connectorRef,
		RepoName:          repoName, // Use just the repo name, not the full name
		IsHarnessCodeRepo: false,
		FilePath:          filePath,
		Identifier:        entityIdentifier, // IDP 2.0 requires identifier
		AccountIdentifier: c.config.AccountID,
		OrgIdentifier:     c.config.OrgID,
		ProjectIdentifier: c.config.ProjectID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal entity import request: %w", err)
	}

	log.Printf("DEBUG: Sending payload to /gateway/v1/entities/import: %s", string(jsonData))

	// Add org and project identifiers as query parameters
	endpoint := fmt.Sprintf("/gateway/v1/entities/import?accountIdentifier=%s&orgIdentifier=%s&projectIdentifier=%s",
		c.config.AccountID, c.config.OrgID, c.config.ProjectID)

	log.Printf("DEBUG: POST %s", endpoint)

	req, err := c.newEntityImportRequest(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	var resp map[string]interface{}
	if err := c.doRequest(req, &resp); err != nil {
		return fmt.Errorf("failed to import entity: %w", err)
	}

	log.Printf("Successfully imported entity for repository: %s", repoFullName)
	return nil
}

// extractEntityIdentifier parses catalog-info.yaml content and extracts the entity identifier
func (c *Client) extractEntityIdentifier(catalogContent string) (string, error) {
	var entity CatalogEntity
	
	err := yaml.Unmarshal([]byte(catalogContent), &entity)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	// Check if it's new IDP 2.0 format with top-level identifier
	if entity.Identifier != "" {
		return entity.Identifier, nil
	}
	
	// Fall back to legacy Backstage format - parse as generic map
	var legacyEntity map[string]interface{}
	err = yaml.Unmarshal([]byte(catalogContent), &legacyEntity)
	if err != nil {
		return "", fmt.Errorf("failed to parse legacy YAML: %w", err)
	}
	
	// Extract name from metadata.name for legacy format
	if metadata, ok := legacyEntity["metadata"].(map[interface{}]interface{}); ok {
		if name, ok := metadata["name"].(string); ok && name != "" {
			return name, nil
		}
	}
	
	return "", fmt.Errorf("entity identifier not found in catalog")
}

func (c *Client) ValidateConnection(ctx context.Context) error {
	endpoint := fmt.Sprintf("/gateway/idp/api/v1/accounts/%s/orgs/%s/projects/%s/catalog/health",
		c.config.AccountID, c.config.OrgID, c.config.ProjectID)

	req, err := c.newRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	var resp map[string]interface{}
	if err := c.doRequest(req, &resp); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	log.Printf("Harness IDP connection validated successfully")
	return nil
}

func (c *Client) newRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	u, err := c.baseURL.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("User-Agent", "harness-onboarder/1.0.0")

	return req, nil
}

func (c *Client) newEntityImportRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	u, err := c.baseURL.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	
	// Try x-api-key authentication first (for PAT tokens)
	if strings.HasPrefix(c.config.APIKey, "pat.") {
		req.Header.Set("x-api-key", c.config.APIKey)
	} else {
		// Use Bearer token for JWT tokens
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))
	}
	
	req.Header.Set("harness-account", c.config.AccountID)
	req.Header.Set("User-Agent", "harness-onboarder/1.0.0")

	return req, nil
}

func (c *Client) doRequest(req *http.Request, result interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(body),
		}
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *Client) validateComponent(component models.HarnessComponent) error {
	// IDP 2.0 requires identifier field
	if component.Identifier == "" {
		return fmt.Errorf("component identifier is required")
	}
	if component.Name == "" {
		return fmt.Errorf("component name is required")
	}
	if component.Type == "" {
		return fmt.Errorf("component type is required")
	}
	if component.Lifecycle == "" {
		return fmt.Errorf("component lifecycle is required")
	}
	if component.Owner == "" {
		return fmt.Errorf("component owner is required")
	}

	validTypes := map[string]bool{
		"service":   true,
		"website":   true,
		"library":   true,
		"resource":  true,
		"api":       true,
		"database":  true,
		"system":    true,
		"domain":    true,
		"component": true,
	}

	if !validTypes[component.Type] {
		log.Printf("Warning: component type '%s' may not be recognized by Harness IDP", component.Type)
	}

	validLifecycles := map[string]bool{
		"experimental": true,
		"production":   true,
		"deprecated":   true,
	}

	if !validLifecycles[component.Lifecycle] {
		log.Printf("Warning: component lifecycle '%s' may not be recognized by Harness IDP", component.Lifecycle)
	}

	return nil
}

type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s - %s", e.StatusCode, e.Status, e.Body)
}

func (e *HTTPError) IsNotFound() bool {
	return e.StatusCode == 404
}

func (e *HTTPError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

func (e *HTTPError) IsForbidden() bool {
	return e.StatusCode == 403
}

func (e *HTTPError) IsRateLimited() bool {
	return e.StatusCode == 429
}

func isNotFoundError(err error) bool {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.IsNotFound()
	}
	return false
}