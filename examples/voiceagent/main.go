package main

import (
	"bufio"
	"log/slog"
	"os"
	"strings"

	agents "github.com/beingPro007/lk-go-agent-sdk"
	"github.com/beingPro007/lk-go-agent-sdk/cli"
	"github.com/beingPro007/lk-go-agent-sdk/plugins/cartesia"
	"github.com/beingPro007/lk-go-agent-sdk/plugins/deepgram"
	"github.com/beingPro007/lk-go-agent-sdk/plugins/gemini"
	"github.com/beingPro007/lk-go-agent-sdk/plugins/silero"
	"github.com/beingPro007/lk-go-agent-sdk/voice"
)

func main() {
	loadDotenv(".env", "../../.env")
	cli.Run(agents.WorkerOptions{
		Entrypoint: entrypoint,
	})
}

func loadDotenv(paths ...string) {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k, v = strings.TrimSpace(k), strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if _, set := os.LookupEnv(k); !set {
				os.Setenv(k, v)
			}
		}
		f.Close()
		return
	}
}

func entrypoint(job *agents.JobContext) error {
	stt, err := deepgram.New()
	if err != nil {
		return err
	}
	llm, err := gemini.New(job.Context())
	if err != nil {
		return err
	}
	tts, err := cartesia.New()
	if err != nil {
		return err
	}
	vad, err := silero.New()
	if err != nil {
		return err
	}
	defer vad.Close()

	session := voice.NewAgentSession(
		voice.WithSTT(stt),
		voice.WithLLM(llm),
		voice.WithTTS(tts),
		voice.WithVAD(vad),
	)

	go func() {
		for ev := range session.Events() {
			switch ev.Type {
			case voice.EventUserTranscript:
				slog.Info("user", "text", ev.Transcript, "final", ev.IsFinal)
			case voice.EventAgentResponse:
				slog.Info("agent", "delta", ev.Text)
			case voice.EventUserStartedSpeaking:
				slog.Info("user started speaking (barge-in)")
			}
		}
	}()

	agent := &voice.Agent{
		Instructions: "You are a helpful voice assistant. Keep your responses short, natural, and conversational.",
	}
	return voice.RunRoomSession(job, session, agent)
}
