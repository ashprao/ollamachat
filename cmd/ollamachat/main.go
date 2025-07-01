package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ashprao/ollamachat/internal/app"
)

func main() {
	// Parse command line flags
	var configPath = flag.String("config", "", "Path to configuration file (default: configs/config.yaml)")
	var logLevel = flag.String("log-level", "", "Log level (debug, info, warn, error)")
	var storagePath = flag.String("storage", "", "Storage directory path")
	var providerType = flag.String("provider", "", "LLM provider type (ollama)")
	var baseURL = flag.String("base-url", "", "Base URL for LLM provider")
	var version = flag.Bool("version", false, "Show version information")
	var help = flag.Bool("help", false, "Show help information")

	flag.Parse()

	// Show version
	if *version {
		fmt.Println("OllamaChat v1.0.0")
		fmt.Println("A modern chat interface for Ollama")
		os.Exit(0)
	}

	// Show help
	if *help {
		showHelp()
		os.Exit(0)
	}

	// Create application configuration
	appConfig := app.AppConfig{
		ConfigPath:   *configPath,
		LogLevel:     *logLevel,
		StoragePath:  *storagePath,
		ProviderType: *providerType,
		BaseURL:      *baseURL,
	}

	// Create and run application
	application, err := app.New(appConfig)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Graceful shutdown handling could be added here
	defer func() {
		if err := application.Shutdown(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	// Run the application
	if err := application.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func showHelp() {
	fmt.Println("OllamaChat - A modern chat interface for Ollama")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ollamachat [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -config string")
	fmt.Println("        Path to configuration file (default: configs/config.yaml)")
	fmt.Println("  -log-level string")
	fmt.Println("        Log level: debug, info, warn, error (default: info)")
	fmt.Println("  -storage string")
	fmt.Println("        Storage directory path (default: data)")
	fmt.Println("  -provider string")
	fmt.Println("        LLM provider type: ollama (default: ollama)")
	fmt.Println("  -base-url string")
	fmt.Println("        Base URL for LLM provider (default: http://localhost:11434)")
	fmt.Println("  -version")
	fmt.Println("        Show version information")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ollamachat")
	fmt.Println("  ollamachat -config custom-config.yaml")
	fmt.Println("  ollamachat -log-level debug -storage /tmp/chat-data")
	fmt.Println("  ollamachat -base-url http://192.168.1.100:11434")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/ashprao/ollamachat")
}
