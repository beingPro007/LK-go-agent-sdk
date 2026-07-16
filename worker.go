package agents

import (
	"context"
	"errors"
	"time"

	"github.com/livekit/protocol/livekit"
)

type JobType = livekit.JobType

const (
	JobTypeRoom        JobType = livekit.JobType_JT_ROOM
	JobTypePublisher   JobType = livekit.JobType_JT_PUBLISHER
	JobTypeParticipant JobType = livekit.JobType_JT_PARTICIPANT
)

type WorkerStatus = livekit.WorkerStatus

const (
	WorkerStatusAvailable WorkerStatus = livekit.WorkerStatus_WS_AVAILABLE
	WorkerStatusFull      WorkerStatus = livekit.WorkerStatus_WS_FULL
)

type EntrypointFunc func(ctx *JobContext) error

type RequestFunc func(req *JobRequest) error

type LoadFunc func(w *Worker) float32

type WorkerOptions struct {
	Entrypoint EntrypointFunc
	RequestFnc RequestFunc
	LoadFnc    LoadFunc

	AgentName     string
	JobType       JobType
	LoadThreshold float32

	URL       string
	APIKey    string
	APISecret string

	PingInterval time.Duration
}

type Worker struct {
	opts WorkerOptions

	id     string
	status WorkerStatus
}

func NewWorker(opts WorkerOptions) (*Worker, error) {
	if opts.Entrypoint == nil {
		return nil, errors.New("agents: WorkerOptions.Entrypoint is required")
	}
	if opts.PingInterval == 0 {
		opts.PingInterval = 10 * time.Second
	}
	if opts.LoadThreshold == 0 {
		opts.LoadThreshold = 0.75
	}
	return &Worker{
		opts:   opts,
		status: WorkerStatusAvailable,
	}, nil
}

func (w *Worker) ID() string {
	return w.id
}

func (w *Worker) Run(ctx context.Context) error {
	return errors.New("agents: not implemented")
}

func (w *Worker) Drain(ctx context.Context) error {
	return errors.New("agents: not implemented")
}
