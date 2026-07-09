package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"
)

// CmdGenReadme はREADME.mdを生成するサブコマンド
var CmdGenReadme = &cli.Command{
	Name:   "genreadme",
	Usage:  "README.tmpl.mdからREADME.mdを生成する",
	Action: runGenReadme,
}

const (
	templateFile = "README.tmpl.md"
	outputFile   = "README.md"
	imageDir     = "internal/states/testdata"
	placeholder  = "<!-- VRT_IMAGES -->"
	columns      = 4
)

func runGenReadme(_ context.Context, _ *cli.Command) error {
	tmpl, err := os.ReadFile(templateFile)
	if err != nil {
		return fmt.Errorf("テンプレートの読み込みに失敗: %w", err)
	}

	table, err := buildImageTable()
	if err != nil {
		return fmt.Errorf("画像テーブルの生成に失敗: %w", err)
	}

	result := strings.Replace(string(tmpl), placeholder, table, 1)
	if err := os.WriteFile(outputFile, []byte(result), 0644); err != nil {
		return fmt.Errorf("README.mdの書き込みに失敗: %w", err)
	}

	fmt.Printf("Generated %s from %s (%s)\n", outputFile, templateFile, imageDir)
	return nil
}

// buildImageTable はtestdata内のPNG画像から4列のMarkdownテーブルを生成する
func buildImageTable() (string, error) {
	return buildImageTableFrom(imageDir)
}

// buildImageTableFrom は指定ディレクトリ内のPNG画像から4列のMarkdownテーブルを生成する
func buildImageTableFrom(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("%sの読み込みに失敗: %w", dir, err)
	}

	var images []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			images = append(images, e.Name())
		}
	}
	sort.Strings(images)

	if len(images) == 0 {
		return "*画像なし*", nil
	}

	var sb strings.Builder

	// Markdownテーブルのヘッダー
	sb.WriteString("|")
	for range columns {
		sb.WriteString(" |")
	}
	sb.WriteString("\n|")
	for range columns {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")

	for i, name := range images {
		if i%columns == 0 {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString("|")
		}
		label := strings.TrimSuffix(strings.TrimPrefix(name, "TestGolden_"), ".png")
		imgPath := filepath.Join(dir, name)
		fmt.Fprintf(&sb, " <img src=\"%s\" width=\"200\" /><br>%s |", imgPath, label)
	}
	// 最終行の残りセルを埋める
	if rem := len(images) % columns; rem != 0 {
		for range columns - rem {
			sb.WriteString(" |")
		}
	}
	sb.WriteString("\n")

	return sb.String(), nil
}
