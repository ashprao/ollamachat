package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"

	"github.com/ashprao/ollamachat/internal/models"
	"github.com/ashprao/ollamachat/pkg/logger"
)

// FileStorage implements the Storage interface using local file system
type FileStorage struct {
	basePath string
	app      fyne.App
	logger   *logger.Logger
}

// NewFileStorage creates a new file-based storage implementation
func NewFileStorage(basePath string, app fyne.App, logger *logger.Logger) (*FileStorage, error) {
	if basePath == "" {
		basePath = "data"
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	fs := &FileStorage{
		basePath: basePath,
		app:      app,
		logger:   logger.WithComponent("file-storage"),
	}

	fs.logger.Info("Initialized file storage", "base_path", basePath)
	return fs, nil
}

// SaveChatSession saves a chat session to file
func (fs *FileStorage) SaveChatSession(ctx context.Context, session models.ChatSession) error {
	fs.logger.Info("Saving chat session", "session_id", session.ID, "message_count", len(session.Messages))

	// Save to session-based structure
	sessionPath := filepath.Join(fs.basePath, "sessions", fmt.Sprintf("%s.json", session.ID))
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0755); err != nil {
		fs.logger.Error("Failed to create sessions directory", "error", err)
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Update the session timestamp
	session.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		fs.logger.Error("Failed to marshal session", "session_id", session.ID, "error", err)
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(sessionPath, data, 0644); err != nil {
		fs.logger.Error("Failed to write session file", "session_id", session.ID, "error", err)
		return fmt.Errorf("failed to write session file: %w", err)
	}

	fs.logger.Info("Successfully saved chat session", "session_id", session.ID)
	return nil
}

// LoadChatSession loads a chat session from file
func (fs *FileStorage) LoadChatSession(ctx context.Context, sessionID string) (models.ChatSession, error) {
	fs.logger.Info("Loading chat session", "session_id", sessionID)

	sessionPath := filepath.Join(fs.basePath, "sessions", fmt.Sprintf("%s.json", sessionID))
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			fs.logger.Warn("Session not found", "session_id", sessionID)
			return models.ChatSession{}, fmt.Errorf("session not found: %s", sessionID)
		}
		fs.logger.Error("Failed to read session file", "session_id", sessionID, "error", err)
		return models.ChatSession{}, fmt.Errorf("failed to read session file: %w", err)
	}

	var session models.ChatSession
	if err := json.Unmarshal(data, &session); err != nil {
		fs.logger.Error("Failed to unmarshal session", "session_id", sessionID, "error", err)
		return models.ChatSession{}, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	fs.logger.Info("Successfully loaded chat session", "session_id", sessionID, "message_count", len(session.Messages))
	return session, nil
}

// ListChatSessions lists all available chat sessions
func (fs *FileStorage) ListChatSessions(ctx context.Context) ([]models.ChatSession, error) {
	fs.logger.Info("Listing chat sessions")

	var sessions []models.ChatSession

	// Check for session-based storage
	sessionsDir := filepath.Join(fs.basePath, "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		// No sessions directory, return empty list
		fs.logger.Info("No sessions directory found, returning empty list")
		return sessions, nil
	}

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		fs.logger.Error("Failed to read sessions directory", "error", err)
		return sessions, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	// Load all sessions from the sessions directory
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), ".json")
		session, err := fs.LoadChatSession(ctx, sessionID)
		if err != nil {
			fs.logger.Warn("Failed to load session", "session_id", sessionID, "error", err)
			continue
		}

		sessions = append(sessions, session)
	}

	// Sort sessions by updated time (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	fs.logger.Info("Successfully listed chat sessions", "count", len(sessions))
	return sessions, nil
}

// DeleteChatSession deletes a chat session
func (fs *FileStorage) DeleteChatSession(ctx context.Context, sessionID string) error {
	fs.logger.Info("Deleting chat session", "session_id", sessionID)

	sessionPath := filepath.Join(fs.basePath, "sessions", fmt.Sprintf("%s.json", sessionID))
	if err := os.Remove(sessionPath); err != nil {
		if os.IsNotExist(err) {
			fs.logger.Warn("Session not found for deletion", "session_id", sessionID)
			return fmt.Errorf("session not found: %s", sessionID)
		}
		fs.logger.Error("Failed to delete session file", "session_id", sessionID, "error", err)
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	fs.logger.Info("Successfully deleted chat session", "session_id", sessionID)
	return nil
}

// SaveAppPreferences saves application preferences
func (fs *FileStorage) SaveAppPreferences(ctx context.Context, prefs AppPreferences) error {
	fs.logger.Info("Saving application preferences")

	prefsPath := filepath.Join(fs.basePath, "preferences.json")
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		fs.logger.Error("Failed to marshal preferences", "error", err)
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	if err := os.WriteFile(prefsPath, data, 0644); err != nil {
		fs.logger.Error("Failed to write preferences file", "error", err)
		return fmt.Errorf("failed to write preferences file: %w", err)
	}

	fs.logger.Info("Successfully saved application preferences")
	return nil
}

// LoadAppPreferences loads application preferences
func (fs *FileStorage) LoadAppPreferences(ctx context.Context) (AppPreferences, error) {
	fs.logger.Info("Loading application preferences")

	prefsPath := filepath.Join(fs.basePath, "preferences.json")
	data, err := os.ReadFile(prefsPath)
	if err != nil {
		if os.IsNotExist(err) {
			fs.logger.Info("Preferences file not found, returning defaults")
			return NewDefaultAppPreferences(), nil
		}
		fs.logger.Error("Failed to read preferences file", "error", err)
		return AppPreferences{}, fmt.Errorf("failed to read preferences file: %w", err)
	}

	var prefs AppPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		fs.logger.Error("Failed to unmarshal preferences", "error", err)
		return AppPreferences{}, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	fs.logger.Info("Successfully loaded application preferences")
	return prefs, nil
}

// SaveMCPServers saves MCP server configurations (placeholder implementation)
func (fs *FileStorage) SaveMCPServers(ctx context.Context, servers []models.MCPServer) error {
	fs.logger.Info("Saving MCP servers", "count", len(servers))

	mcpPath := filepath.Join(fs.basePath, "mcp_servers.json")
	data, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		fs.logger.Error("Failed to marshal MCP servers", "error", err)
		return fmt.Errorf("failed to marshal MCP servers: %w", err)
	}

	if err := os.WriteFile(mcpPath, data, 0644); err != nil {
		fs.logger.Error("Failed to write MCP servers file", "error", err)
		return fmt.Errorf("failed to write MCP servers file: %w", err)
	}

	fs.logger.Info("Successfully saved MCP servers")
	return nil
}

