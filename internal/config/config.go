package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ashprao/ollamachat/internal/constants"
	"github.com/ashprao/ollamachat/internal/validation"
	"gopkg.in/yaml.v3"
)

type Config struct {
	App   AppConfig   `yaml:"app"`
	LLM   LLMConfig   `yaml:"llm"`
	UI    UIConfig    `yaml:"ui"`
	MCP   MCPConfig   `yaml:"mcp"`
	Agent AgentConfig `yaml:"agent"`
}

type AppConfig struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	LogLevel string `yaml:"log_level"`
}

type LLMConfig struct {
	Provider           string                 `yaml:"provider"`            // "ollama", "openai", "eino"
	AvailableProviders []string               `yaml:"available_providers"` // List of configured providers
	Ollama             OllamaConfig           `yaml:"ollama"`
	OpenAI             OpenAIConfig           `yaml:"openai"`
	Eino               EinoConfig             `yaml:"eino"`
	Settings           map[string]interface{} `yaml:"settings"`
}

type OllamaConfig struct {
	BaseURL      string `yaml:"base_url"`
	DefaultModel string `yaml:"default_model"`
}

type OpenAIConfig struct {
	APIKey       string `yaml:"api_key"`
	BaseURL      string `yaml:"base_url"`
	DefaultModel string `yaml:"default_model"`
}

type EinoConfig struct {
	DefaultModel string            `yaml:"default_model"`
	Settings     map[string]string `yaml:"settings"`
}

type UIConfig struct {
	WindowWidth    int    `yaml:"window_width"`
	WindowHeight   int    `yaml:"window_height"`
	MaxMessages    int    `yaml:"max_messages"`
	Theme          string `yaml:"theme"` // "light", "dark", "auto"
	FontSize       int    `yaml:"font_size"`
	ShowTimestamps bool   `yaml:"show_timestamps"`
	SidebarWidth   int    `yaml:"sidebar_width"` // Session sidebar width
}

type MCPConfig struct {
	Enabled bool              `yaml:"enabled"`
	Servers []MCPServerConfig `yaml:"servers"`
}

type MCPServerConfig struct {
	Name    string            `yaml:"name"`
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env"`
	Enabled bool              `yaml:"enabled"`
}

type AgentConfig struct {
	Enabled      bool                   `yaml:"enabled"`
	Framework    string                 `yaml:"framework"` // "eino", "custom"
	DefaultAgent string                 `yaml:"default_agent"`
	Settings     map[string]interface{} `yaml:"settings"`
}

// LoadConfig loads configuration from the specified file path
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	// Create default config if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.LLM.OpenAI.APIKey = apiKey
	}

	return &config, nil
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig(configPath string) error {
	defaultConfig := &Config{
		App: AppConfig{
			Name:     "OllamaChat",
			Version:  "1.0.0",
			LogLevel: "info",
		},
		LLM: LLMConfig{
			Provider: constants.DefaultProvider,
			Ollama: OllamaConfig{
				BaseURL:      "http://localhost:11434",
				DefaultModel: constants.DefaultModelName,
			},
			OpenAI: OpenAIConfig{
				BaseURL:      "https://api.openai.com/v1",
				DefaultModel: "gpt-3.5-turbo",
			},
			Eino: EinoConfig{
				DefaultModel: constants.DefaultModelName,
				Settings:     map[string]string{},
			},
			Settings: map[string]interface{}{
				"timeout_seconds": constants.DefaultTimeoutSeconds,
				"max_tokens":      constants.DefaultMaxTokens,
			},
		},
		UI: UIConfig{
			WindowWidth:    constants.DefaultWindowWidth, // Increased to accommodate session sidebar
			WindowHeight:   constants.DefaultWindowHeight,
			MaxMessages:    constants.DefaultMaxMessages,
			Theme:          "auto",
			FontSize:       constants.DefaultFontSize,
			ShowTimestamps: false,
			SidebarWidth:   constants.DefaultSidebarWidth,
		},
		MCP: MCPConfig{
			Enabled: false,
			Servers: []MCPServerConfig{},
		},
		Agent: AgentConfig{
			Enabled:      false,
			Framework:    "eino",
			DefaultAgent: "chat_agent",
			Settings: map[string]interface{}{
				"max_iterations": 10,
				"timeout":        "30s",
			},
		},
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetLogLevel returns the slog.Level based on the config setting
func (c *Config) GetLogLevel() slog.Level {
	switch c.App.LogLevel {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ValidateConfig validates the entire configuration structure
func (c *Config) ValidateConfig() error {
	// Validate App config
	if c.App.Name == "" {
		return fmt.Errorf("app.name cannot be empty")
	}
	if c.App.Version == "" {
		return fmt.Errorf("app.version cannot be empty")
	}

	// Validate LLM config
	if err := c.ValidateLLMConfig(); err != nil {
		return fmt.Errorf("LLM config validation failed: %w", err)
	}

	// Validate UI config
	if err := c.ValidateUIConfig(); err != nil {
		return fmt.Errorf("UI config validation failed: %w", err)
	}

	return nil
}

// ValidateLLMConfig validates LLM provider configurations
func (c *Config) ValidateLLMConfig() error {
	if c.LLM.Provider == "" {
		return fmt.Errorf("llm.provider cannot be empty")
	}

	// Validate that the current provider is in available providers
	found := false
	for _, provider := range c.LLM.AvailableProviders {
		if provider == c.LLM.Provider {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("current provider '%s' not found in available_providers", c.LLM.Provider)
	}

	// Validate provider-specific configurations
	switch c.LLM.Provider {
	case "ollama":
		if c.LLM.Ollama.BaseURL == "" {
			return fmt.Errorf("ollama.base_url cannot be empty")
		}
		if c.LLM.Ollama.DefaultModel == "" {
			return fmt.Errorf("ollama.default_model cannot be empty")
		}
	case "openai":
		if c.LLM.OpenAI.BaseURL == "" {
			return fmt.Errorf("openai.base_url cannot be empty")
		}
		if c.LLM.OpenAI.DefaultModel == "" {
			return fmt.Errorf("openai.default_model cannot be empty")
		}
		// API key can be from environment variable, so we don't validate it here
	case "eino":
		if c.LLM.Eino.DefaultModel == "" {
			return fmt.Errorf("eino.default_model cannot be empty")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", c.LLM.Provider)
	}

	return nil
}

// ValidateUIConfig validates UI configuration values
func (c *Config) ValidateUIConfig() error {
	// Validate numeric values
	if err := validation.ValidateUIValues(
		c.UI.WindowWidth,
		c.UI.WindowHeight,
		c.UI.MaxMessages,
		c.UI.FontSize,
		c.UI.SidebarWidth,
	); err != nil {
		return err
	}

	// Validate theme
	validThemes := []string{"light", "dark", "auto"}
	themeValid := false
	for _, theme := range validThemes {
		if c.UI.Theme == theme {
			themeValid = true
			break
		}
	}
	if !themeValid {
		return fmt.Errorf("ui.theme must be one of: %v", validThemes)
	}

	return nil
}

// ReloadConfig reloads configuration from file without restart
func ReloadConfig(configPath string) (*Config, error) {
	return LoadConfig(configPath)
}

// SaveConfig saves the current configuration to file
func (c *Config) SaveConfig(configPath string) error {
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}
