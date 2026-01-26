package maptemplate

import (
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
)

// ChunkTemplate はチャンクテンプレート定義
// すべてのマップ要素（小さな部品から大きなレイアウトまで）を表す
type ChunkTemplate struct {
	Name        string           `toml:"name"`   // キー
	Weight      int              `toml:"weight"` // 出現確率の重み
	Size        Size             // マップサイズ（名前から自動パース）
	Palettes    []string         `toml:"palettes"`     // 使用するパレットID
	Map         string           `toml:"map"`          // ASCIIマップ
	Placements  []ChunkPlacement `toml:"placements"`   // ネストされたチャンクの配置
	Exits       []ExitPlacement  `toml:"exits"`        // 出口の配置
	SpawnPoints []SpawnPoint     `toml:"spawn_points"` // スポーン地点の配置
}

// Size はマップのサイズ（幅×高さ）を表す
type Size struct {
	W int // 幅
	H int // 高さ
}

// String はサイズを "{幅}x{高さ}" 形式の文字列として返す
func (s Size) String() string {
	return fmt.Sprintf("%dx%d", s.W, s.H)
}

// ChunkPlacement はネストされたチャンクの配置情報
type ChunkPlacement struct {
	// チャンク名の配列（重み付き選択）
	// バリエーションの出し方には2つある
	// - マイナーな違い（内装配置の違いなど）: 同じ名前で作成する
	//   例: chunks = ["11x6_convenience_store"] → 2つの11x6_convenience_store, 11x6_convenience_store から選択する
	// - メジャーな違い（異なる用途の建物など）: 異なる名前を配列で列挙する
	//   例: chunks = ["11x6_convenience_store", "11x6_clinic"] → コンビニか整骨院のどちらかが選ばれる
	Chunks []string `toml:"chunks"`
	ID     string   `toml:"id"` // プレースホルダ識別子（右下に配置される1文字）
}

// ExitID は出口の識別子
type ExitID string

const (
	// ExitIDMain は次のフロアへのメイン出口を表す
	ExitIDMain ExitID = "exit"
	// ExitIDLeft は次のフロアへの出口（左側）を表す
	ExitIDLeft ExitID = "exit_left"
	// ExitIDCenter は次のフロアへの出口（中央）を表す
	ExitIDCenter ExitID = "exit_center"
	// ExitIDRight は次のフロアへの出口（右側）を表す
	ExitIDRight ExitID = "exit_right"
)

// ExitPlacement は出口の配置情報
type ExitPlacement struct {
	ExitID string `toml:"exit_id"` // 出口ID
	X      int    `toml:"x"`       // X座標
	Y      int    `toml:"y"`       // Y座標
}

// SpawnPoint はスポーン地点の配置情報
type SpawnPoint struct {
	X int `toml:"x"` // X座標
	Y int `toml:"y"` // Y座標
}

// ChunkTemplateFile はTOMLファイルのルート構造
type ChunkTemplateFile struct {
	Chunks []ChunkTemplate `toml:"chunk"`
}

// TemplateLoader はテンプレート定義の読み込みを担当する
// TODO: Loaderが保持するのはおかしい気もする。Resourcesなどに保存することを検討する。が、依存が大きくなるのも微妙である。TemplatePlanner以外じゃ使わないしな...
type TemplateLoader struct {
	chunkCache   map[string][]*ChunkTemplate // 同じ名前で複数のバリエーションをサポート
	paletteCache map[string]*Palette
}

// NewTemplateLoader はTemplateLoaderを生成する
func NewTemplateLoader() *TemplateLoader {
	return &TemplateLoader{
		chunkCache:   make(map[string][]*ChunkTemplate),
		paletteCache: make(map[string]*Palette),
	}
}

// Load はio.Readerからチャンクテンプレート定義を読み込む
func (l *TemplateLoader) Load(r io.Reader) ([]ChunkTemplate, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("テンプレート読み込みエラー: %w", err)
	}

	var file ChunkTemplateFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("テンプレートTOMLパースエラー: %w", err)
	}

	for i := range file.Chunks {
		// 名前からサイズを自動設定
		if err := l.parseSizeFromName(&file.Chunks[i]); err != nil {
			return nil, fmt.Errorf("テンプレート%d名前パースエラー: %w", i, err)
		}

		if err := l.validate(&file.Chunks[i]); err != nil {
			return nil, fmt.Errorf("テンプレート%d検証エラー: %w", i, err)
		}
	}

	return file.Chunks, nil
}

