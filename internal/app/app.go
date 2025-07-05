package app

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"

	"github.com/ashprao/ollamachat/internal/config"
	"github.com/ashprao/ollamachat/internal/constants"
	"github.com/ashprao/ollamachat/internal/llm"
	"github.com/ashprao/ollamachat/internal/storage"
	"github.com/ashprao/ollamachat/internal/ui"
	"github.com/ashprao/ollamachat/pkg/logger"
)

// CustomTheme wraps the default theme to apply custom font size
type CustomTheme struct {
	fyne.Theme
	fontSize int
}

// Size returns the custom font size for text
func (t *CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		if t.fontSize > 0 {
			return float32(t.fontSize)
		}
	}
	return t.Theme.Size(name)
}

// NewCustomTheme creates a theme with custom font size
func NewCustomTheme(fontSize int) fyne.Theme {
	return &CustomTheme{
		Theme:    theme.DefaultTheme(),
		fontSize: fontSize,
	}
}

// App represents the main application container with all dependencies
type App struct {
	// Core dependencies
	config          *config.Config
	configPath      string // Add config path for saving
	logger          *logger.Logger
	provider        llm.Provider
	providerFactory *llm.DefaultProviderFactory
	storage         storage.Storage

	// UI components
	fyneApp fyne.App
	window  fyne.Window
	chatUI  *ui.ChatUI

	// Future MCP client placeholder
	// mcpClient mcp.Client

	// Application state
	isRunning bool
}

// AppConfig holds configuration for creating the application
type AppConfig struct {
	ConfigPath   string
	LogLevel     string
	StoragePath  string
	ProviderType string
	BaseURL      string
}

