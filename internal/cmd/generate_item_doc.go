package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/urfave/cli/v3"
)

// CmdGenerateItemDoc はアイテム出現確率のドキュメントを生成するコマンド
var CmdGenerateItemDoc = &cli.Command{
	Name:        "generate-item-doc",
	Usage:       "generate-item-doc",
	Description: "Generate item spawn probability documentation",
	Action:      runGenerateItemDoc,
	Flags:       []cli.Flag{},
}

func runGenerateItemDoc(_ context.Context, _ *cli.Command) error {
	// raw.tomlを読み込む
	master, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	if err != nil {
		return fmt.Errorf("error loading raw.toml: %w", err)
	}

	// Markdownファイルを開く
	file, err := os.Create("docs/gen/item_tables.md")
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	// ヘッダー
	if _, err := file.WriteString("# アイテム出現テーブル\n\n"); err != nil {
		return fmt.Errorf("error writing header: %w", err)
	}
	if _, err := file.WriteString("各ステージ・階層ごとのアイテム出現確率を示す。\n\n"); err != nil {
		return fmt.Errorf("error writing description: %w", err)
	}

	// 各ItemTableを処理
	for _, table := range master.Raws.ItemTables {
		if err := generateTableDoc(file, table); err != nil {
			return err
		}
	}

	return nil
}

func generateTableDoc(file *os.File, table raw.ItemTable) error {
	if _, err := fmt.Fprintf(file, "## %s\n\n", table.Name); err != nil {
		return fmt.Errorf("error writing table name: %w", err)
	}

	// 各深度での最大アイテム数を計算
	maxItems := 0
	for depth := 1; depth <= consts.GameClearDepth; depth++ {
		probs := calculateProbabilities(table, depth)
		if len(probs) > maxItems {
			maxItems = len(probs)
		}
	}

	// tablewriterを初期化
	tw := tablewriter.NewTable(file,
		tablewriter.WithRenderer(renderer.NewMarkdown()),
	)

	// ヘッダーを設定（深度 + 最大アイテム数分のダミーカラム）
	header := make([]string, maxItems+1)
	header[0] = "深度"
	for i := 0; i < maxItems; i++ {
		header[i+1] = "-"
	}
	tw.Header(header)

	// 各深度の行データを収集
	var rows [][]any
	for depth := 1; depth <= consts.GameClearDepth; depth++ {
		// 各アイテムの出現確率を計算
		probs := calculateProbabilities(table, depth)

		// アイテム名をソート
		items := make([]string, 0, len(probs))
		for item := range probs {
			items = append(items, item)
		}
		sort.Strings(items)

		// 行データを作成
		row := make([]any, maxItems+1)
		row[0] = fmt.Sprintf("%d", depth)

		// アイテムと確率を追加
		idx := 1
		for _, item := range items {
			prob := probs[item]
			row[idx] = fmt.Sprintf("%s %.1f%%", item, prob*100)
			idx++
		}

		// 不足分は空欄で埋める
		for i := len(items); i < maxItems; i++ {
			row[idx] = "-"
			idx++
		}

		rows = append(rows, row)
	}

	if err := tw.Bulk(rows); err != nil {
		return fmt.Errorf("error adding bulk data: %w", err)
	}
	if err := tw.Render(); err != nil {
		return fmt.Errorf("error rendering table: %w", err)
	}

	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("error writing newline: %w", err)
	}
	return nil
}

func calculateProbabilities(table raw.ItemTable, depth int) map[string]float64 {
	// 深度範囲内のエントリをフィルタリング
	validEntries := make([]raw.ItemTableEntry, 0, len(table.Entries))
	for _, entry := range table.Entries {
		// MinDepthチェック（0は制限なし）
		if entry.MinDepth > 0 && depth < entry.MinDepth {
			continue
		}
		// MaxDepthチェック（0は制限なし）
		if entry.MaxDepth > 0 && depth > entry.MaxDepth {
			continue
		}
		validEntries = append(validEntries, entry)
	}

	// 総重みを計算
	var totalWeight float64
	for _, entry := range validEntries {
		totalWeight += entry.Weight
	}

	// 確率を計算
	probs := make(map[string]float64)
	if totalWeight > 0 {
		for _, entry := range validEntries {
			probs[entry.ItemName] = entry.Weight / totalWeight
		}
	}

	return probs
}