// LoadFile はTOMLファイルからチャンクテンプレート定義を読み込む
func (l *TemplateLoader) LoadFile(path string) ([]ChunkTemplate, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("テンプレートファイル読み込みエラー: %w", err)
	}
	defer func() { _ = f.Close() }()

	return l.Load(f)
}

// parseSizeFromName は名前から "{幅}x{高さ}_" 形式のサイズをパースする
// 例: "5x5_meeting_room" -> Size = [5, 5]
func (l *TemplateLoader) parseSizeFromName(t *ChunkTemplate) error {
	// 名前から "{幅}x{高さ}_" 形式を探す
	parts := strings.Split(t.Name, "_")
	if len(parts) == 0 {
		return fmt.Errorf("チャンク名が空です")
	}

	// 最初の部分が "{幅}x{高さ}" 形式かチェック
	sizePart := parts[0]
	dimensions := strings.Split(sizePart, "x")
	if len(dimensions) != 2 {
		return fmt.Errorf("チャンク名は '{幅}x{高さ}_名前' 形式である必要があります: %s", t.Name)
	}

	width, err := strconv.Atoi(dimensions[0])
	if err != nil {
		return fmt.Errorf("幅のパースに失敗: %s", dimensions[0])
	}

	height, err := strconv.Atoi(dimensions[1])
	if err != nil {
		return fmt.Errorf("高さのパースに失敗: %s", dimensions[1])
	}

	t.Size = Size{W: width, H: height}
	return nil
}

// validate はテンプレート定義の妥当性を検証する
func (l *TemplateLoader) validate(t *ChunkTemplate) error {
	if t.Name == "" {
		return fmt.Errorf("チャンク名（キー）が空です")
	}

	if t.Weight <= 0 {
		return fmt.Errorf("重みは正の整数である必要があります: %d", t.Weight)
	}

	if t.Size.W <= 0 || t.Size.H <= 0 {
		return fmt.Errorf("サイズは正の整数である必要があります: %s", t.Size)
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

	expectedWidth := utf8.RuneCountInString(lines[0])
	if expectedWidth == 0 {
		return fmt.Errorf("マップの行が空です")
	}

	for i, line := range lines {
		lineWidth := utf8.RuneCountInString(line)
		if lineWidth != expectedWidth {
			return fmt.Errorf("マップの行%dの長さが不一致です: 期待%d、実際%d", i, expectedWidth, lineWidth)
		}
	}

	actualWidth := expectedWidth
	actualHeight := len(lines)
	if actualWidth != t.Size.W || actualHeight != t.Size.H {
		return fmt.Errorf("マップの実サイズ[%d, %d]が定義サイズ%sと一致しません", actualWidth, actualHeight, t.Size)
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
// 同じ名前のチャンクが複数ある場合、すべて登録される（バリエーション）
func (l *TemplateLoader) LoadChunk(path string) error {
	templates, err := l.LoadFile(path)
	if err != nil {
		return fmt.Errorf("チャンク読み込みエラー: %w", err)
	}

	for i := range templates {
		name := templates[i].Name
		l.chunkCache[name] = append(l.chunkCache[name], &templates[i])
	}

	return nil
}

// GetChunks はキャッシュから指定名のすべてのチャンクバリエーションを取得する
func (l *TemplateLoader) GetChunks(chunkName string) ([]*ChunkTemplate, error) {
	chunks, ok := l.chunkCache[chunkName]
	if !ok || len(chunks) == 0 {
		return nil, fmt.Errorf("チャンク '%s' が見つかりません", chunkName)
	}
	return chunks, nil
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
			palette, err := paletteLoader.LoadFile(path)
			if err != nil {
				return fmt.Errorf("パレット読み込みエラー %s: %w", path, err)
			}

			l.paletteCache[palette.ID] = palette
		}
	}

	return nil
}

// LoadTemplateByName はテンプレート名で展開済みテンプレートとパレットを取得する
// 同じ名前のバリエーションが複数ある場合、重み付きランダム選択する
func (l *TemplateLoader) LoadTemplateByName(name string, seed uint64) (*ChunkTemplate, *Palette, error) {
	chunks, err := l.GetChunks(name)
	if err != nil {
		return nil, nil, err
	}

	// 重み付きランダム選択
	rng := rand.New(rand.NewPCG(seed, seed))
	template, err := l.selectChunkByWeightFromList(chunks, rng)
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

	if len(templateCopy.Placements) > 0 {
		expandedMap, err := templateCopy.ExpandWithPlacements(l, seed)
		if err != nil {
			return nil, nil, fmt.Errorf("チャンク展開エラー: %w", err)
		}
		templateCopy.Map = expandedMap
	}

	return &templateCopy, mergedPalette, nil
}

// selectChunkByWeight は重みづけランダム選択でチャンクを選択する
// 各チャンク名に複数のバリエーションがある場合、すべてを候補に含める
func (l *TemplateLoader) selectChunkByWeight(chunkNames []string, rng *rand.Rand) (*ChunkTemplate, error) {
	if len(chunkNames) == 0 {
		return nil, fmt.Errorf("チャンク候補が空です")
	}

	candidates := make([]*ChunkTemplate, 0)

	// 各チャンク名のすべてのバリエーションを候補に追加
	for _, name := range chunkNames {
		chunks, err := l.GetChunks(name)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, chunks...)
	}

	return l.selectChunkByWeightFromList(candidates, rng)
}

