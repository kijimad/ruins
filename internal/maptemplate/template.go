package maptemplate

import (
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kijimaD/ruins/assets"
	"github.com/pelletier/go-toml/v2"
)

// placeholderChar はプレースホルダ領域を示す文字
const placeholderChar = '@'

// placeholderStr はプレースホルダの文字列表現
const placeholderStr = "@"

// MapCell はマップの1セルを表す解決済みオブジェクト。
// 各チャンクがパレットを独立に解決した結果を保持する
type MapCell struct {
	Terrain string // 地形名
	Prop    string // Prop ID（なければ空文字）
	NPC     string // NPC ID（なければ空文字）
}

// ResolveMapCells はマップ文字列をパレットで解決し、セル配列に変換する。
// パレットで解決できない文字は文字自体をTerrain名として使用する
func ResolveMapCells(mapStr string, palette *Palette) [][]MapCell {
	lines := strings.Split(strings.TrimSpace(mapStr), "\n")
	cells := make([][]MapCell, len(lines))
	for y, line := range lines {
		runes := []rune(line)
		cells[y] = make([]MapCell, len(runes))
		for x, ch := range runes {
			charStr := string(ch)
			var cell MapCell
			if palette != nil {
				if terrain, ok := palette.GetTerrain(charStr); ok {
					cell.Terrain = terrain
				}
				if prop, ok := palette.GetProp(charStr); ok {
					cell.Prop = prop
				}
				if npc, ok := palette.GetNPC(charStr); ok {
					cell.NPC = npc
				}
			}
			// パレットで解決できなかった場合、文字自体をTerrain名として使用する
			if cell.Terrain == "" {
				cell.Terrain = charStr
			}
			cells[y][x] = cell
		}
	}
	return cells
}

// FormatResolvedMap はセル配列を可読文字列に変換する。
// 各セルは "terrain" 形式で、prop/NPCがあれば "terrain:prop" や "terrain:prop:npc" になる
func FormatResolvedMap(cells [][]MapCell) string {
	var sb strings.Builder
	for y, row := range cells {
		if y > 0 {
			sb.WriteByte('\n')
		}
		for x, cell := range row {
			if x > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(cell.Terrain)
			if cell.Prop != "" || cell.NPC != "" {
				sb.WriteByte(':')
				sb.WriteString(cell.Prop)
			}
			if cell.NPC != "" {
				sb.WriteByte(':')
				sb.WriteString(cell.NPC)
			}
		}
	}
	return sb.String()
}

