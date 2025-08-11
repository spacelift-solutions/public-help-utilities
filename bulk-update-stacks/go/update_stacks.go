package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Config holds all configuration from environment variables
type Config struct {
	SpaceliftDomain       string
	SpaceliftAPIKeyID     string
	SpaceliftAPIKeySecret string
	SpaceliftURL          string
}

// NewConfig creates a new config from environment variables
func NewConfig() (*Config, error) {
	domain := os.Getenv("SPACELIFT_DOMAIN")
	keyID := os.Getenv("SPACELIFT_API_KEY_ID")
	keySecret := os.Getenv("SPACELIFT_API_KEY_SECRET")

	if domain == "" {
		return nil, fmt.Errorf("SPACELIFT_DOMAIN environment variable is required")
	}
	if keyID == "" {
		return nil, fmt.Errorf("SPACELIFT_API_KEY_ID environment variable is required")
	}
	if keySecret == "" {
		return nil, fmt.Errorf("SPACELIFT_API_KEY_SECRET environment variable is required")
	}

	config := &Config{
		SpaceliftDomain:       domain,
		SpaceliftAPIKeyID:     keyID,
		SpaceliftAPIKeySecret: keySecret,
		SpaceliftURL:          fmt.Sprintf("https://%s.app.spacelift.io/graphql", domain),
	}

	return config, nil
}

