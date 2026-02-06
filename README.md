# meetingcli

A CLI tool that records meetings (mic + system audio), transcribes them using [Mistral Voxtral](https://docs.mistral.ai/capabilities/audio_transcription), and generates summaries using Claude Haiku 4.5.

## Install

```bash
brew install devbydaniel/tap/meeting
```

Or build from source:

```bash
git clone https://github.com/devbydaniel/meetingcli.git
cd meetingcli
make build
```

## Prerequisites

```bash
brew install ffmpeg
brew install blackhole-2ch
```

- **ffmpeg** — records audio from system devices
- **BlackHole 2ch** — virtual audio driver that captures system audio (other participants' voices in calls)

Run `meeting doctor` to verify everything is ready.

## Setup

Set your API keys via environment variables or config file:

```bash
export MEETINGCLI_MISTRAL_API_KEY="your-key"
export MEETINGCLI_ANTHROPIC_API_KEY="your-key"
```

Or in `~/.config/meetingcli/config.toml`:

```toml
mistral_api_key = "your-key"
anthropic_api_key = "your-key"
```

## Usage

### Record a meeting (background)

```bash
meeting start                    # starts recording in background
meeting stop                     # stops, transcribes, and summarizes
```

### Record a meeting (foreground)

```bash
meeting start --sync             # records until Ctrl+C, then processes
```

### Name a meeting

```bash
meeting start --name "standup"   # folder: 2026-02-06_14-00-00_standup/
```

### List past meetings

```bash
meeting list
```

### Check prerequisites

```bash
meeting doctor
```

## How it works

When you start a recording, meetingcli:

1. Finds the BlackHole virtual audio device
2. Creates temporary audio devices via CoreAudio:
   - **Multi-Output** (your speakers/headphones + BlackHole) — so system audio flows to both your ears and BlackHole
   - **Aggregate** (BlackHole + your mic) — combines system audio and mic into one recordable input
3. Switches your system output to the Multi-Output device
4. Records from the Aggregate device via ffmpeg

When you stop:

1. Stops the recording and restores your original audio output
2. Destroys the temporary audio devices
3. Uploads audio to Mistral Voxtral for transcription with speaker diarization
4. Sends the transcript to Claude Haiku 4.5 for summarization

Each meeting produces a folder:

```
~/meetings/2026-02-06_14-00-00/
├── recording.wav
├── transcript.md
└── summary.md
```

## Configuration

Config file: `~/.config/meetingcli/config.toml`

```toml
# Where meeting folders are created (default: ~/meetings)
meetings_dir = "~/meetings"

# API keys (or use MEETINGCLI_MISTRAL_API_KEY / MEETINGCLI_ANTHROPIC_API_KEY env vars)
mistral_api_key = ""
anthropic_api_key = ""

# Folder name template
# Available: {{.Year}}, {{.Month}}, {{.Day}}, {{.Hour}}, {{.Minute}}, {{.Second}}, {{.Name}}
folder_template = "{{.Year}}-{{.Month}}-{{.Day}}_{{.Hour}}-{{.Minute}}-{{.Second}}{{if .Name}}_{{.Name}}{{end}}"

# Custom system prompt for summary generation (replaces default)
# summary_prompt = "Summarize this meeting as bullet points."
```

### Folder template examples

| Template | Result |
|---|---|
| `{{.Year}}-{{.Month}}-{{.Day}}_{{.Hour}}-{{.Minute}}-{{.Second}}` | `2026-02-06_14-00-00` |
| `{{.Year}}{{.Month}}{{.Day}}-{{.Name}}` | `20260206-standup` |
| `{{.Year}}/{{.Month}}/{{.Day}}_{{.Hour}}-{{.Minute}}` | `2026/02/06_14-00` (nested dirs) |

## Requirements

- macOS (uses CoreAudio for audio device management)
- [ffmpeg](https://ffmpeg.org/)
- [BlackHole 2ch](https://existential.audio/blackhole/)
- [Mistral API key](https://console.mistral.ai/)
- [Anthropic API key](https://console.anthropic.com/)

## License

MIT
