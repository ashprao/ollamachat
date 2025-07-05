package models

import (
	"time"

	"github.com/ashprao/ollamachat/internal/constants"
)

// Model represents an LLM model
type Model struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ChatMessage represents a single message in a chat conversation
type ChatMessage struct {
	Sender    string    `json:"sender"` // "user" or "llm"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"` // Changed from string to time.Time
}

// ChatSession represents a complete chat session with multiple messages
type ChatSession struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`

	// Session-specific preferences
	Model       string  `json:"model"`        // Selected model for this session
	Provider    string  `json:"provider"`     // Selected provider for this session
	MaxMessages int     `json:"max_messages"` // Max context messages for this session
	Temperature float64 `json:"temperature"`  // Model temperature setting
}

// NewChatMessage creates a new chat message with current timestamp
func NewChatMessage(sender, content string) ChatMessage {
	return ChatMessage{
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewChatSession creates a new chat session with a unique ID
func NewChatSession(name, model string) ChatSession {
	now := time.Now()
	return ChatSession{
		ID:          generateSessionID(),
		Name:        name,
		Messages:    []ChatMessage{},
		CreatedAt:   now,
		UpdatedAt:   now,
		Model:       model,
		Provider:    constants.DefaultProvider,    // Default provider
		MaxMessages: constants.DefaultMaxMessages, // Default max context messages
		Temperature: constants.DefaultTemperature, // Default temperature
	}
}

// NewChatSessionWithConfig creates a new chat session with configurable defaults
func NewChatSessionWithConfig(name, model string, maxMessages int, temperature float64) ChatSession {
	now := time.Now()
	return ChatSession{
		ID:          generateSessionID(),
		Name:        name,
		Messages:    []ChatMessage{},
		CreatedAt:   now,
		UpdatedAt:   now,
		Model:       model,
		Provider:    "ollama", // Default provider
		MaxMessages: maxMessages,
		Temperature: temperature,
	}
}

// AddMessage adds a new message to the chat session and updates the timestamp
func (cs *ChatSession) AddMessage(message ChatMessage) {
	cs.Messages = append(cs.Messages, message)
	cs.UpdatedAt = time.Now()
}

// UpdateSessionSettings updates the session-specific settings
func (cs *ChatSession) UpdateSessionSettings(model, provider string, maxMessages int, temperature float64) {
	cs.Model = model
	cs.Provider = provider
	cs.MaxMessages = maxMessages
	cs.Temperature = temperature
	cs.UpdatedAt = time.Now()
}

// GetContextMessages returns the last N messages for context, based on session settings
func (cs *ChatSession) GetContextMessages() []ChatMessage {
	if len(cs.Messages) <= cs.MaxMessages {
		return cs.Messages
	}
	return cs.Messages[len(cs.Messages)-cs.MaxMessages:]
}

// generateSessionID generates a simple session ID (placeholder implementation)
func generateSessionID() string {
	return time.Now().Format("20060102-150405")
}
