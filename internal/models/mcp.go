package models

// MCPServer represents a Model Context Protocol server configuration
type MCPServer struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	Enabled bool              `json:"enabled"`
}

// MCPTool represents a tool available through an MCP server
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Server      string                 `json:"server"`
	Schema      map[string]interface{} `json:"schema"`
}

// MCPToolCall represents a call to an MCP tool
type MCPToolCall struct {
	ID       string                 `json:"id"`
	ToolName string                 `json:"tool_name"`
	Server   string                 `json:"server"`
	Args     map[string]interface{} `json:"args"`
	Result   interface{}            `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// MCPResource represents a resource exposed by an MCP server
type MCPResource struct {
	URI         string            `json:"uri"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	MimeType    string            `json:"mime_type,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
