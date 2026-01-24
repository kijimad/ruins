package maptemplate

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ChunkTemplate はチャンクテンプレート定義
// すべてのマップ要素（小さな部品から大きなレイアウトまで）を表す
type ChunkTemplate struct {
	Name         string              `toml:"name"`          // キー（例: "bedroom", "office_building", "small_town"）
	Weight       int                 `toml:"weight"`        // 出現確率の重み
	Size         [2]int              `toml:"size"`          // [幅, 高さ]
	Palettes     []string            `toml:"palettes"`      // 使用するパレットID
	Map          string              `toml:"map"`           // ASCIIマップ
	Chunks       []string            `toml:"chunks"`        // 使用するチャンク名一覧
	ChunkMapping map[string][]string `toml:"chunk_mapping"` // マップ文字 -> チャンク名の配列（重みづけランダム選択）
}

// ChunkTemplateFile はTOMLファイルのルート構造
type ChunkTemplateFile struct {
	Chunks []ChunkTemplate `toml:"chunk"`
}

// TemplateLoader はテンプレート定義の読み込みを担当する
type TemplateLoader struct {
	chunkCache   map[string]*ChunkTemplate
	paletteCache map[string]*Palette
}

// NewTemplateLoader はTemplateLoaderを生成する
func NewTemplateLoader() *TemplateLoader {
	return &TemplateLoader{
		chunkCache:   make(map[string]*ChunkTemplate),
		paletteCache: make(map[string]*Palette),
	}
}

// LoadFromFile はTOMLファイルからチャンクテンプレート定義を読み込む
func (l *TemplateLoader) LoadFromFile(path string) ([]ChunkTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("テンプレートファイル読み込みエラー: %w", err)
	}

	var file ChunkTemplateFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("テンプレートTOMLパースエラー: %w", err)
	}

	for i := range file.Chunks {
		if err := l.validate(&file.Chunks[i]); err != nil {
			return nil, fmt.Errorf("テンプレート%d検証エラー: %w", i, err)
		}
	}

	return file.Chunks, nil
}

// validate はテンプレート定義の妥当性を検証する
func (l *TemplateLoader) validate(t *ChunkTemplate) error {
	if t.Name == "" {
		return fmt.Errorf("チャンク名（キー）が空です")
	}

	if t.Weight <= 0 {
		return fmt.Errorf("重みは正の整数である必要があります: %d", t.Weight)
	}

	if t.Size[0] <= 0 || t.Size[1] <= 0 {
		return fmt.Errorf("サイズは正の整数である必要があります: %v", t.Size)
	}

	return l.validateMap(t)
}

// validateMap はマップの妥当性を検証する
func (l *TemplateLoader) validateMap(t *ChunkTemplate) error {
	if t.Map == "" {
		return fmt.Errorf("マップが空です")
	}

	lines := strings.Split(strings.TrimSpace(t.Map), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("マップが空です")
	}

	expectedWidth := len(lines[0])
	if expectedWidth == 0 {
		return fmt.Errorf("マップの行が空です")
	}

	for i, line := range lines {
		if len(line) != expectedWidth {
			return fmt.Errorf("マップの行%dの長さが不一致です: 期待%d、実際%d", i, expectedWidth, len(line))
		}
	}

	actualWidth := expectedWidth
	actualHeight := len(lines)
	if actualWidth != t.Size[0] || actualHeight != t.Size[1] {
		return fmt.Errorf("マップの実サイズ[%d, %d]が定義サイズ%vと一致しません", actualWidth, actualHeight, t.Size)
	}

	return nil
}

// GetMapLines はマップを行の配列として取得する
func (t *ChunkTemplate) GetMapLines() []string {
	return strings.Split(strings.TrimSpace(t.Map), "\n")
}

// GetCharAt は指定座標の文字を取得する
func (t *ChunkTemplate) GetCharAt(x, y int) (string, error) {
	lines := t.GetMapLines()
	if y < 0 || y >= len(lines) {
		return "", fmt.Errorf("y座標が範囲外です: %d", y)
	}
	if x < 0 || x >= len(lines[y]) {
		return "", fmt.Errorf("x座標が範囲外です: %d", x)
	}
	return string(lines[y][x]), nil
}

