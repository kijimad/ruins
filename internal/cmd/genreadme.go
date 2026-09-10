package cmd

import (
	"context"
	"fmt"
	"io"
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

func runGenReadme(_ context.Context, cmd *cli.Command) error {
	return genReadme(cmd.Writer, templateFile, outputFile, imageDir, designdoc.DefaultDir)
}

// genReadme はテンプレートの各プレースホルダを画像テーブルと設計ドキュメント状態で置換し、
// outputPath へ書き出す。読み書きするパスはすべて引数で受け、生成の完了を out へ書く。
func genReadme(out io.Writer, templatePath, outputPath, imageDirPath, designDir string) error {
	tmpl, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	table, err := buildImageTableFrom(imageDirPath)
	if err != nil {
		return fmt.Errorf("failed to build image table: %w", err)
	}

	docs, err := designdoc.LoadDir(designDir)
	if err != nil {
		return fmt.Errorf("failed to read design documents: %w", err)
	}
	statusTable := designdoc.RenderStatusSection(docs)

	result := strings.Replace(string(tmpl), placeholder, table, 1)
	result = strings.Replace(result, designStatusPlacehldr, statusTable, 1)
	if err := os.WriteFile(outputPath, []byte(result), 0o644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	_, _ = fmt.Fprintf(out, "Generated %s from %s (%s)\n", outputPath, templatePath, imageDirPath)
	return nil
}

// imageEntry はテーブルに載せる画像1枚。README からの相対パスと見出しを持つ。
type imageEntry struct {
	path  string
	label string
}

// buildImageTableFrom は指定ディレクトリ直下の TestGolden_*.png から4列のMarkdownテーブルを生成する。
func buildImageTableFrom(dir string) (string, error) {
	entries, err := collectTopImages(dir)
	if err != nil {
		return "", err
	}
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
