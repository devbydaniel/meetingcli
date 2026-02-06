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
```

- **ffmpeg** — records mic audio and merges audio streams
- **Screen recording permission** — required for system audio capture (macOS will prompt on first use)

Run `meeting doctor` to verify everything is ready.

## Setup

Set your API keys:

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

```bash
meeting                          # record, Ctrl+C to stop → transcribe → summarize
meeting --name "standup"         # with a name
meeting list                     # list past meetings
meeting doctor                   # check prerequisites
```

`meeting start` also works as an alias for `meeting`.

## How it works

`meeting start` captures two audio streams in parallel:

1. **System audio** — via ScreenCaptureKit (macOS 12.3+), taps directly into the OS audio mixer. Works with any output device including Bluetooth headphones.
2. **Mic audio** — via ffmpeg from the default input device.

Press Ctrl+C to stop. The tool then:

1. Merges system + mic audio into `recording.wav`
2. Transcribes via Mistral Voxtral (with speaker diarization)
3. Summarizes via Claude Haiku 4.5

Each meeting produces:

```
~/meetings/2026-02-06_14-00-00/
├── recording.wav      # merged (used for transcription)
├── system.wav         # system audio
├── mic.wav            # mic audio
├── transcript.md
└── summary.md
```

## Configuration

`~/.config/meetingcli/config.toml`:

```toml
meetings_dir = "~/meetings"
mistral_api_key = ""
anthropic_api_key = ""
folder_template = "{{.Year}}-{{.Month}}-{{.Day}}_{{.Hour}}-{{.Minute}}-{{.Second}}{{if .Name}}_{{.Name}}{{end}}"
# summary_prompt = "Custom prompt here"
```

## Requirements

- macOS 12.3+
- [ffmpeg](https://ffmpeg.org/)
- [Mistral API key](https://console.mistral.ai/)
- [Anthropic API key](https://console.anthropic.com/)

## License

MIT
