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
		&cli.StringFlag{
			Name:  "palette-dir",
			Value: "assets/levels/palettes",
			Usage: "パレットファイルのディレクトリ",
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

	paletteStore, err := editor.NewPaletteStore(cmd.String("palette-dir"))
	if err != nil {
		return fmt.Errorf("パレットストアの初期化に失敗: %w", err)
	}

	layoutDirs := []string{
		"assets/levels/layouts",
		"assets/levels/chunks",
		"assets/levels/facilities",
	}
	layoutStore, err := editor.NewLayoutStore(layoutDirs)
	if err != nil {
		return fmt.Errorf("レイアウトストアの初期化に失敗: %w", err)
	}

	outputDir := cmd.String("output-dir")
	server := editor.NewServer(store,
		editor.WithAssetsFS(assets.FS),
		editor.WithOutputDir(outputDir),
		editor.WithPaletteStore(paletteStore),
		editor.WithLayoutStore(layoutStore),
	)
	return server.ListenAndServe(addr)
}
