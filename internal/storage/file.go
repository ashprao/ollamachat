package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

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

	// For backward compatibility, if this is the "default" session, save to legacy location
	if session.ID == "default" || session.ID == "" {
		return fs.saveLegacyChatHistory(session.Messages)
	}

	// Save to new session-based structure
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

	// For backward compatibility, try to load legacy format first
	if sessionID == "default" || sessionID == "" {
		messages, err := fs.loadLegacyChatHistory()
		if err != nil {
			fs.logger.Error("Failed to load legacy chat history", "error", err)
			return models.ChatSession{}, err
		}

		// Convert legacy messages to new session format
		session := models.NewChatSession("Default Session", "")
		session.ID = "default"
		session.Messages = messages
		return session, nil
	}

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

	sessionsDir := filepath.Join(fs.basePath, "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		// Check for legacy chat history and create default session if exists
		if messages, err := fs.loadLegacyChatHistory(); err == nil && len(messages) > 0 {
			defaultSession := models.NewChatSession("Default Session", "")
			defaultSession.ID = "default"
			defaultSession.Messages = messages
			fs.logger.Info("Found legacy chat history, returning as default session")
			return []models.ChatSession{defaultSession}, nil
		}
		return []models.ChatSession{}, nil
	}

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		fs.logger.Error("Failed to read sessions directory", "error", err)
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []models.ChatSession
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

	fs.logger.Info("Successfully listed chat sessions", "count", len(sessions))
	return sessions, nil
}

// DeleteChatSession deletes a chat session
func (fs *FileStorage) DeleteChatSession(ctx context.Context, sessionID string) error {
	fs.logger.Info("Deleting chat session", "session_id", sessionID)

	if sessionID == "default" {
		fs.logger.Warn("Cannot delete default session")
		return fmt.Errorf("cannot delete default session")
	}

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

// Legacy support methods for backward compatibility with existing chat_history.json

// saveLegacyChatHistory saves messages in the legacy format for backward compatibility
func (fs *FileStorage) saveLegacyChatHistory(messages []models.ChatMessage) error {
	if fs.app == nil {
		fs.logger.Warn("No app instance available for legacy storage")
		return fmt.Errorf("no app instance available for legacy storage")
	}

	root := fs.app.Storage().RootURI()
	uri, err := storage.Child(root, "chat_history.json")
	if err != nil {
		fs.logger.Error("Failed to get legacy chat history URI", "error", err)
		return fmt.Errorf("failed to get legacy chat history URI: %w", err)
	}

	file, err := storage.Writer(uri)
	if err != nil {
		fs.logger.Error("Failed to create legacy chat history writer", "error", err)
		return fmt.Errorf("failed to create legacy chat history writer: %w", err)
	}
	defer file.Close()

	// Convert new format messages to legacy format
	legacyMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		legacyMessages[i] = map[string]interface{}{
			"sender":    msg.Sender,
			"content":   msg.Content,
			"timestamp": msg.Timestamp.Format(time.RFC3339),
		}
	}

	data, err := json.Marshal(legacyMessages)
	if err != nil {
		fs.logger.Error("Failed to marshal legacy messages", "error", err)
		return fmt.Errorf("failed to marshal legacy messages: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		fs.logger.Error("Failed to write legacy chat history", "error", err)
		return fmt.Errorf("failed to write legacy chat history: %w", err)
	}

	fs.logger.Info("Successfully saved legacy chat history", "message_count", len(messages))
	return nil
}

// loadLegacyChatHistory loads messages from the legacy chat_history.json format
func (fs *FileStorage) loadLegacyChatHistory() ([]models.ChatMessage, error) {
	if fs.app == nil {
		fs.logger.Warn("No app instance available for legacy storage")
		return []models.ChatMessage{}, nil
	}

	root := fs.app.Storage().RootURI()
	uri, err := storage.Child(root, "chat_history.json")
	if err != nil {
		fs.logger.Error("Failed to get legacy chat history URI", "error", err)
		return nil, fmt.Errorf("failed to get legacy chat history URI: %w", err)
	}

	file, err := storage.Reader(uri)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") {
			fs.logger.Info("Legacy chat history file not found")
			return []models.ChatMessage{}, nil
		}
		fs.logger.Error("Failed to open legacy chat history reader", "error", err)
		return nil, fmt.Errorf("failed to open legacy chat history reader: %w", err)
	}
	defer file.Close()

	var legacyMessages []map[string]interface{}
	if err := json.NewDecoder(file).Decode(&legacyMessages); err != nil {
		fs.logger.Error("Failed to decode legacy chat history", "error", err)
		return nil, fmt.Errorf("failed to decode legacy chat history: %w", err)
	}

	// Convert legacy format to new format
	messages := make([]models.ChatMessage, len(legacyMessages))
	for i, legacyMsg := range legacyMessages {
		// Handle timestamp conversion
		var timestamp time.Time
		if tsStr, ok := legacyMsg["timestamp"].(string); ok && tsStr != "" {
			if parsed, err := time.Parse(time.RFC3339, tsStr); err == nil {
				timestamp = parsed
			} else {
				timestamp = time.Now()
			}
		} else {
			timestamp = time.Now()
		}

		messages[i] = models.ChatMessage{
			Sender:    getString(legacyMsg, "sender"),
			Content:   getString(legacyMsg, "content"),
			Timestamp: timestamp,
		}
	}

	fs.logger.Info("Successfully loaded legacy chat history", "message_count", len(messages))
	return messages, nil
}

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
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
