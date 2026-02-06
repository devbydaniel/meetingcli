package app

import (
	"github.com/devbydaniel/meetingcli/config"
	"github.com/devbydaniel/meetingcli/internal/audio"
	"github.com/devbydaniel/meetingcli/internal/domain/meeting/usecases"
)

type App struct {
	Record     *usecases.Record
	Transcribe *usecases.Transcribe
	Summarize  *usecases.Summarize
}

func New(cfg *config.Config) (*App, error) {
	capturer, err := audio.NewSystemAudioCapturer()
	if err != nil {
		return nil, err
	}

	return &App{
		Record: &usecases.Record{
			Capturer:       capturer,
			Recorder:       audio.NewRecorder(),
			MeetingsDir:    cfg.MeetingsDir,
			FolderTemplate: cfg.FolderTemplate,
		},
		Transcribe: &usecases.Transcribe{
			APIKey: cfg.MistralAPIKey,
		},
		Summarize: &usecases.Summarize{
			APIKey:       cfg.AnthropicKey,
			SystemPrompt: cfg.SummaryPrompt,
		},
	}, nil
}
