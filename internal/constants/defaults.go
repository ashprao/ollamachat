package constants

// Default values shared across the application
const (
	// Default model name for LLM operations
	DefaultModelName = "llama3.2:latest"

	// Default provider name
	DefaultProvider = "ollama"

	// Default temperature for LLM operations
	DefaultTemperature = 0.7

	// Default max messages for context
	DefaultMaxMessages = 10

	// Default max tokens for LLM responses
	DefaultMaxTokens = 2048

	// Default timeout for LLM requests (in seconds)
	DefaultTimeoutSeconds = 30

	// UI dimension defaults
	DefaultWindowWidth  = 800
	DefaultWindowHeight = 700
	DefaultFontSize     = 12
	DefaultSidebarWidth = 200

	// Date/time format for timestamps
	TimestampFormat = "Jan 2 15:04:05"
)

// Message titles
const (
	UserMessageTitle = "You:"
	LLMMessageTitle  = "LLM:"
)
