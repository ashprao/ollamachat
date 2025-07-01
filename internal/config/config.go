package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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
	Provider string                 `yaml:"provider"` // "ollama", "openai", "eino"
	Ollama   OllamaConfig           `yaml:"ollama"`
	OpenAI   OpenAIConfig           `yaml:"openai"`
	Eino     EinoConfig             `yaml:"eino"`
	Settings map[string]interface{} `yaml:"settings"`
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
	WindowWidth  int `yaml:"window_width"`
	WindowHeight int `yaml:"window_height"`
	MaxMessages  int `yaml:"max_messages"`
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
			Provider: "ollama",
			Ollama: OllamaConfig{
				BaseURL:      "http://localhost:11434",
				DefaultModel: "llama3.2:latest",
			},
			OpenAI: OpenAIConfig{
				BaseURL:      "https://api.openai.com/v1",
				DefaultModel: "gpt-3.5-turbo",
			},
			Eino: EinoConfig{
				DefaultModel: "llama3.2:latest",
				Settings:     map[string]string{},
			},
			Settings: map[string]interface{}{
				"timeout_seconds": 30,
				"max_tokens":      2048,
			},
		},
		UI: UIConfig{
			WindowWidth:  600,
			WindowHeight: 700,
			MaxMessages:  10,
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
