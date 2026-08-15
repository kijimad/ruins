package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kijimaD/ruins/internal/designdoc"
	"github.com/urfave/cli/v3"
)

// CmdGenReadme はREADME.mdを生成するサブコマンド
var CmdGenReadme = &cli.Command{
	Name:   "genreadme",
	Usage:  "generate README.md from README.tmpl.md",
	Action: runGenReadme,
}

const (
	templateFile          = "README.tmpl.md"
	outputFile            = "README.md"
	imageDir              = "internal/states/testdata"
	placeholder           = "<!-- VRT_IMAGES -->"
	designStatusPlacehldr = "<!-- DESIGN_STATUS -->"
	columns               = 4
)

func runGenReadme(_ context.Context, _ *cli.Command) error {
	tmpl, err := os.ReadFile(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	table, err := buildImageTable()
	if err != nil {
		return fmt.Errorf("failed to build image table: %w", err)
	}

	docs, err := designdoc.LoadDir(designdoc.DefaultDir)
	if err != nil {
		return fmt.Errorf("failed to read design documents: %w", err)
	}
	statusTable := designdoc.RenderStatusSection(docs)

	result := strings.Replace(string(tmpl), placeholder, table, 1)
	result = strings.Replace(result, designStatusPlacehldr, statusTable, 1)
	if err := os.WriteFile(outputFile, []byte(result), 0o644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	fmt.Printf("Generated %s from %s (%s)\n", outputFile, templateFile, imageDir)
	return nil
}

// buildImageTable はtestdata内のPNG画像から4列のMarkdownテーブルを生成する
func buildImageTable() (string, error) {
	return buildImageTableFrom(imageDir)
}

// render3DImageSubdir はローポリ3D表示の参照画像を置くサブディレクトリ。
const render3DImageSubdir = "TestRender3DImages"

// imageEntry はテーブルに載せる画像1枚。README からの相対パスと見出しを持つ。
type imageEntry struct {
	path  string
	label string
}

// buildImageTableFrom は指定ディレクトリのPNG画像から4列のMarkdownテーブルを生成する。
// 3Dの参照画像を先頭に置き、続けて直下の TestGolden_*.png を並べる。3D表示はローポリ化の要なので目立たせる。
func buildImageTableFrom(dir string) (string, error) {
	entries, err := collectSubdirImages(dir, render3DImageSubdir, "3D ")
	if err != nil {
		return "", err
	}
	top, err := collectTopImages(dir)
	if err != nil {
		return "", err
	}
	entries = append(entries, top...)

	if len(entries) == 0 {
		return "*no images*", nil
	}
	return renderImageTable(entries), nil
}

// pngNames はディレクトリ直下のPNGファイル名をソートして返す。
func pngNames(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range files {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

// collectSubdirImages はサブディレクトリ内のPNGを集める。見出しは labelPrefix + 拡張子なしのファイル名。
// サブディレクトリが無ければ空を返す。
func collectSubdirImages(dir, sub, labelPrefix string) ([]imageEntry, error) {
	subPath := filepath.Join(dir, sub)
	names, err := pngNames(subPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", subPath, err)
	}
	entries := make([]imageEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, imageEntry{
			path:  filepath.Join(subPath, name),
			label: labelPrefix + strings.TrimSuffix(name, ".png"),
		})
	}
	return entries, nil
}

// collectTopImages はディレクトリ直下の TestGolden_*.png を集める。見出しは TestGolden_ 接頭辞を外した名前。
func collectTopImages(dir string) ([]imageEntry, error) {
	names, err := pngNames(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", dir, err)
	}
	entries := make([]imageEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, imageEntry{
			path:  filepath.Join(dir, name),
			label: strings.TrimSuffix(strings.TrimPrefix(name, "TestGolden_"), ".png"),
		})
	}
	return entries, nil
}

// renderImageTable は画像エントリを4列のMarkdownテーブルにする。
func renderImageTable(entries []imageEntry) string {
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

	for i, e := range entries {
		if i%columns == 0 {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString("|")
		}
		fmt.Fprintf(&sb, " <img src=\"%s\" width=\"200\" /><br>%s |", e.path, e.label)
	}
	// 最終行の残りセルを埋める
	if rem := len(entries) % columns; rem != 0 {
		for range columns - rem {
			sb.WriteString(" |")
		}
	}
	sb.WriteString("\n")

	return sb.String()
}