// ChunkTemplate はチャンクテンプレート定義
// すべてのマップ要素（小さな部品から大きなレイアウトまで）を表す
type ChunkTemplate struct {
	Name        string           `toml:"name"`   // キー
	Weight      int              `toml:"weight"` // 出現確率の重み
	Size        Size             // マップサイズ（名前から自動パース）
	Palettes    []string         `toml:"palettes"`      // 使用するパレットID
	Map         string           `toml:"map,multiline"` // ASCIIマップ
	Placements  []ChunkPlacement `toml:"placements"`    // ネストされたチャンクの配置
	SpawnPoints []SpawnPoint     `toml:"spawn_points"`  // スポーン地点の配置
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

// SpawnPoint はスポーン地点の配置情報
// TOMLから読み込んだ後、mapplannerで使用される
type SpawnPoint struct {
	X int `toml:"x" json:"x"` // X座標
	Y int `toml:"y" json:"y"` // Y座標
}

// ChunkTemplateFile はTOMLファイルのルート構造
type ChunkTemplateFile struct {
	Chunks []ChunkTemplate `toml:"chunk"`
}

// MarshalChunkTemplateFile はChunkTemplateFileをTOMLバイト列にエンコードする
func MarshalChunkTemplateFile(file ChunkTemplateFile) ([]byte, error) {
	return toml.Marshal(file)
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
	f, err := assets.FS.Open(path)
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

// RegisterChunk はチャンクテンプレートをキャッシュに登録する
func (l *TemplateLoader) RegisterChunk(t *ChunkTemplate) {
	l.chunkCache[t.Name] = append(l.chunkCache[t.Name], t)
}

// RegisterPalette はパレットをキャッシュに登録する
func (l *TemplateLoader) RegisterPalette(p *Palette) {
	l.paletteCache[p.ID] = p
}

// RegisterAllChunks は指定されたディレクトリ配下のすべての.tomlファイルをチャンクとして登録する
func (l *TemplateLoader) RegisterAllChunks(directories []string) error {
	for _, dir := range directories {
		entries, err := fs.ReadDir(assets.FS, dir)
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
		entries, err := fs.ReadDir(assets.FS, dir)
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

// LoadTemplateByName はテンプレート名で展開済みテンプレートとパレットと解決済みセル配列を取得する。
// 同じ名前のバリエーションが複数ある場合、重み付きランダム選択する。
// 各チャンクは自身のパレットで独立に解決する
func (l *TemplateLoader) LoadTemplateByName(name string, seed uint64) (*ChunkTemplate, *Palette, [][]MapCell, error) {
	chunks, err := l.GetChunks(name)
	if err != nil {
		return nil, nil, nil, err
	}

	// 重み付きランダム選択
	rng := rand.New(rand.NewPCG(seed, seed))
	template, err := l.selectChunkByWeightFromList(chunks, rng)
	if err != nil {
		return nil, nil, nil, err
	}

	templateCopy := *template

	palettes := make([]*Palette, 0, len(templateCopy.Palettes))
	for _, paletteName := range templateCopy.Palettes {
		palette, ok := l.paletteCache[paletteName]
		if !ok {
			return nil, nil, nil, fmt.Errorf("パレット '%s' が見つかりません（RegisterAllPalettesで事前登録が必要）", paletteName)
		}
		palettes = append(palettes, palette)
	}

	mergedPalette := MergePalettes(palettes...)

	// セル配列に展開する
	resolvedMap, err := templateCopy.ExpandWithPlacements(l, seed)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("チャンク展開エラー: %w", err)
	}

	return &templateCopy, mergedPalette, resolvedMap, nil
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

// buildMergedPalette はパレット名のリストからマージ済みパレットを構築する
func (l *TemplateLoader) buildMergedPalette(paletteNames []string) *Palette {
	if len(paletteNames) == 0 {
		return nil
	}
	palettes := make([]*Palette, 0, len(paletteNames))
	for _, name := range paletteNames {
		if p, ok := l.paletteCache[name]; ok {
			palettes = append(palettes, p)
		}
	}
	if len(palettes) == 0 {
		return nil
	}
	return MergePalettes(palettes...)
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

// ExpandWithPlacements はplacements方式でチャンクを展開し、解決済みセル配列を返す。
// 各子チャンクは自身のパレットで独立に解決してからオーバーレイする（CDDA方式）
func (t *ChunkTemplate) ExpandWithPlacements(loader *TemplateLoader, seed uint64) ([][]MapCell, error) {
	visiting := make(map[string]bool)
	return t.expandWithPlacementsRecursive(loader, seed, 0, visiting)
}

// expandWithPlacementsRecursive はチャンクを再帰的に展開し、セル配列を返す。
// 各チャンクは自身のパレットで独立に解決する
func (t *ChunkTemplate) expandWithPlacementsRecursive(loader *TemplateLoader, seed uint64, depth int, visiting map[string]bool) ([][]MapCell, error) {
	const maxDepth = 10

	if depth > maxDepth {
		return nil, fmt.Errorf("チャンク展開の深度が制限(%d)を超えました", maxDepth)
	}

	if visiting[t.Name] {
		return nil, fmt.Errorf("チャンクの循環参照を検出しました: %s", t.Name)
	}

	// このチャンクのパレットで文字をセルに解決する
	palette := loader.buildMergedPalette(t.Palettes)
	cells := ResolveMapCells(t.Map, palette)

	if len(t.Placements) == 0 {
		return cells, nil
	}

	visiting[t.Name] = true
	defer delete(visiting, t.Name)

	lines := t.GetMapLines()
	rng := rand.New(rand.NewPCG(seed, seed))

	// 最初に全ての識別子の位置を検出（元のマップから）
	placementRegions := make([][]placeholderRegion, len(t.Placements))

	for idx, placement := range t.Placements {
		if placement.ID == "" {
			return nil, fmt.Errorf("placement %d: ID が指定されていません", idx)
		}

		regions, err := findAllPlaceholderRegionsByID(lines, placement.ID)
		if err != nil {
			return nil, fmt.Errorf("placement %d (ID='%s'): %w", idx, placement.ID, err)
		}

		placementRegions[idx] = regions
	}

	// 各placements定義を処理。同じIDが複数箇所にある場合はそれぞれ独立に展開する
	for idx, placement := range t.Placements {
		for regionIdx, region := range placementRegions[idx] {
			// チャンクを重み付き選択（領域ごとに独立して選択する）
			selectedChunk, err := loader.selectChunkByWeight(placement.Chunks, rng)
			if err != nil {
				return nil, fmt.Errorf("チャンク選択エラー (placement %d, 領域 %d): %w", idx, regionIdx, err)
			}

			// 再帰的に展開（子は自身のパレットで解決する）
			regionSeed := seed + uint64(idx)*1000 + uint64(regionIdx)*100
			childCells, err := selectedChunk.expandWithPlacementsRecursive(loader, regionSeed, depth+1, visiting)
			if err != nil {
				return nil, err
			}

			// サイズチェック
			if len(childCells) != selectedChunk.Size.H {
				return nil, fmt.Errorf("チャンク '%s' の高さが不一致: 期待%d、実際%d", selectedChunk.Name, selectedChunk.Size.H, len(childCells))
			}
			if len(childCells) > 0 && len(childCells[0]) != selectedChunk.Size.W {
				return nil, fmt.Errorf("チャンク '%s' の幅が不一致: 期待%d、実際%d", selectedChunk.Name, selectedChunk.Size.W, len(childCells[0]))
			}

			// サイズの完全一致を検証
			if region.width != selectedChunk.Size.W || region.height != selectedChunk.Size.H {
				return nil, fmt.Errorf(
					"親チャンク '%s': placement %d (ID='%s', 子チャンク='%s'): プレースホルダ領域[%d,%d] (位置[%d,%d])とチャンクサイズ%s が不一致",
					t.Name, idx, placement.ID, selectedChunk.Name, region.width, region.height, region.x, region.y, selectedChunk.Size,
				)
			}

			// 子のセルを親のセルにオーバーレイする
			for cy := range selectedChunk.Size.H {
				for cx := range selectedChunk.Size.W {
					targetX := region.x + cx
					targetY := region.y + cy
					if targetY < len(cells) && targetX < len(cells[targetY]) {
						cells[targetY][targetX] = childCells[cy][cx]
					}
				}
			}
		}
	}

	// 展開後にTerrainが空のセル（未解決プレースホルダ）がないか検証
	if err := validateNoPlaceholdersRemaining(cells); err != nil {
		return nil, err
	}

	return cells, nil
}

// validateNoPlaceholdersRemaining は展開後のセル配列にプレースホルダが残っていないか検証する
func validateNoPlaceholdersRemaining(cells [][]MapCell) error {
	for y, row := range cells {
		for x, cell := range row {
			if cell.Terrain == placeholderStr {
				return fmt.Errorf("展開後のマップに未展開のプレースホルダ '@' が残っています (位置: x=%d, y=%d)", x, y)
			}
		}
	}
	return nil
}

// placeholderRegion はプレースホルダ領域の位置とサイズ
type placeholderRegion struct {
	x, y, width, height int
}

// findAllPlaceholderRegionsByID は識別子に一致する全てのプレースホルダ領域を検出する。
// 同じIDが複数箇所にある場合、それぞれ独立した領域として返す
func findAllPlaceholderRegionsByID(lines []string, id string) ([]placeholderRegion, error) {
	if len(id) != 1 {
		return nil, fmt.Errorf("識別子は1文字である必要があります: %q", id)
	}

	idChar := rune(id[0])

	positions := findAllIdentifierPositions(lines, idChar)
	if len(positions) == 0 {
		return nil, fmt.Errorf("識別子 '%s' が見つかりません", id)
	}

	var regions []placeholderRegion
	for _, pos := range positions {
		idX, idY := pos[0], pos[1]

		startX := findLeftEdge(lines, idY, idX, placeholderChar, idChar)
		tempWidth := calculateRowWidth(lines[idY], startX, placeholderChar, idChar)
		startY := findTopEdge(lines, idY, startX, tempWidth, placeholderChar, idChar)
		width := calculateRowWidth(lines[startY], startX, placeholderChar, idChar)
		height := calculateHeight(lines, startY, startX, placeholderChar, idChar)

		if err := validateRectangle(lines, id, startX, startY, width, height, placeholderChar, idChar); err != nil {
			return nil, err
		}

		regions = append(regions, placeholderRegion{x: startX, y: startY, width: width, height: height})
	}

	return regions, nil
}

// findAllIdentifierPositions は識別子の全出現位置を返す
func findAllIdentifierPositions(lines []string, idChar rune) [][2]int {
	var positions [][2]int
	for y, line := range lines {
		for x, ch := range line {
			if ch == idChar {
				positions = append(positions, [2]int{x, y})
			}
		}
	}
	return positions
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
	for dy := range height {
		y := startY + dy
		if y >= len(lines) {
			return fmt.Errorf("識別子 '%s': 矩形領域が不完全です (y=%d が範囲外)", id, y)
		}
		for dx := range width {
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

		regions, err := findAllPlaceholderRegionsByID(lines, placement.ID)
		if err != nil {
			return fmt.Errorf("placement %d (%s): %w", idx, placement.Chunks[0], err)
		}

		// 全領域のサイズの完全一致を検証
		for regionIdx, region := range regions {
			if region.width != expectedWidth || region.height != expectedHeight {
				return fmt.Errorf(
					"親チャンク '%s': placement %d (ID='%s', 領域 %d, 子チャンク='%s'): プレースホルダ領域のサイズが不一致: 領域[%d,%d] (位置[%d,%d])、チャンクサイズ%s",
					t.Name, idx, placement.ID, regionIdx, placement.Chunks[0], region.width, region.height, region.x, region.y, chunks[0].Size,
				)
			}
		}
	}

	return nil
}
