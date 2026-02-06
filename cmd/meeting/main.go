package main

import (
	"fmt"
	"os"

	"github.com/devbydaniel/meetingcli/config"
	"github.com/devbydaniel/meetingcli/internal/app"
	"github.com/devbydaniel/meetingcli/internal/cli"
	"github.com/devbydaniel/meetingcli/internal/output"
)

func main() {
	if err := run(); err != nil {
		formatter := output.NewFormatter(os.Stderr)
		formatter.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		return fmt.Errorf("initializing app: %w", err)
	}

	deps := &cli.Dependencies{
		App:    application,
		Config: cfg,
	}

	return cli.NewRootCmd(deps).Execute()
}
