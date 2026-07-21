package voice

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/beingPro007/lk-go-agent-sdk/llm"
	"github.com/beingPro007/lk-go-agent-sdk/stt"
	"github.com/beingPro007/lk-go-agent-sdk/tts"
	"github.com/beingPro007/lk-go-agent-sdk/vad"
)

type AgentSession struct {
	stt stt.STT
	llm llm.LLM
	tts tts.TTS
	vad vad.VAD

	chatCtx *llm.ChatContext
	events  chan Event

	mu           sync.Mutex
	state        AgentState
	activeCancel context.CancelFunc
}

type SessionOption func(*AgentSession)

func WithSTT(s stt.STT) SessionOption { return func(a *AgentSession) { a.stt = s } }
func WithLLM(l llm.LLM) SessionOption { return func(a *AgentSession) { a.llm = l } }
func WithTTS(t tts.TTS) SessionOption { return func(a *AgentSession) { a.tts = t } }
func WithVAD(v vad.VAD) SessionOption { return func(a *AgentSession) { a.vad = v } }

func NewAgentSession(opts ...SessionOption) *AgentSession {
	s := &AgentSession{
		chatCtx: &llm.ChatContext{},
		events:  make(chan Event, 128),
		state:   StateListening,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *AgentSession) Events() <-chan Event { return s.events }

func (s *AgentSession) ChatContext() *llm.ChatContext { return s.chatCtx }

func (s *AgentSession) Start(ctx context.Context, agent *Agent, in AudioInput, out AudioOutput) error {
	if s.stt == nil || s.llm == nil || s.tts == nil {
		return errors.New("voice: STT, LLM and TTS are required")
	}
	if agent.Instructions != "" {
		s.chatCtx.Add(llm.RoleSystem, agent.Instructions)
	}

	sttStream := s.stt.Stream(ctx)
	defer sttStream.Close()

	var vadStream vad.Stream
	if s.vad != nil {
		vadStream = s.vad.Stream(ctx)
		defer vadStream.Close()
	}

	go func() {
		for f := range in.Frames() {
			sttStream.PushFrame(f)
			if vadStream != nil {
				vadStream.PushFrame(f)
			}
		}
		sttStream.EndInput()
		if vadStream != nil {
			vadStream.EndInput()
		}
	}()

	if vadStream != nil {
		go func() {
			for ev := range vadStream.Chan() {
				switch ev.Type {
				case vad.StartOfSpeech:
					s.emit(Event{Type: EventUserStartedSpeaking})
					s.interrupt(out)
				case vad.EndOfSpeech:
					s.emit(Event{Type: EventUserStoppedSpeaking})
				}
			}
		}()
	}

	var userText strings.Builder
	for ev := range sttStream.Chan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		text := firstAlt(ev)
		switch ev.Type {
		case stt.InterimTranscript:
			s.emit(Event{Type: EventUserTranscript, Transcript: text, IsFinal: false})
		case stt.FinalTranscript:
			userText.WriteString(text)
			userText.WriteString(" ")
			s.emit(Event{Type: EventUserTranscript, Transcript: text, IsFinal: true})
		case stt.EndOfSpeech:
			final := strings.TrimSpace(userText.String())
			userText.Reset()
			if final != "" {
				s.generate(ctx, agent, out, final)
			}
		}
	}
	return nil
}

func (s *AgentSession) generate(ctx context.Context, agent *Agent, out AudioOutput, userText string) {
	genCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.activeCancel = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.activeCancel = nil
		s.mu.Unlock()
		cancel()
	}()

	s.chatCtx.Add(llm.RoleUser, userText)
	s.setState(StateThinking)

	llmStream := s.llm.Chat(genCtx, s.chatCtx, agent.Tools)
	defer llmStream.Close()
	ttsStream := s.tts.Stream(genCtx)
	defer ttsStream.Close()

	audioDone := make(chan struct{})
	go func() {
		defer close(audioDone)
		started := false
		for a := range ttsStream.Chan() {
			if len(a.Frame.Data) == 0 {
				continue
			}
			if !started {
				started = true
				s.setState(StateSpeaking)
				s.emit(Event{Type: EventAgentStartedSpeaking})
			}
			if err := out.CaptureFrame(a.Frame); err != nil {
				return
			}
		}
	}()

	var resp strings.Builder
	for chunk := range llmStream.Chan() {
		if genCtx.Err() != nil {
			break
		}
		d := chunk.Delta.Content
		if d == "" {
			continue
		}
		resp.WriteString(d)
		s.emit(Event{Type: EventAgentResponse, Text: d})
		ttsStream.PushText(d)
	}
	ttsStream.EndInput()
	<-audioDone

	if genCtx.Err() == nil {
		out.WaitForPlayout()
	}
	if resp.Len() > 0 {
		s.chatCtx.Add(llm.RoleAssistant, resp.String())
	}
	s.emit(Event{Type: EventAgentStoppedSpeaking})
	s.setState(StateListening)
}

func (s *AgentSession) interrupt(out AudioOutput) {
	s.mu.Lock()
	cancel := s.activeCancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		out.ClearBuffer()
	}
}

func (s *AgentSession) setState(st AgentState) {
	s.mu.Lock()
	s.state = st
	s.mu.Unlock()
	s.emit(Event{Type: EventStateChanged, State: st})
}

func (s *AgentSession) emit(ev Event) {
	select {
	case s.events <- ev:
	default:
	}
}

func firstAlt(ev stt.SpeechEvent) string {
	if len(ev.Alternatives) == 0 {
		return ""
	}
	return ev.Alternatives[0].Text
}
