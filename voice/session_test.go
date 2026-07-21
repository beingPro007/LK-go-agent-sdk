package voice

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/beingPro007/lk-go-agent-sdk/audio"
	"github.com/beingPro007/lk-go-agent-sdk/llm"
	"github.com/beingPro007/lk-go-agent-sdk/stt"
	"github.com/beingPro007/lk-go-agent-sdk/tts"
)

// --- fake STT: emits a scripted transcript + end-of-turn, ignores audio ---

type fakeSTT struct{ script []stt.SpeechEvent }

func (f *fakeSTT) Stream(ctx context.Context) stt.SpeechStream {
	s := &fakeSTTStream{events: make(chan stt.SpeechEvent, 8)}
	for _, ev := range f.script {
		s.events <- ev
	}
	return s
}

type fakeSTTStream struct {
	events chan stt.SpeechEvent
	once   sync.Once
}

func (s *fakeSTTStream) PushFrame(audio.Frame)        {}
func (s *fakeSTTStream) Flush()                       {}
func (s *fakeSTTStream) EndInput()                    { s.once.Do(func() { close(s.events) }) }
func (s *fakeSTTStream) Chan() <-chan stt.SpeechEvent { return s.events }
func (s *fakeSTTStream) Err() error                   { return nil }
func (s *fakeSTTStream) Close() error                 { s.once.Do(func() { close(s.events) }); return nil }

// --- fake LLM: streams a fixed reply as one chunk ---

type fakeLLM struct{ reply string }

func (f *fakeLLM) Chat(ctx context.Context, _ *llm.ChatContext, _ []llm.Tool) llm.LLMStream {
	ch := make(chan llm.ChatChunk, 1)
	ch <- llm.ChatChunk{Delta: llm.ChoiceDelta{Role: llm.RoleAssistant, Content: f.reply}}
	close(ch)
	return &fakeLLMStream{chunks: ch}
}

type fakeLLMStream struct{ chunks chan llm.ChatChunk }

func (s *fakeLLMStream) Chan() <-chan llm.ChatChunk { return s.chunks }
func (s *fakeLLMStream) Err() error                 { return nil }
func (s *fakeLLMStream) Close() error               { return nil }

// --- fake TTS: turns each pushed text token into one audio frame ---

type fakeTTS struct{}

func (f *fakeTTS) SampleRate() int  { return 24000 }
func (f *fakeTTS) NumChannels() int { return 1 }
func (f *fakeTTS) Stream(ctx context.Context) tts.SynthesizeStream {
	return &fakeTTSStream{audio: make(chan tts.SynthesizedAudio, 8)}
}

type fakeTTSStream struct {
	audio chan tts.SynthesizedAudio
	once  sync.Once
}

func (s *fakeTTSStream) PushText(tok string) {
	s.audio <- tts.SynthesizedAudio{Frame: audio.NewFrame([]int16{1, 2, 3, 4}, 24000, 1)}
}
func (s *fakeTTSStream) Flush()                            {}
func (s *fakeTTSStream) EndInput()                         { s.once.Do(func() { close(s.audio) }) }
func (s *fakeTTSStream) Chan() <-chan tts.SynthesizedAudio { return s.audio }
func (s *fakeTTSStream) Err() error                        { return nil }
func (s *fakeTTSStream) Close() error                      { s.once.Do(func() { close(s.audio) }); return nil }

// --- fake audio input/output ---

type fakeInput struct{ frames chan audio.Frame }

func (f *fakeInput) Frames() <-chan audio.Frame { return f.frames }

type fakeOutput struct {
	mu       sync.Mutex
	captured int
	cleared  bool
}

func (o *fakeOutput) CaptureFrame(audio.Frame) error {
	o.mu.Lock()
	o.captured++
	o.mu.Unlock()
	return nil
}
func (o *fakeOutput) ClearBuffer()    { o.mu.Lock(); o.cleared = true; o.mu.Unlock() }
func (o *fakeOutput) WaitForPlayout() {}

// TestFullTurn drives one complete conversational turn through the real
// AgentSession orchestrator with fake plugins: a user transcript + end-of-turn
// should trigger an LLM reply, get synthesized to audio, and land on the output.
func TestFullTurn(t *testing.T) {
	in := &fakeInput{frames: make(chan audio.Frame, 4)}
	in.frames <- audio.NewFrame(make([]int16, 320), 16000, 1)
	close(in.frames)

	out := &fakeOutput{}

	session := NewAgentSession(
		WithSTT(&fakeSTT{script: []stt.SpeechEvent{
			{Type: stt.FinalTranscript, Alternatives: []stt.SpeechData{{Text: "hello world"}}},
			{Type: stt.EndOfSpeech},
		}}),
		WithLLM(&fakeLLM{reply: "hi there"}),
		WithTTS(&fakeTTS{}),
	)

	var events []Event
	var emu sync.Mutex
	go func() {
		for ev := range session.Events() {
			emu.Lock()
			events = append(events, ev)
			emu.Unlock()
		}
	}()

	done := make(chan error, 1)
	go func() {
		done <- session.Start(context.Background(), &Agent{Instructions: "be nice"}, in, out)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("session did not complete in time")
	}

	// The agent should have produced at least one synthesized audio frame.
	out.mu.Lock()
	captured := out.captured
	out.mu.Unlock()
	if captured == 0 {
		t.Fatal("no audio frames reached the output")
	}

	// The chat context should record the system prompt, the user turn, and the reply.
	msgs := session.ChatContext().Messages
	if len(msgs) != 3 {
		t.Fatalf("chat context has %d messages, want 3: %+v", len(msgs), msgs)
	}
	if msgs[1].Role != llm.RoleUser || msgs[1].Content != "hello world" {
		t.Fatalf("user message wrong: %+v", msgs[1])
	}
	if msgs[2].Role != llm.RoleAssistant || msgs[2].Content != "hi there" {
		t.Fatalf("assistant message wrong: %+v", msgs[2])
	}
}

// TestStartRequiresPlugins verifies Start refuses to run without the core plugins.
func TestStartRequiresPlugins(t *testing.T) {
	session := NewAgentSession(WithSTT(&fakeSTT{}))
	err := session.Start(context.Background(), &Agent{}, &fakeInput{frames: make(chan audio.Frame)}, &fakeOutput{})
	if err == nil {
		t.Fatal("expected error when LLM/TTS are missing")
	}
}