// LoadMCPServers loads MCP server configurations (placeholder implementation)
func (fs *FileStorage) LoadMCPServers(ctx context.Context) ([]models.MCPServer, error) {
	fs.logger.Info("Loading MCP servers")

	mcpPath := filepath.Join(fs.basePath, "mcp_servers.json")
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		if os.IsNotExist(err) {
			fs.logger.Info("MCP servers file not found, returning empty list")
			return []models.MCPServer{}, nil
		}
		fs.logger.Error("Failed to read MCP servers file", "error", err)
		return nil, fmt.Errorf("failed to read MCP servers file: %w", err)
	}

	var servers []models.MCPServer
	if err := json.Unmarshal(data, &servers); err != nil {
		fs.logger.Error("Failed to unmarshal MCP servers", "error", err)
		return nil, fmt.Errorf("failed to unmarshal MCP servers: %w", err)
	}

	fs.logger.Info("Successfully loaded MCP servers", "count", len(servers))
	return servers, nil
}

// SaveAgentConfig saves agent configuration (placeholder implementation)
func (fs *FileStorage) SaveAgentConfig(ctx context.Context, config models.AgentConfig) error {
	fs.logger.Info("Saving agent configuration")

	agentPath := filepath.Join(fs.basePath, "agent_config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fs.logger.Error("Failed to marshal agent config", "error", err)
		return fmt.Errorf("failed to marshal agent config: %w", err)
	}

	if err := os.WriteFile(agentPath, data, 0644); err != nil {
		fs.logger.Error("Failed to write agent config file", "error", err)
		return fmt.Errorf("failed to write agent config file: %w", err)
	}

	fs.logger.Info("Successfully saved agent configuration")
	return nil
}

// LoadAgentConfig loads agent configuration (placeholder implementation)
func (fs *FileStorage) LoadAgentConfig(ctx context.Context) (models.AgentConfig, error) {
	fs.logger.Info("Loading agent configuration")

	agentPath := filepath.Join(fs.basePath, "agent_config.json")
	data, err := os.ReadFile(agentPath)
	if err != nil {
		if os.IsNotExist(err) {
			fs.logger.Info("Agent config file not found, returning defaults")
			return models.AgentConfig{
				Enabled:            false,
				DefaultFramework:   "eino",
				MaxConcurrent:      1,
				Timeout:            30,
				EnableMCPTools:     false,
				EnableBuiltinTools: true,
				LogLevel:           "info",
				EnableTracing:      false,
				UpdatedAt:          time.Now(),
			}, nil
		}
		fs.logger.Error("Failed to read agent config file", "error", err)
		return models.AgentConfig{}, fmt.Errorf("failed to read agent config file: %w", err)
	}

	var config models.AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		fs.logger.Error("Failed to unmarshal agent config", "error", err)
		return models.AgentConfig{}, fmt.Errorf("failed to unmarshal agent config: %w", err)
	}

	fs.logger.Info("Successfully loaded agent configuration")
	return config, nil
}

// Close closes the storage (no-op for file storage)
func (fs *FileStorage) Close() error {
	fs.logger.Info("Closing file storage")
	return nil
}

// Ping checks if the storage is accessible
func (fs *FileStorage) Ping(ctx context.Context) error {
	// Check if base directory is accessible
	if _, err := os.Stat(fs.basePath); err != nil {
		fs.logger.Error("Storage ping failed", "error", err)
		return fmt.Errorf("storage ping failed: %w", err)
	}
	return nil
}

// DefaultFileStorageFactory creates file storage instances
type DefaultFileStorageFactory struct {
	app    fyne.App
	logger *logger.Logger
}

// NewDefaultFileStorageFactory creates a new factory instance
func NewDefaultFileStorageFactory(app fyne.App, logger *logger.Logger) *DefaultFileStorageFactory {
	return &DefaultFileStorageFactory{
		app:    app,
		logger: logger,
	}
}

// CreateStorage creates a file storage instance from configuration
func (f *DefaultFileStorageFactory) CreateStorage(config StorageConfig) (Storage, error) {
	if config.Type != "file" {
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}

	basePath := config.BasePath
	if basePath == "" {
		basePath = "data"
	}

	return NewFileStorage(basePath, f.app, f.logger)
}

// SupportedTypes returns the list of supported storage types
func (f *DefaultFileStorageFactory) SupportedTypes() []string {
	return []string{"file"}
}