// selectChunkByWeightFromList はChunkTemplateのリストから重み付きランダム選択する
func (l *TemplateLoader) selectChunkByWeightFromList(candidates []*ChunkTemplate, rng *rand.Rand) (*ChunkTemplate, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("チャンク候補が空です")
	}

	totalWeight := 0
	for _, candidate := range candidates {
		totalWeight += candidate.Weight
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

// ExpandWithPlacements はplacements方式でチャンクを展開したマップ文字列を返す（CDDA方式）
func (t *ChunkTemplate) ExpandWithPlacements(loader *TemplateLoader, seed uint64) (string, error) {
	if len(t.Placements) == 0 {
		return t.Map, nil
	}

	// プレースホルダ検証を実行
	if err := t.validatePlaceholders(loader); err != nil {
		return "", fmt.Errorf("プレースホルダ検証エラー: %w", err)
	}

	visiting := make(map[string]bool)
	return t.expandWithPlacementsRecursive(loader, seed, 0, visiting)
}

// expandWithPlacementsRecursive はチャンクを再帰的に展開する
func (t *ChunkTemplate) expandWithPlacementsRecursive(loader *TemplateLoader, seed uint64, depth int, visiting map[string]bool) (string, error) {
	const maxDepth = 10

	if depth > maxDepth {
		return "", fmt.Errorf("チャンク展開の深度が制限(%d)を超えました", maxDepth)
	}

	if visiting[t.Name] {
		return "", fmt.Errorf("チャンクの循環参照を検出しました: %s", t.Name)
	}

	if len(t.Placements) == 0 {
		return t.Map, nil
	}

	visiting[t.Name] = true
	defer delete(visiting, t.Name)

	lines := t.GetMapLines()
	result := make([]string, len(lines))
	copy(result, lines)

	rng := rand.New(rand.NewPCG(seed, seed))

	// 最初に全ての識別子の位置を検出（元のマップから）
	type placementInfo struct {
		x, y, width, height int
	}
	placementPositions := make([]placementInfo, len(t.Placements))

	for idx, placement := range t.Placements {
		if placement.ID == "" {
			return "", fmt.Errorf("placement %d: ID が指定されていません", idx)
		}

		x, y, width, height, err := findPlaceholderRegionByID(lines, placement.ID)
		if err != nil {
			return "", fmt.Errorf("placement %d (ID='%s'): %w", idx, placement.ID, err)
		}

		placementPositions[idx] = placementInfo{x: x, y: y, width: width, height: height}
	}

	// 各placements定義を処理
	for idx, placement := range t.Placements {
		// チャンクを重み付き選択
		selectedChunk, err := loader.selectChunkByWeight(placement.Chunks, rng)
		if err != nil {
			return "", fmt.Errorf("チャンク選択エラー (placement %d): %w", idx, err)
		}

		// 再帰的に展開
		expandedChunkMap, err := selectedChunk.expandWithPlacementsRecursive(loader, seed+uint64(idx)*1000, depth+1, visiting)
		if err != nil {
			return "", err
		}

		chunkLines := strings.Split(strings.TrimSpace(expandedChunkMap), "\n")

		// サイズチェック
		if len(chunkLines) != selectedChunk.Size.H {
			return "", fmt.Errorf("チャンク '%s' の高さが不一致: 期待%d、実際%d", selectedChunk.Name, selectedChunk.Size.H, len(chunkLines))
		}
		if len(chunkLines) > 0 && len(chunkLines[0]) != selectedChunk.Size.W {
			return "", fmt.Errorf("チャンク '%s' の幅が不一致: 期待%d、実際%d", selectedChunk.Name, selectedChunk.Size.W, len(chunkLines[0]))
		}

		// 事前に検出した位置情報を取得
		pos := placementPositions[idx]
		placementX, placementY := pos.x, pos.y
		width, height := pos.width, pos.height

		// サイズの完全一致を検証
		if width != selectedChunk.Size.W || height != selectedChunk.Size.H {
			return "", fmt.Errorf(
				"親チャンク '%s': placement %d (ID='%s', 子チャンク='%s'): プレースホルダ領域[%d,%d] (位置[%d,%d])とチャンクサイズ%s が不一致",
				t.Name, idx, placement.ID, selectedChunk.Name, width, height, placementX, placementY, selectedChunk.Size,
			)
		}

		// チャンクを配置
		for cy := 0; cy < selectedChunk.Size.H; cy++ {
			for cx := 0; cx < selectedChunk.Size.W; cx++ {
				targetX := placementX + cx
				targetY := placementY + cy

				if targetY < len(result) && targetX < len(result[targetY]) {
					oldLine := result[targetY]
					newLine := oldLine[:targetX] + string(chunkLines[cy][cx]) + oldLine[targetX+1:]
					result[targetY] = newLine
				}
			}
		}
	}

	expandedMap := strings.Join(result, "\n")

	// 展開後にプレースホルダが残っていないか検証
	if err := validateNoPlaceholdersRemaining(expandedMap); err != nil {
		return "", err
	}

	return expandedMap, nil
}

// validateNoPlaceholdersRemaining は展開後のマップにプレースホルダが残っていないか検証する
func validateNoPlaceholdersRemaining(mapStr string) error {
	const placeholder = '@'
	lines := strings.Split(strings.TrimSpace(mapStr), "\n")

	for y, line := range lines {
		for x, ch := range line {
			if ch == placeholder {
				return fmt.Errorf("展開後のマップに未展開のプレースホルダ '@' が残っています (位置: x=%d, y=%d)", x, y)
			}
		}
	}

	return nil
}

// findPlaceholderRegionByID は識別子からプレースホルダ領域(矩形)を検出する
// 識別子はプレースホルダ領域の右下に配置されている
// 戻り値: (左上X, 左上Y, 幅, 高さ, エラー)
func findPlaceholderRegionByID(lines []string, id string) (int, int, int, int, error) {
	if len(id) != 1 {
		return 0, 0, 0, 0, fmt.Errorf("識別子は1文字である必要があります: %q", id)
	}

	idChar := rune(id[0])
	const placeholder = '@'

	// 識別子の位置を探す
	idX, idY, err := findIdentifierPosition(lines, idChar)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("識別子 '%s' が見つかりません", id)
	}

	// 左端を探す
	startX := findLeftEdge(lines, idY, idX, placeholder, idChar)

	// 上端を探す
	tempWidth := calculateRowWidth(lines[idY], startX, placeholder, idChar)
	startY := findTopEdge(lines, idY, startX, tempWidth, placeholder, idChar)

	// 幅を計算
	width := calculateRowWidth(lines[startY], startX, placeholder, idChar)

	// 高さを計算
	height := calculateHeight(lines, startY, startX, placeholder, idChar)

	// 矩形領域全体が@または識別子で埋まっているか検証
	if err := validateRectangle(lines, id, startX, startY, width, height, placeholder, idChar); err != nil {
		return 0, 0, 0, 0, err
	}

	return startX, startY, width, height, nil
}

