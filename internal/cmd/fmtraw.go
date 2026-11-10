package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli/v3"
)

// rawTOMLPath は fmtraw が既定で整形する raw.toml のパス
const rawTOMLPath = "assets/metadata/entities/raw/raw.toml"

// CmdFmtRaw は raw.toml を正規形に整形するサブコマンド
var CmdFmtRaw = &cli.Command{
	Name:      "fmtraw",
	Usage:     "format raw.toml into its canonical form",
	ArgsUsage: "[path]",
	Action:    runFmtRaw,
}

func runFmtRaw(_ context.Context, cmd *cli.Command) error {
	path := rawTOMLPath
	if cmd.Args().Present() {
		path = cmd.Args().First()
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}
	out, err := formatRaw(src)
	if err != nil {
		return fmt.Errorf("failed to format %s: %w", path, err)
	}
	if bytes.Equal(src, out) {
		return nil
	}
	return os.WriteFile(path, out, 0o644)
}

// formatRaw は TOML を正規形に整える。キーを並べ替え、子の表をインデントし、同じ表の飛び飛びの配置を併合する。
// どの道具で書いた TOML も同じ形に戻る。コメントは残らない。
func formatRaw(src []byte) ([]byte, error) {
	var m map[string]any
	if _, err := toml.Decode(string(src), &m); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	enc := toml.NewEncoder(&b)
	enc.Indent = "  "
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
