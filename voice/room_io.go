package voice

import (
	"sync"

	agents "github.com/beingPro007/lk-go-agent-sdk"
	"github.com/beingPro007/lk-go-agent-sdk/audio/track"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
)

type RoomOption func(*roomConfig)

type roomConfig struct {
	inputSampleRate  int
	inputChannels    int
	outputSampleRate int
	outputChannels   int
	trackName        string
}

func WithInputSampleRate(r int) RoomOption  { return func(c *roomConfig) { c.inputSampleRate = r } }
func WithOutputSampleRate(r int) RoomOption { return func(c *roomConfig) { c.outputSampleRate = r } }
func WithTrackName(n string) RoomOption     { return func(c *roomConfig) { c.trackName = n } }

func RunRoomSession(job *agents.JobContext, s *AgentSession, agent *Agent, opts ...RoomOption) error {
	cfg := &roomConfig{
		inputSampleRate: 16000,
		inputChannels:   1,
		trackName:       "agent",
	}
	for _, o := range opts {
		o(cfg)
	}

	outRate := cfg.outputSampleRate
	outCh := cfg.outputChannels
	if outRate == 0 && s.tts != nil {
		outRate = s.tts.SampleRate()
		outCh = s.tts.NumChannels()
	}
	if outRate == 0 {
		outRate = 24000
	}
	if outCh == 0 {
		outCh = 1
	}

	inputCh := make(chan *track.Input, 1)
	var once sync.Once
	cb := lksdk.NewRoomCallback()
	cb.OnTrackSubscribed = func(remote *webrtc.TrackRemote, pub *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		if remote.Kind() != webrtc.RTPCodecTypeAudio {
			return
		}
		in, err := track.NewRemoteTrackInput(remote, cfg.inputSampleRate, cfg.inputChannels)
		if err != nil {
			return
		}
		once.Do(func() { inputCh <- in })
	}

	room, err := job.Connect(cb)
	if err != nil {
		return err
	}

	out, err := track.NewOutput(outRate, outCh)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := room.LocalParticipant.PublishTrack(out.Track(), &lksdk.TrackPublicationOptions{Name: cfg.trackName}); err != nil {
		return err
	}

	select {
	case in := <-inputCh:
		defer in.Close()
		return s.Start(job.Context(), agent, in, out)
	case <-job.Done():
		return nil
	}
}
