package cmd

import (
	"context"
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/urfave/cli/v3"
)

const splash = `───────────────────────────────────────────────────────
██████  ██    ██ ██ ███    ██ ███████
██   ██ ██    ██ ██ ████   ██ ██
██████  ██    ██ ██ ██ ██  ██ ███████
██   ██ ██    ██ ██ ██  ██ ██      ██
██   ██  ██████  ██ ██   ████ ███████
───────────────────────────────────────────────────────
`

// NewMainApp は新しいメインアプリケーションを作成する
func NewMainApp() *cli.Command {
	app := &cli.Command{
		Name:        "ruins",
		Usage:       "ruins [subcommand] [args]",
		Description: splash + "\nThis is Roguelike!",
		Version:     consts.AppVersion,
		Commands: []*cli.Command{
			CmdPlay,
			CmdScreenshot,
			CmdGenerateItemDoc,
			CmdGenerateEnemyDoc,
		},
	}

	return app
}

// RunMainApp はメインアプリケーションを実行する
func RunMainApp(app *cli.Command, args ...string) error {
	err := app.Run(context.Background(), args)
	if err != nil {
		return fmt.Errorf("コマンド実行が失敗した: %w", err)
	}

	return nil
}
