package voice

import "github.com/beingPro007/lk-go-agent-sdk/llm"

type Agent struct {
	Instructions string
	Tools        []llm.Tool
}
