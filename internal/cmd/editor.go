package cmd

import (
	"context"
	"fmt"

	"github.com/kijimaD/ruins/assets"
	"github.com/kijimaD/ruins/internal/editor"
	"github.com/urfave/cli/v3"
)

// CmdEditor はゲームデータエディタを起動するコマンド
var CmdEditor = &cli.Command{
	Name:        "editor",
	Usage:       "editor",
	Description: "ゲームデータエディタをブラウザで開く",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "addr",
			Value: "localhost:8080",
			Usage: "サーバーのアドレス",
		},
		&cli.StringFlag{
			Name:  "file",
			Value: "assets/metadata/entities/raw/raw.toml",
			Usage: "raw.tomlのパス",
		},
		&cli.StringFlag{
			Name:  "output-dir",
			Value: "assets/file/textures/single",
			Usage: "スプライト保存先ディレクトリ",
		},
	},
	Action: runEditor,
}

func runEditor(_ context.Context, cmd *cli.Command) error {
	path := cmd.String("file")
	addr := cmd.String("addr")

	store, err := editor.NewStore(path)
	if err != nil {
		return fmt.Errorf("ストアの初期化に失敗: %w", err)
	}

	outputDir := cmd.String("output-dir")
	server := editor.NewServer(store, editor.WithAssetsFS(assets.FS), editor.WithOutputDir(outputDir))
	return server.ListenAndServe(addr)
}
