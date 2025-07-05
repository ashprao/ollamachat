package llm

import (
	"context"

	"github.com/ashprao/ollamachat/internal/constants"
	"github.com/ashprao/ollamachat/internal/models"
)

// StreamCallback is called for each chunk of streaming response
type StreamCallback func(chunk string, isNewStream bool)

// Provider defines the interface for LLM providers
type Provider interface {
	// GetModels returns the list of available models
	GetModels(ctx context.Context) ([]models.Model, error)

	// SendQuery sends a query to the LLM and streams the response
	SendQuery(ctx context.Context, model, query string, onUpdate StreamCallback) error

	// SendQueryWithOptions sends a query with additional options (temperature, etc.)
	SendQueryWithOptions(ctx context.Context, model, query string, options QueryOptions, onUpdate StreamCallback) error

	// GetName returns the provider name
	GetName() string

	// SupportsTools returns whether this provider supports tool calling
	SupportsTools() bool

	// SendQueryWithTools sends a query with available tools for MCP integration
	SendQueryWithTools(ctx context.Context, model, query string, tools []models.MCPTool, onUpdate StreamCallback) error
}

// ProviderConfig holds configuration for creating providers
type ProviderConfig struct {
	Type     string                 // "ollama", "openai", "eino"
	BaseURL  string                 // Base URL for API
	APIKey   string                 // API key if required
	Settings map[string]interface{} // Additional provider-specific settings
}

// ProviderFactory creates providers based on configuration
type ProviderFactory interface {
	CreateProvider(config ProviderConfig) (Provider, error)
	SupportedProviders() []string
}

// QueryOptions holds additional parameters for LLM queries
type QueryOptions struct {
	Temperature float64
	MaxTokens   int
}

// DefaultQueryOptions returns default query options
func DefaultQueryOptions() QueryOptions {
	return QueryOptions{
		Temperature: constants.DefaultTemperature,
		MaxTokens:   constants.DefaultMaxTokens,
	}
}
