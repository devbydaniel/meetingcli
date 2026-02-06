# CLAUDE.md

Meeting recorder CLI with transcription and AI summary.

## Architecture

Layered: `cmd/` → `internal/cli/` → `internal/app/` → `internal/domain/*/usecases/`

- **Use case-based**: Each operation is its own struct with focused dependencies
- **App wiring**: `internal/app/app.go` creates all use cases and resolves dependencies

## Adding a CLI Command

1. Create `internal/cli/<command>.go`
2. Use `deps.App.<UseCase>.Execute()` to call business logic
3. Register with `rootCmd.AddCommand()` in `NewRootCmd()`

## Audio

- macOS 12.3+ (ScreenCaptureKit for system audio via cgo, ffmpeg for mic)
- Two parallel recordings: system audio (in-process cgo) + mic (ffmpeg)
- Merged into single `recording.wav` on stop
- Foreground only — Ctrl+C to stop
- Objective-C code: `internal/audio/capture_darwin.m`

## Config

- Config file: `~/.config/meetingcli/config.toml`
- `meetings_dir` — where meeting folders are created (default: `~/meetings`)
- API keys via config or env vars (`MEETINGCLI_MISTRAL_API_KEY`, `MEETINGCLI_ANTHROPIC_API_KEY`)

## Output

- Use `internal/output/Formatter` for all user-facing output
- Don't print directly in commands

## Development

- Use `make dev` to run with dev settings
- `make build` produces the `meeting` binary