// New creates a new application instance with all dependencies
func New(appConfig AppConfig) (*App, error) {
	// Initialize logger first
	logLevel := appConfig.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}

	// Convert string log level to slog.Level
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := logger.NewLogger(level)

	logger.Info("Starting application initialization")

	// Load configuration
	configPath := appConfig.ConfigPath
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Warn("Failed to load config, creating default", "error", err)
		// Create a default config structure
		cfg = &config.Config{
			App: config.AppConfig{
				Name:     "OllamaChat",
				Version:  "1.0.0",
				LogLevel: "info",
			},
			LLM: config.LLMConfig{
				Provider: constants.DefaultProvider,
				Ollama: config.OllamaConfig{
					BaseURL:      "http://localhost:11434",
					DefaultModel: constants.DefaultModelName,
				},
			},
			UI: config.UIConfig{
				WindowWidth:  constants.DefaultWindowWidth, // Increased from 600 to accommodate session sidebar
				WindowHeight: constants.DefaultWindowHeight,
			},
		}
	}

	logger.Info("Configuration loaded", "config_path", configPath)

	// Create Fyne app
	fyneApp := app.NewWithID("github.com.ashprao.ollamachat")
	window := fyneApp.NewWindow("Ollama Chat Interface")
	window.Resize(fyne.NewSize(
		float32(cfg.UI.WindowWidth),
		float32(cfg.UI.WindowHeight),
	))

	// Initialize storage
	storagePath := appConfig.StoragePath
	if storagePath == "" {
		storagePath = "data" // Default storage path
	}

	storageFactory := storage.NewDefaultFileStorageFactory(fyneApp, logger)
	storageConfig := storage.StorageConfig{
		Type:     "file",
		BasePath: storagePath,
	}

	stor, err := storageFactory.CreateStorage(storageConfig)
	if err != nil {
		logger.Error("Failed to create storage", "error", err)
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	logger.Info("Storage initialized", "type", "file", "path", storagePath)

	// Initialize LLM provider factory
	providerFactory := llm.NewDefaultProviderFactory(cfg, logger)

	// Get provider type from config or command line
	providerType := appConfig.ProviderType
	if providerType == "" {
		providerType = cfg.LLM.Provider
	}

	// Validate provider configuration
	if err := providerFactory.ValidateProviderConfig(providerType); err != nil {
		logger.Error("Provider configuration validation failed", "provider", providerType, "error", err)
		return nil, fmt.Errorf("provider configuration validation failed: %w", err)
	}

	// Create provider instance
	provider, err := providerFactory.CreateProviderFromConfig(providerType)
	if err != nil {
		logger.Error("Failed to create LLM provider", "provider", providerType, "error", err)
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	logger.Info("LLM provider initialized", "provider", provider.GetName(), "provider_type", providerType)

	// Ping storage to ensure it's working
	ctx := context.Background()
	if err := stor.Ping(ctx); err != nil {
		logger.Warn("Storage ping failed", "error", err)
	}

	// Create app instance first (without chatUI)
	app := &App{
		config:          cfg,
		configPath:      configPath, // Store config path for saving
		logger:          logger,
		provider:        provider,
		providerFactory: providerFactory,
		storage:         stor,
		fyneApp:         fyneApp,
		window:          window,
	}

	// Create ChatUI with provider information and app reference
	chatUI := ui.NewChatUI(window, provider, stor, logger, providerFactory.GetAvailableProviders(), providerType, cfg, app)

	// Set the chatUI in the app
	app.chatUI = chatUI

	// Apply font size from config
	if cfg.UI.FontSize > 0 {
		// Set custom theme with font size
		fyneApp.Settings().SetTheme(NewCustomTheme(cfg.UI.FontSize))
		logger.Info("Applied custom font size", "size", cfg.UI.FontSize)
	}

	logger.Info("Application initialization completed successfully")
	return app, nil
}

// Run starts the application
func (a *App) Run() error {
	if a.isRunning {
		return fmt.Errorf("application is already running")
	}

	a.logger.Info("Starting application")
	a.isRunning = true

	// Initialize the chat UI
	if err := a.chatUI.Initialize(); err != nil {
		a.logger.Error("Failed to initialize chat UI", "error", err)
		return fmt.Errorf("failed to initialize chat UI: %w", err)
	}

	a.logger.Info("Chat UI initialized successfully")

	// Show window and run the application
	a.window.Show()
	a.fyneApp.Run()

	a.isRunning = false
	return nil
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() error {
	a.logger.Info("Shutting down application")

	if a.storage != nil {
		if err := a.storage.Close(); err != nil {
			a.logger.Error("Failed to close storage", "error", err)
		}
	}

	a.isRunning = false
	a.logger.Info("Application shutdown completed")
	return nil
}

// GetConfig returns the application configuration
func (a *App) GetConfig() *config.Config {
	return a.config
}

// GetLogger returns the application logger
func (a *App) GetLogger() *logger.Logger {
	return a.logger
}

// GetProvider returns the LLM provider
func (a *App) GetProvider() llm.Provider {
	return a.provider
}

// GetStorage returns the storage implementation
func (a *App) GetStorage() storage.Storage {
	return a.storage
}

// GetFyneApp returns the Fyne application instance
func (a *App) GetFyneApp() fyne.App {
	return a.fyneApp
}

// GetWindow returns the main window
func (a *App) GetWindow() fyne.Window {
	return a.window
}

// GetChatUI returns the chat UI instance
func (a *App) GetChatUI() *ui.ChatUI {
	return a.chatUI
}

// IsRunning returns whether the application is currently running
func (a *App) IsRunning() bool {
	return a.isRunning
}

// SwitchProvider switches to a different LLM provider
func (a *App) SwitchProvider(providerType string) error {
	a.logger.Info("Switching LLM provider", "from", a.provider.GetName(), "to", providerType)

	// Validate the new provider configuration
	if err := a.providerFactory.ValidateProviderConfig(providerType); err != nil {
		a.logger.Error("Provider configuration validation failed", "provider", providerType, "error", err)
		return fmt.Errorf("provider configuration validation failed: %w", err)
	}

	// Create new provider instance
	newProvider, err := a.providerFactory.CreateProviderFromConfig(providerType)
	if err != nil {
		a.logger.Error("Failed to create new provider", "provider", providerType, "error", err)
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Update the provider
	a.provider = newProvider

	// Update the config to reflect the new provider
	a.config.LLM.Provider = providerType

	// Update the UI with the new provider
	if a.chatUI != nil {
		a.chatUI.UpdateProvider(newProvider)
	}

	a.logger.Info("Successfully switched LLM provider", "provider", newProvider.GetName())
	return nil
}

// GetAvailableProviders returns the list of available providers
func (a *App) GetAvailableProviders() []string {
	return a.providerFactory.GetAvailableProviders()
}

// GetCurrentProviderType returns the currently configured provider type
func (a *App) GetCurrentProviderType() string {
	return a.config.LLM.Provider
}

// ReloadConfigFromFile reloads configuration from file and updates current settings
func (a *App) ReloadConfigFromFile(configPath string) error {
	a.logger.Info("Reloading configuration from file", "path", configPath)

	// Load new configuration
	newConfig, err := config.ReloadConfig(configPath)
	if err != nil {
		a.logger.Error("Failed to reload configuration", "error", err)
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// Validate new configuration
	if err := newConfig.ValidateConfig(); err != nil {
		a.logger.Error("New configuration validation failed", "error", err)
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Update logger level if it changed
	if newConfig.App.LogLevel != a.config.App.LogLevel {
		a.logger.Info("Updating log level", "old", a.config.App.LogLevel, "new", newConfig.App.LogLevel)
		// Note: Logger level change would require recreating the logger
		// For now, we'll just log the change
	}

	// Update window size if it changed
	if newConfig.UI.WindowWidth != a.config.UI.WindowWidth ||
		newConfig.UI.WindowHeight != a.config.UI.WindowHeight {
		a.logger.Info("Updating window size",
			"old_size", fmt.Sprintf("%dx%d", a.config.UI.WindowWidth, a.config.UI.WindowHeight),
			"new_size", fmt.Sprintf("%dx%d", newConfig.UI.WindowWidth, newConfig.UI.WindowHeight))
		a.window.Resize(fyne.NewSize(
			float32(newConfig.UI.WindowWidth),
			float32(newConfig.UI.WindowHeight),
		))
	}

	// Check if provider changed
	if newConfig.LLM.Provider != a.config.LLM.Provider {
		a.logger.Info("Provider configuration changed",
			"old", a.config.LLM.Provider,
			"new", newConfig.LLM.Provider)

		// Switch to new provider
		if err := a.SwitchProvider(newConfig.LLM.Provider); err != nil {
			a.logger.Error("Failed to switch to new provider", "error", err)
			return fmt.Errorf("failed to switch provider: %w", err)
		}
	}

	// Update the configuration
	a.config = newConfig

	// Update ChatUI with new config
	if a.chatUI != nil {
		a.chatUI.UpdateConfig(newConfig)
	}

	a.logger.Info("Configuration reloaded successfully")
	return nil
}

// SaveConfig saves the current configuration to file
func (a *App) SaveConfig() error {
	a.logger.Info("Saving configuration to file", "path", a.configPath)

	if err := a.config.SaveConfig(a.configPath); err != nil {
		a.logger.Error("Failed to save configuration", "error", err)
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	a.logger.Info("Configuration saved successfully")
	return nil
}

// GetConfigPath returns the current config file path
func (a *App) GetConfigPath() string {
	return a.configPath
}

// UpdateWindowSize updates the window size immediately
func (a *App) UpdateWindowSize(width, height int) {
	a.logger.Info("Updating window size", "width", width, "height", height)
	a.window.Resize(fyne.NewSize(float32(width), float32(height)))
}

// UpdateFontSize updates the font size by applying a new theme
func (a *App) UpdateFontSize(fontSize int) {
	a.logger.Info("Updating font size", "size", fontSize)
	if fontSize > 0 {
		a.fyneApp.Settings().SetTheme(NewCustomTheme(fontSize))
		a.logger.Info("Applied new font size theme", "size", fontSize)
	}
}
