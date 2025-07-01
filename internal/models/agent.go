package models

import "time"

// Agent represents an autonomous agent configuration
type Agent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`      // "chat", "workflow", "tool"
	Framework   string                 `json:"framework"` // "eino", "custom"
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// AgentMessage represents a message in an agent conversation
type AgentMessage struct {
	Role      string                 `json:"role"` // "user", "assistant", "system", "tool"
	Content   string                 `json:"content"`
	ToolCalls []ToolCall             `json:"tool_calls,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ToolCall represents a function/tool call by an agent
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // "function", "mcp_tool"
	Function map[string]interface{} `json:"function"`
	Result   interface{}            `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// AgentWorkflow represents a multi-step agent workflow
type AgentWorkflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Steps       []WorkflowStep         `json:"steps"`
	Status      string                 `json:"status"` // "pending", "running", "completed", "failed"
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// WorkflowStep represents a single step in an agent workflow
type WorkflowStep struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "llm_query", "tool_call", "condition", "loop"
	Config      map[string]interface{} `json:"config"`
	Status      string                 `json:"status"` // "pending", "running", "completed", "failed"
	Input       interface{}            `json:"input,omitempty"`
	Output      interface{}            `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// AgentConfig represents global agent configuration settings
type AgentConfig struct {
	// Global Agent Settings
	Enabled          bool   `json:"enabled"`
	DefaultFramework string `json:"default_framework"` // "eino", "custom"
	MaxConcurrent    int    `json:"max_concurrent"`
	Timeout          int    `json:"timeout"` // seconds

	// Framework-specific Settings
	EinoConfig   map[string]interface{} `json:"eino_config,omitempty"`
	CustomConfig map[string]interface{} `json:"custom_config,omitempty"`

	// Tool Integration
	EnableMCPTools     bool `json:"enable_mcp_tools"`
	EnableBuiltinTools bool `json:"enable_builtin_tools"`

	// Logging and Monitoring
	LogLevel      string `json:"log_level"`
	EnableTracing bool   `json:"enable_tracing"`

	UpdatedAt time.Time `json:"updated_at"`
}