// LoadChunk はチャンク定義を読み込んでキャッシュする
func (l *TemplateLoader) LoadChunk(path string) error {
	templates, err := l.LoadFromFile(path)
	if err != nil {
		return fmt.Errorf("チャンク読み込みエラー: %w", err)
	}

	for i := range templates {
		l.chunkCache[templates[i].Name] = &templates[i]
	}

	return nil
}

// GetChunk はキャッシュからチャンクを取得する
func (l *TemplateLoader) GetChunk(chunkName string) (*ChunkTemplate, error) {
	chunk, ok := l.chunkCache[chunkName]
	if !ok {
		return nil, fmt.Errorf("チャンク '%s' が見つかりません", chunkName)
	}
	return chunk, nil
}

// RegisterAllChunks は指定されたディレクトリ配下のすべての.tomlファイルをチャンクとして登録する
func (l *TemplateLoader) RegisterAllChunks(directories []string) error {
	for _, dir := range directories {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("ディレクトリ読み込みエラー %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
				continue
			}

			path := dir + "/" + entry.Name()
			if err := l.LoadChunk(path); err != nil {
				return err
			}
		}
	}

	return nil
}

// RegisterAllPalettes は指定されたディレクトリ配下のすべての.tomlファイルをパレットとして登録する
func (l *TemplateLoader) RegisterAllPalettes(directories []string) error {
	paletteLoader := NewPaletteLoader()

	for _, dir := range directories {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("ディレクトリ読み込みエラー %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
				continue
			}

			path := dir + "/" + entry.Name()
			palette, err := paletteLoader.LoadFromFile(path)
			if err != nil {
				return fmt.Errorf("パレット読み込みエラー %s: %w", path, err)
			}

			l.paletteCache[palette.ID] = palette
		}
	}

	return nil
}

// LoadTemplateByName はテンプレート名で展開済みテンプレートとパレットを取得する
func (l *TemplateLoader) LoadTemplateByName(name string, seed uint64) (*ChunkTemplate, *Palette, error) {
	template, err := l.GetChunk(name)
	if err != nil {
		return nil, nil, err
	}

	templateCopy := *template

	palettes := make([]*Palette, 0, len(templateCopy.Palettes))
	for _, paletteName := range templateCopy.Palettes {
		palette, ok := l.paletteCache[paletteName]
		if !ok {
			return nil, nil, fmt.Errorf("パレット '%s' が見つかりません（RegisterAllPalettesで事前登録が必要）", paletteName)
		}
		palettes = append(palettes, palette)
	}

	mergedPalette := MergePalettes(palettes...)

	if len(templateCopy.ChunkMapping) > 0 {
		expandedMap, err := templateCopy.ExpandWithChunks(l, seed)
		if err != nil {
			return nil, nil, fmt.Errorf("チャンク展開エラー: %w", err)
		}
		templateCopy.Map = expandedMap
	}

	return &templateCopy, mergedPalette, nil
}

// selectChunkByWeight は重みづけランダム選択でチャンクを選択する
func (l *TemplateLoader) selectChunkByWeight(chunkNames []string, rng *rand.Rand) (*ChunkTemplate, error) {
	if len(chunkNames) == 0 {
		return nil, fmt.Errorf("チャンク候補が空です")
	}

	candidates := make([]*ChunkTemplate, 0, len(chunkNames))
	totalWeight := 0

	for _, name := range chunkNames {
		chunk, err := l.GetChunk(name)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, chunk)
		totalWeight += chunk.Weight
	}

	if totalWeight == 0 {
		return nil, fmt.Errorf("合計重みが0です")
	}

	r := rng.IntN(totalWeight)
	cumulative := 0
	for _, candidate := range candidates {
		cumulative += candidate.Weight
		if r < cumulative {
			return candidate, nil
		}
	}

	return candidates[len(candidates)-1], nil
}

// ExpandWithChunks はチャンクを展開したマップ文字列を返す
func (t *ChunkTemplate) ExpandWithChunks(loader *TemplateLoader, seed uint64) (string, error) {
	if len(t.ChunkMapping) == 0 {
		return t.Map, nil
	}

	visiting := make(map[string]bool)
	return t.expandWithChunksRecursive(loader, seed, 0, visiting)
}

