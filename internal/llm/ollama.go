package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ashprao/ollamachat/internal/models"
	"github.com/ashprao/ollamachat/pkg/logger"
)

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	baseURL    string
	httpClient *http.Client
	logger     *logger.Logger
}

// NewOllamaProvider creates a new Ollama provider instance
func NewOllamaProvider(baseURL string, logger *logger.Logger) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &OllamaProvider{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.WithComponent("ollama-provider"),
	}
}

// GetName returns the provider name
func (o *OllamaProvider) GetName() string {
	return "ollama"
}

// SupportsTools returns whether this provider supports tool calling
func (o *OllamaProvider) SupportsTools() bool {
	// Ollama doesn't natively support tool calling yet, but this can be extended
	return false
}

// GetModels retrieves available models from Ollama
func (o *OllamaProvider) GetModels(ctx context.Context) ([]models.Model, error) {
	o.logger.Info("Fetching available models")

	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/api/tags", nil)
	if err != nil {
		o.logger.Error("Failed to create request for models", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		o.logger.Error("Failed to fetch models from Ollama", "error", err)
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		o.logger.Error("Unexpected status code from Ollama", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		o.logger.Error("Failed to decode models response", "error", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our models.Model struct
	modelList := make([]models.Model, len(result.Models))
	for i, m := range result.Models {
		modelList[i] = models.Model{
			Name:        m.Name,
			Description: fmt.Sprintf("Ollama model: %s", m.Name),
		}
	}

	o.logger.Info("Successfully fetched models", "count", len(modelList))
	return modelList, nil
}

// SendQuery sends a query to Ollama and streams the response
func (o *OllamaProvider) SendQuery(ctx context.Context, model, query string, onUpdate StreamCallback) error {
	o.logger.Info("Sending query to Ollama",
		"model", model,
		"query_length", len(query))

	requestBody := map[string]interface{}{
		"model":  model,
		"prompt": query,
		"stream": true, // Enable streaming
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		o.logger.Error("Failed to marshal request body", "error", err)
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		o.logger.Error("Failed to create request", "error", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		o.logger.Error("Failed to send request to Ollama", "error", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		o.logger.Error("Unexpected status code from Ollama", "status_code", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return o.handleStreamingResponse(resp.Body, onUpdate)
}

// SendQueryWithTools sends a query with tools - not supported by Ollama yet
func (o *OllamaProvider) SendQueryWithTools(ctx context.Context, model, query string, tools []models.MCPTool, onUpdate StreamCallback) error {
	o.logger.Warn("Tool calling not supported by Ollama provider", "model", model)
	return fmt.Errorf("tool calling not supported by Ollama provider")
}

// handleStreamingResponse processes the streaming response from Ollama
func (o *OllamaProvider) handleStreamingResponse(body io.Reader, onUpdate StreamCallback) error {
	decoder := json.NewDecoder(body)
	newStream := true

	for {
		var response struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
			Error    string `json:"error,omitempty"`
		}

		if err := decoder.Decode(&response); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			o.logger.Error("Failed to decode streaming response", "error", err)
			return fmt.Errorf("failed to decode response: %w", err)
		}

		// Check for errors in the response
		if response.Error != "" {
			o.logger.Error("Error in Ollama response", "error", response.Error)
			return fmt.Errorf("ollama error: %s", response.Error)
		}

		// Call the update callback with the response chunk
		if response.Response != "" {
			onUpdate(response.Response, newStream)
			newStream = false
		}

		// Break if this is the final chunk
		if response.Done {
			break
		}
	}

	o.logger.Info("Successfully completed streaming response")
	return nil
}

// DefaultOllamaProviderFactory creates a default factory for Ollama providers
type DefaultOllamaProviderFactory struct {
	logger *logger.Logger
}

// NewDefaultOllamaProviderFactory creates a new factory instance
func NewDefaultOllamaProviderFactory(logger *logger.Logger) *DefaultOllamaProviderFactory {
	return &DefaultOllamaProviderFactory{
		logger: logger,
	}
}

// CreateProvider creates an Ollama provider from configuration
func (f *DefaultOllamaProviderFactory) CreateProvider(config ProviderConfig) (Provider, error) {
	if config.Type != "ollama" {
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return NewOllamaProvider(baseURL, f.logger), nil
}

// SupportedProviders returns the list of supported provider types
func (f *DefaultOllamaProviderFactory) SupportedProviders() []string {
	return []string{"ollama"}
}