// findIdentifierPosition は識別子の位置を探す
func findIdentifierPosition(lines []string, idChar rune) (int, int, error) {
	for y, line := range lines {
		for x, ch := range line {
			if ch == idChar {
				return x, y, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("not found")
}

// findLeftEdge は左端を探す
func findLeftEdge(lines []string, idY, idX int, placeholder, idChar rune) int {
	startX := idX
	for startX > 0 {
		ch := rune(lines[idY][startX-1])
		if ch == placeholder || ch == idChar {
			startX--
		} else {
			break
		}
	}
	return startX
}

// calculateRowWidth は行の幅を計算
func calculateRowWidth(line string, startX int, placeholder, idChar rune) int {
	width := 0
	for x := startX; x < len(line); x++ {
		ch := rune(line[x])
		if ch == placeholder || ch == idChar {
			width++
		} else {
			break
		}
	}
	return width
}

// findTopEdge は上端を探す
func findTopEdge(lines []string, idY, startX, tempWidth int, placeholder, idChar rune) int {
	startY := idY
	for startY > 0 {
		if startX+tempWidth > len(lines[startY-1]) {
			break
		}
		allMatch := true
		for x := startX; x < startX+tempWidth; x++ {
			ch := rune(lines[startY-1][x])
			if ch != placeholder && ch != idChar {
				allMatch = false
				break
			}
		}
		if allMatch {
			startY--
		} else {
			break
		}
	}
	return startY
}

// calculateHeight は高さを計算
func calculateHeight(lines []string, startY, startX int, placeholder, idChar rune) int {
	height := 0
	for y := startY; y < len(lines) && startX < len(lines[y]); y++ {
		ch := rune(lines[y][startX])
		if ch == placeholder || ch == idChar {
			height++
		} else {
			break
		}
	}
	return height
}

// validateRectangle は矩形領域全体が@または識別子で埋まっているか検証
func validateRectangle(lines []string, id string, startX, startY, width, height int, placeholder, idChar rune) error {
	for dy := 0; dy < height; dy++ {
		y := startY + dy
		if y >= len(lines) {
			return fmt.Errorf("識別子 '%s': 矩形領域が不完全です (y=%d が範囲外)", id, y)
		}
		for dx := 0; dx < width; dx++ {
			x := startX + dx
			if x >= len(lines[y]) {
				return fmt.Errorf("識別子 '%s': 矩形領域が不完全です (x=%d, y=%d が範囲外)", id, x, y)
			}
			ch := rune(lines[y][x])
			if ch != placeholder && ch != idChar {
				return fmt.Errorf(
					"識別子 '%s': プレースホルダ領域[%d,%d]に不正な文字 '%c' があります（期待: '@' または '%s'）",
					id, x, y, ch, id,
				)
			}
		}
	}
	return nil
}

// validatePlaceholders はプレースホルダ(@)とplacementsの整合性を検証する
func (t *ChunkTemplate) validatePlaceholders(loader *TemplateLoader) error {
	lines := t.GetMapLines()

	// 各placements定義をチェック
	for idx, placement := range t.Placements {
		// チャンクの最初のバリエーションからサイズを取得（すべて同じサイズのはず）
		chunks, err := loader.GetChunks(placement.Chunks[0])
		if err != nil {
			return fmt.Errorf("placement %d: %w", idx, err)
		}
		if len(chunks) == 0 {
			return fmt.Errorf("placement %d: チャンク '%s' が見つかりません", idx, placement.Chunks[0])
		}

		expectedWidth := chunks[0].Size.W
		expectedHeight := chunks[0].Size.H

		// 識別子から位置を検出して検証
		if placement.ID == "" {
			return fmt.Errorf("placement %d (%s): ID が指定されていません", idx, placement.Chunks[0])
		}

		x, y, width, height, err := findPlaceholderRegionByID(lines, placement.ID)
		if err != nil {
			return fmt.Errorf("placement %d (%s): %w", idx, placement.Chunks[0], err)
		}

		// サイズの完全一致を検証
		if width != expectedWidth || height != expectedHeight {
			return fmt.Errorf(
				"親チャンク '%s': placement %d (ID='%s', 子チャンク='%s'): プレースホルダ領域のサイズが不一致: 領域[%d,%d] (位置[%d,%d])、チャンクサイズ%s",
				t.Name, idx, placement.ID, placement.Chunks[0], width, height, x, y, chunks[0].Size,
			)
		}
	}

	return nil
}
