package models

import "time"

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
	Model     string        `json:"model"`
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
		ID:        generateSessionID(),
		Name:      name,
		Messages:  []ChatMessage{},
		CreatedAt: now,
		UpdatedAt: now,
		Model:     model,
	}
}

// AddMessage adds a new message to the chat session and updates the timestamp
func (cs *ChatSession) AddMessage(message ChatMessage) {
	cs.Messages = append(cs.Messages, message)
	cs.UpdatedAt = time.Now()
}

// generateSessionID generates a simple session ID (placeholder implementation)
func generateSessionID() string {
	return time.Now().Format("20060102-150405")
}
