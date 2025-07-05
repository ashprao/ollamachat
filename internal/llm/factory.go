package llm

import (
	"fmt"

	"github.com/ashprao/ollamachat/internal/config"
	"github.com/ashprao/ollamachat/internal/constants"
	"github.com/ashprao/ollamachat/pkg/logger"
)

// DefaultProviderFactory implements ProviderFactory interface
type DefaultProviderFactory struct {
	config *config.Config
	logger *logger.Logger
}

// NewDefaultProviderFactory creates a new provider factory
func NewDefaultProviderFactory(config *config.Config, logger *logger.Logger) *DefaultProviderFactory {
	return &DefaultProviderFactory{
		config: config,
		logger: logger.WithComponent("provider-factory"),
	}
}

// CreateProvider creates a provider instance based on the provider configuration
func (f *DefaultProviderFactory) CreateProvider(providerConfig ProviderConfig) (Provider, error) {
	f.logger.Info("Creating LLM provider", "provider_type", providerConfig.Type)

	switch providerConfig.Type {
	case "ollama":
		return f.createOllamaProvider(providerConfig)
	case "openai":
		return f.createOpenAIProvider(providerConfig)
	case "eino":
		return f.createEinoProvider(providerConfig)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerConfig.Type)
	}
}

// SupportedProviders returns the list of supported provider types
func (f *DefaultProviderFactory) SupportedProviders() []string {
	return []string{"ollama", "openai", "eino"}
}

// CreateProviderFromConfig creates a provider using the current config
func (f *DefaultProviderFactory) CreateProviderFromConfig(providerType string) (Provider, error) {
	f.logger.Info("Creating LLM provider from config", "provider_type", providerType)

	switch providerType {
	case "ollama":
		return f.createOllamaFromConfig()
	case "openai":
		return f.createOpenAIFromConfig()
	case "eino":
		return f.createEinoFromConfig()
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetAvailableProviders returns the list of providers that can be created
func (f *DefaultProviderFactory) GetAvailableProviders() []string {
	return f.config.LLM.AvailableProviders
}

// GetCurrentProvider returns the currently configured provider type
func (f *DefaultProviderFactory) GetCurrentProvider() string {
	return f.config.LLM.Provider
}

// getTimeoutFromConfig extracts timeout_seconds from LLM config settings
func (f *DefaultProviderFactory) getTimeoutFromConfig() int {
	if timeoutValue, ok := f.config.LLM.Settings["timeout_seconds"]; ok {
		if timeout, ok := timeoutValue.(int); ok {
			return timeout
		}
	}
	// Fallback to default
	return constants.DefaultTimeoutSeconds
}

// ValidateProviderConfig validates if a provider can be created with current config
func (f *DefaultProviderFactory) ValidateProviderConfig(providerType string) error {
	switch providerType {
	case "ollama":
		if f.config.LLM.Ollama.BaseURL == "" {
			return fmt.Errorf("ollama base_url is required")
		}
		return nil
	case "openai":
		// Check if API key is available (from config or environment)
		if f.config.LLM.OpenAI.APIKey == "" {
			// TODO: Check environment variable OPENAI_API_KEY
			return fmt.Errorf("openai api_key is required")
		}
		return nil
	case "eino":
		// Eino provider validation (placeholder for now)
		return nil
	default:
		return fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// createOllamaProvider creates an Ollama provider instance from ProviderConfig
func (f *DefaultProviderFactory) createOllamaProvider(config ProviderConfig) (Provider, error) {
	baseURL, ok := config.Settings["base_url"].(string)
	if !ok || baseURL == "" {
		return nil, fmt.Errorf("ollama base_url is required")
	}

	// Get timeout from provider config or use default
	timeout := constants.DefaultTimeoutSeconds
	if timeoutValue, ok := config.Settings["timeout_seconds"]; ok {
		if timeoutInt, ok := timeoutValue.(int); ok {
			timeout = timeoutInt
		}
	}

	f.logger.Info("Creating Ollama provider",
		"base_url", baseURL,
		"timeout_seconds", timeout)
	provider := NewOllamaProviderWithTimeout(baseURL, timeout, f.logger)
	return provider, nil
}

// createOllamaFromConfig creates an Ollama provider instance from app config
func (f *DefaultProviderFactory) createOllamaFromConfig() (Provider, error) {
	config := f.config.LLM.Ollama
	if config.BaseURL == "" {
		return nil, fmt.Errorf("ollama base_url is required")
	}

	// Get timeout from config
	timeout := f.getTimeoutFromConfig()

	f.logger.Info("Creating Ollama provider from config",
		"base_url", config.BaseURL,
		"timeout_seconds", timeout)
	provider := NewOllamaProviderWithTimeout(config.BaseURL, timeout, f.logger)
	return provider, nil
}

// createOpenAIProvider creates an OpenAI provider instance (placeholder)
func (f *DefaultProviderFactory) createOpenAIProvider(config ProviderConfig) (Provider, error) {
	f.logger.Info("OpenAI provider not yet implemented")
	return nil, fmt.Errorf("openai provider not yet implemented")
}

// createOpenAIFromConfig creates an OpenAI provider instance from app config (placeholder)
func (f *DefaultProviderFactory) createOpenAIFromConfig() (Provider, error) {
	f.logger.Info("OpenAI provider not yet implemented")
	return nil, fmt.Errorf("openai provider not yet implemented")
}

// createEinoProvider creates an Eino provider instance (placeholder)
func (f *DefaultProviderFactory) createEinoProvider(config ProviderConfig) (Provider, error) {
	f.logger.Info("Eino provider not yet implemented")
	return nil, fmt.Errorf("eino provider not yet implemented")
}

// createEinoFromConfig creates an Eino provider instance from app config (placeholder)
func (f *DefaultProviderFactory) createEinoFromConfig() (Provider, error) {
	f.logger.Info("Eino provider not yet implemented")
	return nil, fmt.Errorf("eino provider not yet implemented")
}
