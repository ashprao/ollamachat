package storage

import (
	"context"

	"github.com/ashprao/ollamachat/internal/models"
)

// Storage defines the interface for data persistence operations
type Storage interface {
	// Session Management
	SaveChatSession(ctx context.Context, session models.ChatSession) error
	LoadChatSession(ctx context.Context, sessionID string) (models.ChatSession, error)
	ListChatSessions(ctx context.Context) ([]models.ChatSession, error)
	DeleteChatSession(ctx context.Context, sessionID string) error

	// Application Preferences
	SaveAppPreferences(ctx context.Context, prefs AppPreferences) error
	LoadAppPreferences(ctx context.Context) (AppPreferences, error)

	// MCP Server Configuration
	SaveMCPServers(ctx context.Context, servers []models.MCPServer) error
	LoadMCPServers(ctx context.Context) ([]models.MCPServer, error)

	// Agent Configuration
	SaveAgentConfig(ctx context.Context, config models.AgentConfig) error
	LoadAgentConfig(ctx context.Context) (models.AgentConfig, error)

	// Health and Management
	Close() error
	Ping(ctx context.Context) error
}

// AppPreferences holds user application preferences
type AppPreferences struct {
	// UI Preferences
	WindowWidth      int    `json:"window_width"`
	WindowHeight     int    `json:"window_height"`
	Theme            string `json:"theme"` // "light", "dark", "auto"
	FontSize         int    `json:"font_size"`
	MaxHistoryLength int    `json:"max_history_length"`

	// Chat Preferences
	DefaultModel    string `json:"default_model"`
	DefaultProvider string `json:"default_provider"`
	AutoSaveHistory bool   `json:"auto_save_history"`
	EnableMarkdown  bool   `json:"enable_markdown"`
	ShowTimestamps  bool   `json:"show_timestamps"`

	// Advanced Preferences
	MaxContextLength  int    `json:"max_context_length"`
	EnableToolCalling bool   `json:"enable_tool_calling"`
	EnableMCPServers  bool   `json:"enable_mcp_servers"`
	EnableAgents      bool   `json:"enable_agents"`
	LogLevel          string `json:"log_level"`
}

// StorageConfig holds configuration for storage implementations
type StorageConfig struct {
	Type     string                 `json:"type"`      // "file", "sqlite", "memory"
	BasePath string                 `json:"base_path"` // Base directory for file storage
	Settings map[string]interface{} `json:"settings"`  // Implementation-specific settings
}

// NewDefaultAppPreferences creates default application preferences
func NewDefaultAppPreferences() AppPreferences {
	return AppPreferences{
		// UI Preferences
		WindowWidth:      600,
		WindowHeight:     700,
		Theme:            "auto",
		FontSize:         12,
		MaxHistoryLength: 100,

		// Chat Preferences
		DefaultModel:    "llama3.2:latest",
		DefaultProvider: "ollama",
		AutoSaveHistory: true,
		EnableMarkdown:  true,
		ShowTimestamps:  false,

		// Advanced Preferences
		MaxContextLength:  10,
		EnableToolCalling: false,
		EnableMCPServers:  false,
		EnableAgents:      false,
		LogLevel:          "info",
	}
}

// StorageFactory creates storage implementations based on configuration
type StorageFactory interface {
	CreateStorage(config StorageConfig) (Storage, error)
	SupportedTypes() []string
}
