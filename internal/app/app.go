package app

import (
	"github.com/devbydaniel/meetingcli/config"
	"github.com/devbydaniel/meetingcli/internal/audio"
	"github.com/devbydaniel/meetingcli/internal/domain/meeting/usecases"
)

type App struct {
	StartRecording *usecases.StartRecording
	StopRecording  *usecases.StopRecording
	Transcribe     *usecases.Transcribe
	Summarize      *usecases.Summarize
}

func New(cfg *config.Config) (*App, error) {
	dm, err := audio.NewDeviceManager()
	if err != nil {
		return nil, err
	}

	recorder := audio.NewRecorder()

	startRecording := &usecases.StartRecording{
		DeviceManager:  dm,
		Recorder:       recorder,
		MeetingsDir:    cfg.MeetingsDir,
		StateDir:       cfg.StateDir,
		FolderTemplate: cfg.FolderTemplate,
	}

	stopRecording := &usecases.StopRecording{
		DeviceManager: dm,
		Recorder:      recorder,
		StateDir:      cfg.StateDir,
	}

	transcribe := &usecases.Transcribe{
		APIKey: cfg.MistralAPIKey,
	}

	summarize := &usecases.Summarize{
		APIKey:       cfg.AnthropicKey,
		SystemPrompt: cfg.SummaryPrompt,
	}

	return &App{
		StartRecording: startRecording,
		StopRecording:  stopRecording,
		Transcribe:     transcribe,
		Summarize:      summarize,
	}, nil
}
