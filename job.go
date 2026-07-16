package agents

import (
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

type JobRequest struct {
	job *livekit.Job
}

func (r *JobRequest) Job() *livekit.Job {
	return r.job
}

func (r *JobRequest) Room() *livekit.Room {
	return r.job.GetRoom()
}

type JobContext struct {
	job  *livekit.Job
	room *lksdk.Room
}

func (c *JobContext) Job() *livekit.Job {
	return c.job
}

func (c *JobContext) Room() *lksdk.Room {
	return c.room
}
