package voice

type AgentState int

const (
	StateListening AgentState = iota
	StateThinking
	StateSpeaking
)

type EventType int

const (
	EventUserStartedSpeaking EventType = iota
	EventUserStoppedSpeaking
	EventUserTranscript
	EventAgentResponse
	EventAgentStartedSpeaking
	EventAgentStoppedSpeaking
	EventStateChanged
)

type Event struct {
	Type       EventType
	Transcript string
	IsFinal    bool
	Text       string
	State      AgentState
}
