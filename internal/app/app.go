package app

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/ashprao/ollamachat/internal/config"
	"github.com/ashprao/ollamachat/internal/llm"
	"github.com/ashprao/ollamachat/internal/storage"
	"github.com/ashprao/ollamachat/internal/ui"
	"github.com/ashprao/ollamachat/pkg/logger"
)

// App represents the main application container with all dependencies
type App struct {
	// Core dependencies
	config   *config.Config
	logger   *logger.Logger
	provider llm.Provider
	storage  storage.Storage

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
				Provider: "ollama",
				Ollama: config.OllamaConfig{
					BaseURL:      "http://localhost:11434",
					DefaultModel: "llama3.2:latest",
				},
			},
			UI: config.UIConfig{
				WindowWidth:  600,
				WindowHeight: 700,
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

	// Initialize LLM provider
	providerType := appConfig.ProviderType
	if providerType == "" {
		providerType = cfg.LLM.Provider
	}

	baseURL := appConfig.BaseURL
	if baseURL == "" {
		baseURL = cfg.LLM.Ollama.BaseURL
	}

	var provider llm.Provider
	switch providerType {
	case "ollama", "":
		ollamaFactory := llm.NewDefaultOllamaProviderFactory(logger)
		providerConfig := llm.ProviderConfig{
			Type:    "ollama",
			BaseURL: baseURL,
		}
		provider, err = ollamaFactory.CreateProvider(providerConfig)
		if err != nil {
			logger.Error("Failed to create Ollama provider", "error", err)
			return nil, fmt.Errorf("failed to create Ollama provider: %w", err)
		}
	default:
		logger.Error("Unsupported provider type", "provider", providerType)
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	logger.Info("LLM provider initialized", "provider", provider.GetName(), "base_url", baseURL)

	// Ping storage to ensure it's working
	ctx := context.Background()
	if err := stor.Ping(ctx); err != nil {
		logger.Warn("Storage ping failed", "error", err)
	}

	// Create ChatUI
	chatUI := ui.NewChatUI(window, provider, stor, logger)

	app := &App{
		config:   cfg,
		logger:   logger,
		provider: provider,
		storage:  stor,
		fyneApp:  fyneApp,
		window:   window,
		chatUI:   chatUI,
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
