package gemini

import (
	"testing"

	"github.com/beingPro007/lk-go-agent-sdk/llm"
	"google.golang.org/genai"
)

// A text part in a Gemini response becomes a ChatChunk with that text as the delta.
func TestRespToChunkText(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			Content: &genai.Content{Parts: []*genai.Part{{Text: "hello"}}},
		}},
	}
	chunk, ok := respToChunk(resp)
	if !ok {
		t.Fatal("expected chunk")
	}
	if chunk.Delta.Content != "hello" {
		t.Fatalf("content = %q, want hello", chunk.Delta.Content)
	}
}

// A Gemini functionCall part becomes a ToolCall with the args JSON-encoded into Arguments.
func TestRespToChunkFunctionCall(t *testing.T) {
	resp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			Content: &genai.Content{Parts: []*genai.Part{{
				FunctionCall: &genai.FunctionCall{Name: "get_weather", Args: map[string]any{"city": "paris"}},
			}}},
		}},
	}
	chunk, ok := respToChunk(resp)
	if !ok {
		t.Fatal("expected chunk")
	}
	if len(chunk.Delta.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(chunk.Delta.ToolCalls))
	}
	tc := chunk.Delta.ToolCalls[0]
	if tc.Name != "get_weather" || tc.Arguments != `{"city":"paris"}` {
		t.Fatalf("tool call wrong: %+v", tc)
	}
}

// An empty/candidate-less response yields no chunk (ok=false), so we skip it in the stream.
func TestRespToChunkEmpty(t *testing.T) {
	if _, ok := respToChunk(&genai.GenerateContentResponse{}); ok {
		t.Fatal("expected no chunk for empty response")
	}
}

// Our roles translate to Gemini's model: system -> SystemInstruction (not a message),
// user -> "user", assistant -> "model".
func TestToContentsRolesAndSystem(t *testing.T) {
	cc := &llm.ChatContext{}
	cc.Add(llm.RoleSystem, "be brief")
	cc.Add(llm.RoleUser, "hi")
	cc.Add(llm.RoleAssistant, "hello")
	contents, sys := toContents(cc)
	if sys == nil || sys.Parts[0].Text != "be brief" {
		t.Fatalf("system instruction wrong: %+v", sys)
	}
	if len(contents) != 2 {
		t.Fatalf("contents = %d, want 2", len(contents))
	}
	if contents[0].Role != "user" || contents[1].Role != "model" {
		t.Fatalf("roles wrong: %q %q", contents[0].Role, contents[1].Role)
	}
}

// A JSON-schema tool parameters map converts recursively into a *genai.Schema
// (object type, typed properties, required list preserved).
func TestToSchema(t *testing.T) {
	m := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"city": map[string]any{"type": "string", "description": "city name"},
		},
		"required": []any{"city"},
	}
	s := toSchema(m)
	if s.Type != genai.TypeObject {
		t.Fatalf("type = %v, want OBJECT", s.Type)
	}
	if s.Properties["city"].Type != genai.TypeString {
		t.Fatalf("city type wrong: %v", s.Properties["city"].Type)
	}
	if len(s.Required) != 1 || s.Required[0] != "city" {
		t.Fatalf("required wrong: %v", s.Required)
	}
}
