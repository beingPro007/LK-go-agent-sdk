package llm

import "context"

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

type ChatContext struct {
	Messages []Message
}

func (c *ChatContext) Add(role Role, content string) {
	c.Messages = append(c.Messages, Message{Role: role, Content: content})
}

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type ChoiceDelta struct {
	Role      Role
	Content   string
	ToolCalls []ToolCall
}

type ChatChunk struct {
	ID    string
	Delta ChoiceDelta
}

type LLM interface {
	Chat(ctx context.Context, chatCtx *ChatContext, tools []Tool) LLMStream
}

type LLMStream interface {
	Chan() <-chan ChatChunk
	Err() error
	Close() error
}
