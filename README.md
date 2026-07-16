# lk-go-agent-sdk

A Go SDK for building realtime voice agents on [LiveKit](https://livekit.io), mirroring the architecture of [livekit/agents](https://github.com/livekit/agents) (Python) and [livekit/agents-js](https://github.com/livekit/agents-js).

> **Status:** early development. Worker protocol (registration, dispatch, job lifecycle) in progress.

## Architecture

| Package | Mirrors (Python) | Purpose |
|---|---|---|
| `agents` (root) | `worker.py`, `job.py` | Worker protocol client, job lifecycle |
| `cli/` | `cli/` | `dev` / `start` commands |
| `stt/`, `tts/`, `llm/`, `vad/` | same | Vendor-neutral streaming interfaces |
| `audio/` | `utils/audio` | Frames, buffering, resampling |
| `voice/` | `voice/` | `Agent`, `AgentSession`, pipeline orchestration |
| `plugins/*` | `livekit-plugins-*` | Vendor implementations (separate modules) |

Jobs run in goroutines with panic recovery instead of Python's subprocess pool — a deliberate divergence.

## Development

Run a local LiveKit server:

```sh
docker run --rm -p 7880:7880 livekit/livekit-server --dev
```

Dev credentials: API key `devkey`, secret `secret`, URL `ws://localhost:7880`.
