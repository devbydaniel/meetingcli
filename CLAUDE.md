# CLAUDE.md

Meeting recorder CLI with transcription and AI summary.

## Architecture

Layered: `cmd/` → `internal/cli/` → `internal/app/` → `internal/domain/*/usecases/`

- **Use case-based**: Each operation is its own struct with focused dependencies
- **Consumer-defined interfaces**: Cross-domain deps use interfaces defined by the consumer
- **App wiring**: `internal/app/app.go` creates all use cases and resolves dependencies

## Adding a CLI Command

1. Create `internal/cli/<command>.go`
2. Use `deps.App.<UseCase>.Execute()` to call business logic
3. Register with `rootCmd.AddCommand()` in `NewRootCmd()`

## Audio

- macOS only (CoreAudio via Swift helper)
- Requires BlackHole 2ch (`brew install blackhole-2ch`) and ffmpeg
- Audio devices created/destroyed programmatically per recording session
- Swift helper at `internal/audio/devices_helper.swift`

## Config

- Config file: `~/.config/meetingcli/config.toml`
- `meetings_dir` — where meeting folders are created (default: `~/meetings`)
- API keys via config or env vars (`MISTRAL_API_KEY`, `ANTHROPIC_API_KEY`)

## Output

- Use `internal/output/Formatter` for all user-facing output
- Don't print directly in commands

## Development

- Use `make dev` to run with dev settings
- `make build` produces the `meeting` binary