// expandWithChunksRecursive はチャンクを再帰的に展開する
func (t *ChunkTemplate) expandWithChunksRecursive(loader *TemplateLoader, seed uint64, depth int, visiting map[string]bool) (string, error) {
	const maxDepth = 10

	if depth > maxDepth {
		return "", fmt.Errorf("チャンク展開の深度が制限(%d)を超えました", maxDepth)
	}

	if visiting[t.Name] {
		return "", fmt.Errorf("チャンクの循環参照を検出しました: %s", t.Name)
	}

	if len(t.ChunkMapping) == 0 {
		return t.Map, nil
	}

	visiting[t.Name] = true
	defer delete(visiting, t.Name)

	lines := t.GetMapLines()

	rng := rand.New(rand.NewPCG(seed, seed))

	// フェーズ1: チャンク領域を元のマップから検出し、展開したチャンクを準備
	type chunkPlacement struct {
		x, y          int
		width, height int
		expandedLines []string
	}
	var placements []chunkPlacement

	processedRegions := make(map[string]bool)

	for y := 0; y < len(lines); y++ {
		for x := 0; x < len(lines[y]); x++ {
			char := string(lines[y][x])

			chunkNames, ok := t.ChunkMapping[char]
			if !ok {
				continue
			}

			regionKey := fmt.Sprintf("%d,%d,%s", x, y, char)
			if processedRegions[regionKey] {
				continue
			}

			width, height := t.detectChunkRegion(lines, x, y, char)

			selectedChunk, err := loader.selectChunkByWeight(chunkNames, rng)
			if err != nil {
				return "", fmt.Errorf("チャンク選択エラー (%s): %w", char, err)
			}

			if selectedChunk.Size[0] != width || selectedChunk.Size[1] != height {
				return "", fmt.Errorf("チャンク '%s' のサイズ%vが配置領域[%d, %d]と一致しません (文字='%s', 座標=(%d,%d))", selectedChunk.Name, selectedChunk.Size, width, height, char, x, y)
			}

			expandedChunkMap, err := selectedChunk.expandWithChunksRecursive(loader, seed+uint64(x)+uint64(y)*1000, depth+1, visiting)
			if err != nil {
				return "", err
			}

			chunkLines := strings.Split(strings.TrimSpace(expandedChunkMap), "\n")
			placements = append(placements, chunkPlacement{
				x:             x,
				y:             y,
				width:         width,
				height:        height,
				expandedLines: chunkLines,
			})

			// この領域を処理済みとしてマーク
			for cy := 0; cy < height; cy++ {
				for cx := 0; cx < width; cx++ {
					regionKey := fmt.Sprintf("%d,%d,%s", x+cx, y+cy, char)
					processedRegions[regionKey] = true
				}
			}
		}
	}

	// フェーズ2: 展開されたチャンクを配置
	result := make([]string, len(lines))
	copy(result, lines)

	for _, placement := range placements {
		for cy := 0; cy < placement.height; cy++ {
			for cx := 0; cx < placement.width; cx++ {
				targetX := placement.x + cx
				targetY := placement.y + cy

				if targetY < len(result) && targetX < len(result[targetY]) {
					oldLine := result[targetY]
					newLine := oldLine[:targetX] + string(placement.expandedLines[cy][cx]) + oldLine[targetX+1:]
					result[targetY] = newLine
				}
			}
		}
	}

	return strings.Join(result, "\n"), nil
}

// detectChunkRegion はチャンク領域のサイズを検出する
func (t *ChunkTemplate) detectChunkRegion(lines []string, startX, startY int, targetChar string) (width, height int) {
	if startY >= len(lines) || startX >= len(lines[startY]) {
		return 0, 0
	}

	// 幅を検出
	width = 0
	for x := startX; x < len(lines[startY]) && string(lines[startY][x]) == targetChar; x++ {
		width++
	}

	// 高さを検出（各行で幅全体が同じ文字であることを確認）
	height = 0
	for y := startY; y < len(lines); y++ {
		// この行の幅全体をチェック
		valid := true
		for x := startX; x < startX+width && x < len(lines[y]); x++ {
			if string(lines[y][x]) != targetChar {
				valid = false
				break
			}
		}
		if !valid || startX+width > len(lines[y]) {
			break
		}
		height++
	}

	return width, height
}
