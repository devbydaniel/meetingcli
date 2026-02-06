package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// DefaultSummaryPrompt is used when no custom prompt is configured.
const DefaultSummaryPrompt = `You are a meeting summarizer. Given a meeting transcript, produce a clear and concise summary in markdown format with these sections:

## Summary
A brief 2-3 sentence overview of what the meeting was about.

## Key Decisions
Bullet points of any decisions that were made.

## Action Items
Bullet points of tasks or follow-ups assigned, with the responsible person if identifiable.

## Discussion Highlights
Brief notes on the main topics discussed.

If any section has no content, omit it. Be concise but don't miss important details.`

// DefaultFolderTemplate is the default meeting folder name template.
// Available placeholders: {{.Year}}, {{.Month}}, {{.Day}}, {{.Hour}}, {{.Minute}}, {{.Second}}, {{.Name}}
const DefaultFolderTemplate = "{{.Year}}-{{.Month}}-{{.Day}}_{{.Hour}}-{{.Minute}}-{{.Second}}{{if .Name}}_{{.Name}}{{end}}"

type Config struct {
	MeetingsDir    string
	MistralAPIKey  string
	AnthropicKey   string
	SummaryPrompt  string // system prompt for summary generation
	FolderTemplate string // Go template for meeting folder names
}

type fileConfig struct {
	MeetingsDir    string `toml:"meetings_dir"`
	MistralAPIKey  string `toml:"mistral_api_key"`
	AnthropicKey   string `toml:"anthropic_api_key"`
	SummaryPrompt  string `toml:"summary_prompt"`
	FolderTemplate string `toml:"folder_template"`
}

func Load() (*Config, error) {
	cfg := &Config{
		MeetingsDir:    defaultMeetingsDir(),
		SummaryPrompt:  DefaultSummaryPrompt,
		FolderTemplate: DefaultFolderTemplate,
	}

	if configPath := configFilePath(); configPath != "" {
		var fc fileConfig
		if _, err := toml.DecodeFile(configPath, &fc); err == nil {
			if fc.MeetingsDir != "" {
				cfg.MeetingsDir = expandTilde(fc.MeetingsDir)
			}
			cfg.MistralAPIKey = fc.MistralAPIKey
			cfg.AnthropicKey = fc.AnthropicKey
			if fc.SummaryPrompt != "" {
				cfg.SummaryPrompt = fc.SummaryPrompt
			}
			if fc.FolderTemplate != "" {
				cfg.FolderTemplate = fc.FolderTemplate
			}
		}
	}

	applyEnvOverrides(cfg)

	// Ensure directories exist
	if err := os.MkdirAll(cfg.MeetingsDir, 0o755); err != nil {
		return nil, err
	}

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("MEETINGCLI_MISTRAL_API_KEY"); v != "" {
		cfg.MistralAPIKey = v
	}
	if v := os.Getenv("MEETINGCLI_ANTHROPIC_API_KEY"); v != "" {
		cfg.AnthropicKey = v
	}
	if v := os.Getenv("MEETINGCLI_MEETINGS_DIR"); v != "" {
		cfg.MeetingsDir = expandTilde(v)
	}
}

func configFilePath() string {
	var configDir string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		configDir = filepath.Join(xdg, "meetingcli")
	} else if home, err := os.UserHomeDir(); err == nil {
		configDir = filepath.Join(home, ".config", "meetingcli")
	} else {
		return ""
	}

	path := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func defaultMeetingsDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "meetings")
	}
	return filepath.Join(".", "meetings")
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
