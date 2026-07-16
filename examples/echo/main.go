package main

import (
	"log/slog"
	"time"

	agents "github.com/beingPro007/lk-go-agent-sdk"
	"github.com/beingPro007/lk-go-agent-sdk/cli"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

func main() {
	cli.Run(agents.WorkerOptions{
		Entrypoint: echo,
	})
}

func echo(job *agents.JobContext) error {
	echoTrack, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeOpus,
		ClockRate: 48000,
		Channels:  2,
	})
	if err != nil {
		return err
	}

	cb := lksdk.NewRoomCallback()
	cb.OnTrackSubscribed = func(track *webrtc.TrackRemote, pub *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		if track.Kind() != webrtc.RTPCodecTypeAudio {
			return
		}
		slog.Info("echoing audio track", "from", rp.Identity(), "track", pub.SID())
		go func() {
			for {
				pkt, _, err := track.ReadRTP()
				if err != nil {
					return
				}
				if len(pkt.Payload) == 0 {
					continue
				}
				echoTrack.WriteSample(media.Sample{
					Data:     pkt.Payload,
					Duration: 20 * time.Millisecond,
				}, nil)
			}
		}()
	}

	room, err := job.Connect(cb)
	if err != nil {
		return err
	}
	if _, err := room.LocalParticipant.PublishTrack(echoTrack, &lksdk.TrackPublicationOptions{
		Name: "echo",
	}); err != nil {
		return err
	}
	slog.Info("echo agent ready", "room", room.Name())

	<-job.Done()
	return nil
}