// GraphQL request structure
type GraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// GraphQL response structure
type GraphQLResponse struct {
	Data   any `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Authentication response
type AuthResponse struct {
	Data struct {
		ApiKeyUser struct {
			ID  string `json:"id"`
			JWT string `json:"jwt"`
		} `json:"apiKeyUser"`
	} `json:"data"`
}

// Stack structures
type WorkerPool struct {
	ID    string `json:"id"`
	Space string `json:"space"`
}

type VCSIntegration struct {
	ID string `json:"id"`
}

type VendorConfig struct {
	Typename string `json:"__typename"`
}

type Stack struct {
	Administrative   bool            `json:"administrative"`
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Branch           string          `json:"branch"`
	Repository       string          `json:"repository"`
	RepositoryURL    string          `json:"repositoryURL"`
	ProjectRoot      string          `json:"projectRoot"`
	Space            string          `json:"space"`
	Namespace        string          `json:"namespace"`
	Provider         string          `json:"provider"`
	Labels           []string        `json:"labels"`
	Description      *string         `json:"description"`
	VCSIntegration   *VCSIntegration `json:"vcsIntegration"`
	TerraformVersion *string         `json:"terraformVersion"`
	WorkerPool       *WorkerPool     `json:"workerPool,omitempty"`
	VendorConfig     *VendorConfig   `json:"vendorConfig"`
}

type StacksResponse struct {
	Data struct {
		Stacks []Stack `json:"stacks"`
	} `json:"data"`
}

// Update result tracking
type UpdateResult struct {
	StackID string
	Success bool
	Error   error
}

type UpdateSummary struct {
	TotalStacks   int
	SuccessCount  int
	FailureCount  int
	SuccessStacks []string
	FailureStacks []UpdateResult
	ExecutionTime time.Duration
}

// SpaceliftClient handles GraphQL requests to Spacelift
type SpaceliftClient struct {
	token  string
	config *Config
	client *http.Client
}

func NewSpaceliftClient(config *Config) *SpaceliftClient {
	return &SpaceliftClient{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *SpaceliftClient) authenticate() error {
	authQuery := `
		mutation GetSpaceliftToken($keyId: ID!, $keySecret: String!) {
			apiKeyUser(id: $keyId, secret: $keySecret) {
				id
				jwt
			}
		}
	`

	req := GraphQLRequest{
		Query: authQuery,
		Variables: map[string]interface{}{
			"keyId":     c.config.SpaceliftAPIKeyID,
			"keySecret": c.config.SpaceliftAPIKeySecret,
		},
	}

	var authResp AuthResponse
	if err := c.makeRequest(req, &authResp); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	c.token = authResp.Data.ApiKeyUser.JWT
	return nil
}

func (c *SpaceliftClient) getAllStacks() ([]Stack, error) {
	query := `
		query {
			stacks {
				id
				name
				branch
				repository 
				repositoryURL
				labels
				projectRoot
				description
				space
				administrative
				namespace
				provider
				terraformVersion
				workerPool {
					id
					space
				}
				vcsIntegration {
					id
				}
				vendorConfig {
					__typename
				}
			}
		}
	`

	req := GraphQLRequest{Query: query}
	var resp StacksResponse
	if err := c.makeRequest(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to get stacks: %w", err)
	}

	return resp.Data.Stacks, nil
}

func (c *SpaceliftClient) updateStack(stackID string, stack *Stack) error {
	updateQuery := `
		mutation UpdateStack($stackId: ID!, $input: StackInput!) {
			stackUpdate(id: $stackId, input: $input) {
				id
				__typename
			}
		}
	`

	// Preserve existing labels and add new one
	addNewLabels := make([]string, len(stack.Labels))
	copy(addNewLabels, stack.Labels)

	input := map[string]any{}

	// Administrative
	if val := os.Getenv("SPACELIFT_ADMINISTRATIVE"); val != "" {
		input["administrative"] = val
	} else {
		input["administrative"] = stack.Administrative
	}

	// Name
	if val := os.Getenv("SPACELIFT_NAME"); val != "" {
		input["name"] = val
	} else {
		input["name"] = stack.Name
	}

	// Branch
	if val := os.Getenv("SPACELIFT_BRANCH"); val != "" {
		input["branch"] = val
	} else {
		input["branch"] = stack.Branch
	}

	// Namespace
	if val := os.Getenv("SPACELIFT_NAMESPACE"); val != "" {
		input["namespace"] = val
	} else {
		input["namespace"] = stack.Namespace
	}

	// Provider
	if val := os.Getenv("SPACELIFT_PROVIDER"); val != "" {
		input["provider"] = val
	} else {
		input["provider"] = stack.Provider
	}

	// Repository
	if val := os.Getenv("SPACELIFT_REPOSITORY"); val != "" {
		input["repository"] = val
	} else {
		input["repository"] = stack.Repository
	}

	// RepositoryURL
	if val := os.Getenv("SPACELIFT_REPOSITORY_URL"); val != "" {
		input["repositoryURL"] = val
	} else {
		input["repositoryURL"] = stack.RepositoryURL
	}

	// ProjectRoot
	if val := os.Getenv("SPACELIFT_PROJECT_ROOT"); val != "" {
		input["projectRoot"] = val
	} else {
		input["projectRoot"] = stack.ProjectRoot
	}

	// Space
	if val := os.Getenv("SPACELIFT_SPACE"); val != "" {
		input["space"] = val
	} else {
		input["space"] = stack.Space
	}

	// Labels (comma-separated)
	if val := os.Getenv("SPACELIFT_LABELS"); val != "" {
		labels := strings.Split(val, ",")
		for i := range labels {
			labels[i] = strings.TrimSpace(labels[i])
		}
		input["labels"] = append(addNewLabels, labels...)
	} else {
		input["labels"] = stack.Labels
	}

	// WorkerPool
	if workerPool := os.Getenv("SPACELIFT_WORKER_POOL_ID"); workerPool != "" {
		input["workerPool"] = workerPool
	} else if stack.WorkerPool != nil {
		input["workerPool"] = stack.WorkerPool.ID
	}
	req := GraphQLRequest{
		Query: updateQuery,
		Variables: map[string]any{
			"stackId": stackID,
			"input":   input,
		},
	}

	var resp GraphQLResponse
	return c.makeRequest(req, &resp)
}

func (c *SpaceliftClient) makeRequest(req GraphQLRequest, result interface{}) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", c.config.SpaceliftURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d, body: %s", resp.StatusCode, string(body))
	}

	// Check for GraphQL errors
	var graphqlResp GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResp); err != nil {
		return err
	}

	if len(graphqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", graphqlResp.Errors[0].Message)
	}

	return json.Unmarshal(body, result)
}

func updateStack(client *SpaceliftClient, stack Stack, resultChan chan<- UpdateResult, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("Processing stack: %s\n", stack.ID)

	// Update the stack directly using the information from getAllStacks
	err := client.updateStack(stack.ID, &stack)
	result := UpdateResult{
		StackID: stack.ID,
		Success: err == nil,
		Error:   err,
	}

	if err == nil {
		fmt.Printf("✓ Successfully updated stack: %s\n", stack.ID)
	} else {
		fmt.Printf("✗ Failed to update stack %s: %v\n", stack.ID, err)
	}

	resultChan <- result
}

func main() {
	startTime := time.Now()

	// Load configuration from environment variables
	config, err := NewConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Create client and authenticate
	client := NewSpaceliftClient(config)

	fmt.Println("Authenticating with Spacelift...")
	if err := client.authenticate(); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	fmt.Println("✓ Authentication successful")

	// Get all stacks
	fmt.Println("Retrieving all stacks...")
	stacks, err := client.getAllStacks()
	if err != nil {
		log.Fatalf("Failed to get stacks: %v", err)
	}
	fmt.Printf("✓ Retrieved %d stacks\n", len(stacks))

	// Start updating stacks concurrently
	fmt.Println("Starting bulk update...")
	var wg sync.WaitGroup
	resultChan := make(chan UpdateResult, len(stacks))

	// Launch goroutine for each stack
	terraformStacks := 0
	for _, stack := range stacks {
		if stack.VendorConfig.Typename != "StackConfigVendorTerraform" {
			continue
		}
		terraformStacks++
		wg.Add(1)
		go updateStack(client, stack, resultChan, &wg)
	}

	// Wait for all updates to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	summary := UpdateSummary{
		TotalStacks:   terraformStacks,
		SuccessCount:  0,
		FailureCount:  0,
		FailureStacks: make([]UpdateResult, 0),
		ExecutionTime: time.Since(startTime),
	}

	for result := range resultChan {
		if result.Success {
			summary.SuccessCount++
		} else {
			summary.FailureCount++
			summary.FailureStacks = append(summary.FailureStacks, result)
		}
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("BULK UPDATE SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total stacks processed: %d\n", summary.TotalStacks)
	fmt.Printf("Successful updates: %d\n", summary.SuccessCount)
	fmt.Printf("Failed updates: %d\n", summary.FailureCount)
	fmt.Printf("Execution time: %v\n", summary.ExecutionTime)
	fmt.Printf("Success rate: %.2f%%\n", float64(summary.SuccessCount)/float64(summary.TotalStacks)*100)

	if summary.FailureCount > 0 {
		fmt.Println("\n✗ Failed to update stacks:")
		for _, failure := range summary.FailureStacks {
			fmt.Printf("  - %s: %v\n", failure.StackID, failure.Error)
		}
	}

	fmt.Println(strings.Repeat("=", 60))
}
