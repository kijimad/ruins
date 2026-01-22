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

// CmdGenerateEnemyDoc は敵出現確率のドキュメントを生成するコマンド
var CmdGenerateEnemyDoc = &cli.Command{
	Name:        "generate-enemy-doc",
	Usage:       "generate-enemy-doc",
	Description: "Generate enemy spawn probability documentation",
	Action:      runGenerateEnemyDoc,
	Flags:       []cli.Flag{},
}

func runGenerateEnemyDoc(_ context.Context, _ *cli.Command) error {
	// raw.tomlを読み込む
	master, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	if err != nil {
		return fmt.Errorf("error loading raw.toml: %w", err)
	}

	// Markdownファイルを開く
	file, err := os.Create("docs/gen/enemy_tables.md")
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	// ヘッダー
	if _, err := file.WriteString("# 敵出現テーブル\n\n"); err != nil {
		return fmt.Errorf("error writing header: %w", err)
	}
	if _, err := file.WriteString("各ステージ・階層ごとの敵出現確率を示す。\n\n"); err != nil {
		return fmt.Errorf("error writing description: %w", err)
	}

	// 各EnemyTableを処理
	for _, table := range master.Raws.EnemyTables {
		if err := generateEnemyTableDoc(file, table); err != nil {
			return err
		}
	}

	return nil
}

func generateEnemyTableDoc(file *os.File, table raw.EnemyTable) error {
	if _, err := fmt.Fprintf(file, "## %s\n\n", table.Name); err != nil {
		return fmt.Errorf("error writing table name: %w", err)
	}

	// 各深度での最大敵数を計算
	maxEnemies := 0
	for depth := 1; depth <= consts.GameClearDepth; depth++ {
		probs := calculateEnemyProbabilities(table, depth)
		if len(probs) > maxEnemies {
			maxEnemies = len(probs)
		}
	}

	// tablewriterを初期化
	tw := tablewriter.NewTable(file,
		tablewriter.WithRenderer(renderer.NewMarkdown()),
	)

	// ヘッダーを設定（深度 + 最大敵数分のダミーカラム）
	header := make([]string, maxEnemies+1)
	header[0] = "深度"
	for i := 0; i < maxEnemies; i++ {
		header[i+1] = "-"
	}
	tw.Header(header)

	// 各深度の行データを収集
	var rows [][]any
	for depth := 1; depth <= consts.GameClearDepth; depth++ {
		// 各敵の出現確率を計算
		probs := calculateEnemyProbabilities(table, depth)

		// 敵名をソート
		enemies := make([]string, 0, len(probs))
		for enemy := range probs {
			enemies = append(enemies, enemy)
		}
		sort.Strings(enemies)

		// 行データを作成
		row := make([]any, maxEnemies+1)
		row[0] = fmt.Sprintf("%d", depth)

		// 敵と確率を追加
		idx := 1
		for _, enemy := range enemies {
			prob := probs[enemy]
			row[idx] = fmt.Sprintf("%s %.1f%%", enemy, prob*100)
			idx++
		}

		// 不足分は空欄で埋める
		for i := len(enemies); i < maxEnemies; i++ {
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

func calculateEnemyProbabilities(table raw.EnemyTable, depth int) map[string]float64 {
	// 深度範囲内のエントリをフィルタリング
	validEntries := make([]raw.EnemyTableEntry, 0, len(table.Entries))
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
			probs[entry.EnemyName] = entry.Weight / totalWeight
		}
	}

	return probs
}
