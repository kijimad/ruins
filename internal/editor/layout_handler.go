package editor

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/kijimaD/ruins/internal/maptemplate"
)

// layoutItem はサイドバーに表示する1チャンク分のデータ
type layoutItem struct {
	DirName   string
	FileName  string
	ChunkName string
	FileKey   string
	Active    bool
}

// layoutsData はレイアウト一覧テンプレートに渡すデータ
type layoutsData struct {
	Items    []layoutItem
	Edit     *layoutEditData
	DirNames []string
}

// layoutEditData はチャンク編集テンプレートに渡すデータ
type layoutEditData struct {
	DirName           string
	FileName          string
	ChunkName         string
	FileKey           string
	Chunk             maptemplate.ChunkTemplate
	CheatSheet        []cheatSheetEntry
	AvailablePalettes []string
	AvailableChunks   []string
	Preview           *previewData
}

// cheatSheetEntry はチートシートの1行分のデータ
type cheatSheetEntry struct {
	Char     string
	Category string
	Name     string
	Style    template.CSS
}

// previewData はプレビューテンプレートに渡すデータ
type previewData struct {
	Cols  int
	Cells []previewCell
}

// previewCell はプレビューグリッドの1セルデータ
type previewCell struct {
	Char    string
	Terrain string
	Prop    string
	NPC     string
	Sprites []previewSprite
}

// previewSprite は1セルに表示するスプライト1枚分のCSS情報
type previewSprite struct {
	Style template.CSS
}

// buildLayoutEditData はチャンクからパレットをマージし、編集データを構築する
func (s *Server) buildLayoutEditData(dirName, fileName, chunkName, fileKey string, chunk *maptemplate.ChunkTemplate) *layoutEditData {
	ed := &layoutEditData{
		DirName:   dirName,
		FileName:  fileName,
		ChunkName: chunkName,
		FileKey:   fileKey,
		Chunk:     *chunk,
	}
	if s.paletteStore != nil {
		palettes, _ := s.paletteStore.List()
		for _, p := range palettes {
			ed.AvailablePalettes = append(ed.AvailablePalettes, p.ID)
		}
	}
	ed.AvailableChunks = s.layoutStore.ChunkNames()
	merged := s.mergePalettes(chunk.Palettes)
	if merged != nil {
		sm := s.buildSpriteStyleMaps()
		ed.CheatSheet = s.buildCheatSheetWith(merged, sm)
		// プレビューデータを構築する
		var cells [][]maptemplate.MapCell
		if len(chunk.Placements) > 0 && s.layoutStore != nil && s.paletteStore != nil {
			loader, err := s.layoutStore.BuildTemplateLoader(s.paletteStore.Dir())
			if err != nil {
				log.Printf("テンプレートローダー構築エラー: %v", err)
			} else {
				resolved, err := chunk.ExpandWithPlacements(loader, 0)
				if err != nil {
					log.Printf("チャンク展開エラー (%s): %v", chunk.Name, err)
				} else {
					cells = resolved
				}
			}
		}
		if cells == nil {
			cells = maptemplate.ResolveMapCells(chunk.Map, merged)
		}
		preview := s.buildPreviewDataFromCellsWith(cells, sm)
		ed.Preview = &preview
	}
	return ed
}

// mergePalettes はパレット名リストからマージ済みパレットを返す
func (s *Server) mergePalettes(paletteNames []string) *maptemplate.Palette {
	if s.paletteStore == nil {
		return nil
	}
	palettes := make([]*maptemplate.Palette, 0, len(paletteNames))
	for _, name := range paletteNames {
		p, err := s.paletteStore.Get(name)
		if err != nil {
			continue
		}
		palettes = append(palettes, p)
	}
	if len(palettes) == 0 {
		return nil
	}
	return maptemplate.MergePalettes(palettes...)
}

func (s *Server) handleLayouts(w http.ResponseWriter, _ *http.Request) {
	s.renderLayouts(w, "", "", "")
}

func (s *Server) renderLayouts(w http.ResponseWriter, activeDirName, activeFileName, activeChunkName string) {
	items, err := s.buildLayoutItems(activeDirName, activeFileName, activeChunkName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := layoutsData{Items: items, DirNames: s.layoutStore.DirNames()}
	if activeChunkName != "" {
		chunk, err := s.layoutStore.GetChunk(activeDirName, activeFileName, activeChunkName)
		if err == nil {
			data.Edit = s.buildLayoutEditData(activeDirName, activeFileName, activeChunkName, FileKey(activeDirName, activeFileName), chunk)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "layouts", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) buildLayoutItems(activeDirName, activeFileName, activeChunkName string) ([]layoutItem, error) {
	entries, err := s.layoutStore.List()
	if err != nil {
		return nil, err
	}
	var items []layoutItem
	for _, e := range entries {
		for _, c := range e.Chunks {
			active := e.Dir == activeDirName && e.FileName == activeFileName && c.Name == activeChunkName
			items = append(items, layoutItem{
				DirName:   e.Dir,
				FileName:  e.FileName,
				ChunkName: c.Name,
				FileKey:   FileKey(e.Dir, e.FileName),
				Active:    active,
			})
		}
	}
	return items, nil
}

func (s *Server) handleLayoutCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}

	dirName := strings.TrimSpace(r.FormValue("dir"))
	fileName := strings.TrimSpace(r.FormValue("file"))
	chunkName := strings.TrimSpace(r.FormValue("name"))

	if dirName == "" || fileName == "" || chunkName == "" {
		http.Error(w, "ディレクトリ、ファイル名、チャンク名は必須です", http.StatusBadRequest)
		return
	}

	if !strings.HasSuffix(fileName, ".toml") {
		fileName += ".toml"
	}

	// チャンク名からサイズをパースして初期マップを生成する
	chunk := maptemplate.ChunkTemplate{
		Name:   chunkName,
		Weight: 100,
	}
	parts := strings.SplitN(chunkName, "_", 2)
	if len(parts) < 2 {
		http.Error(w, "チャンク名は 'WxH_名前' 形式で指定してください", http.StatusBadRequest)
		return
	}
	dims := strings.Split(parts[0], "x")
	if len(dims) != 2 {
		http.Error(w, "チャンク名は 'WxH_名前' 形式で指定してください", http.StatusBadRequest)
		return
	}
	width, wErr := strconv.Atoi(dims[0])
	height, hErr := strconv.Atoi(dims[1])
	if wErr != nil || hErr != nil || width <= 0 || height <= 0 {
		http.Error(w, "幅と高さは正の整数で指定してください", http.StatusBadRequest)
		return
	}

	// 指定サイズの '.' で埋めたマップを生成する
	row := strings.Repeat(".", width)
	rows := make([]string, height)
	for i := range rows {
		rows[i] = row
	}
	chunk.Map = strings.Join(rows, "\n") + "\n"
	chunk.Size = maptemplate.Size{W: width, H: height}

	if err := s.layoutStore.AddChunk(dirName, fileName, chunk); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	redirectURL := fmt.Sprintf("/layouts/%s/%s/%s/edit",
		dirName, strings.TrimSuffix(fileName, ".toml"), chunkName)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (s *Server) handleLayoutEdit(w http.ResponseWriter, r *http.Request) {
	dirName := r.PathValue("dir")
	fileName := r.PathValue("file") + ".toml"
	chunkName := r.PathValue("chunk")
	s.renderLayouts(w, dirName, fileName, chunkName)
}

func (s *Server) handleLayoutUpdate(w http.ResponseWriter, r *http.Request) {
	dirName := r.PathValue("dir")
	fileName := r.PathValue("file") + ".toml"
	chunkName := r.PathValue("chunk")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	mapContent := strings.ReplaceAll(r.FormValue("map_content"), "\r\n", "\n")
	palettes := r.Form["palettes"]
	placements := parsePlacements(r)

	// パレットに定義されていない文字がないか検証してから保存する
	validate := func(chunk *maptemplate.ChunkTemplate) error {
		// パレットを更新してからバリデーションする
		if palettes != nil {
			chunk.Palettes = palettes
		}
		merged := s.mergePalettes(chunk.Palettes)
		if merged == nil {
			return nil
		}
		return validateMapContent(mapContent, merged, placements)
	}

	update := SaveChunkUpdate{
		MapContent: mapContent,
		Palettes:   palettes,
		Placements: placements,
	}
	if err := s.layoutStore.SaveChunk(dirName, fileName, chunkName, update, validate); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 保存後は編集ページにリダイレクトする
	redirectURL := fmt.Sprintf("/layouts/%s/%s/%s/edit",
		dirName, strings.TrimSuffix(fileName, ".toml"), chunkName)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// parsePlacements はフォームからplacements情報をパースする。
// placement_id_0, placement_chunks_0[] の形式で送信される
func parsePlacements(r *http.Request) []maptemplate.ChunkPlacement {
	var placements []maptemplate.ChunkPlacement
	for i := 0; ; i++ {
		id := strings.TrimSpace(r.FormValue(fmt.Sprintf("placement_id_%d", i)))
		if id == "" {
			break
		}
		chunks := r.Form[fmt.Sprintf("placement_chunks_%d[]", i)]
		if len(chunks) == 0 {
			continue
		}
		placements = append(placements, maptemplate.ChunkPlacement{
			ID:     id,
			Chunks: chunks,
		})
	}
	return placements
}

// validateMapContent はマップ内の全文字がパレットに定義されているか検証する。
// placementsがある場合、プレースホルダ文字(@)と配置IDはスキップする
func validateMapContent(mapContent string, palette *maptemplate.Palette, placements []maptemplate.ChunkPlacement) error {
	// プレースホルダとして許可する文字を収集する
	skip := make(map[rune]bool)
	if len(placements) > 0 {
		skip['@'] = true
		for _, p := range placements {
			if len(p.ID) == 1 {
				skip[rune(p.ID[0])] = true
			}
		}
	}

	undefined := make(map[rune]bool)
	for _, ch := range mapContent {
		if ch == '\n' || ch == '\r' || ch == ' ' {
			continue
		}
		if skip[ch] {
			continue
		}
		if _, ok := palette.GetTerrain(string(ch)); !ok {
			undefined[ch] = true
		}
	}
	if len(undefined) == 0 {
		return nil
	}

	chars := make([]string, 0, len(undefined))
	for ch := range undefined {
		chars = append(chars, fmt.Sprintf("%c", ch))
	}
	sort.Strings(chars)
	return fmt.Errorf("パレットに未定義の文字があります: %s", strings.Join(chars, ", "))
}

// handleLayoutPreview はチャンクの展開済みマップをパレットで解決し、スプライトグリッドHTMLを返す。
// GETの場合は保存済みマップを使い、POSTの場合はフォームのmap_contentを使う
func (s *Server) handleLayoutPreview(w http.ResponseWriter, r *http.Request) {
	dirName := r.PathValue("dir")
	fileName := r.PathValue("file") + ".toml"
	chunkName := r.PathValue("chunk")

	chunk, err := s.layoutStore.GetChunk(dirName, fileName, chunkName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// POSTの場合はフォームの内容を使う
	mapStr := chunk.Map
	if r.Method == http.MethodPost {
		if parseErr := r.ParseForm(); parseErr != nil {
			http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
			return
		}
		if content := r.FormValue("map_content"); content != "" {
			mapStr = content
		}
	}

	merged := s.mergePalettes(chunk.Palettes)

	// placementsがある場合はチャンクを展開する
	var cells [][]maptemplate.MapCell
	if len(chunk.Placements) > 0 {
		loader, loaderErr := s.layoutStore.BuildTemplateLoader(s.paletteStore.Dir())
		if loaderErr != nil {
			http.Error(w, "テンプレートローダー構築エラー: "+loaderErr.Error(), http.StatusInternalServerError)
			return
		}
		tempChunk := *chunk
		tempChunk.Map = mapStr
		cells, err = tempChunk.ExpandWithPlacements(loader, 0)
		if err != nil {
			http.Error(w, "チャンク展開エラー: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		cells = maptemplate.ResolveMapCells(mapStr, merged)
	}

	data := s.buildPreviewDataFromCells(cells)
	if err := s.templates.ExecuteTemplate(w, "layout-preview", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// spriteStyleMaps はタイル・置物・NPCのスプライトスタイルのルックアップマップ
type spriteStyleMaps struct {
	tile map[string]string
	prop map[string]string
	npc  map[string]string
}

// buildSpriteStyleMaps はraw定義からスプライトスタイルのルックアップマップを構築する
func (s *Server) buildSpriteStyleMaps() spriteStyleMaps {
	m := spriteStyleMaps{
		tile: make(map[string]string),
		prop: make(map[string]string),
		npc:  make(map[string]string),
	}
	for _, t := range s.store.Tiles() {
		if style := s.resolveSpriteStyle(t.SpriteRender.SpriteSheetName, t.SpriteRender.SpriteKey); style != "" {
			m.tile[t.Name] = style
		}
	}
	for _, p := range s.store.Props() {
		if style := s.resolveSpriteStyle(p.SpriteRender.SpriteSheetName, p.SpriteRender.SpriteKey); style != "" {
			m.prop[p.Name] = style
		}
	}
	for _, mem := range s.store.Members() {
		if style := s.resolveSpriteStyle(mem.SpriteSheetName, mem.SpriteKey); style != "" {
			m.npc[mem.Name] = style
		}
	}
	return m
}

// buildCheatSheetWith はマージ済みパレットとスプライトマップからチートシートデータを構築する
func (s *Server) buildCheatSheetWith(merged *maptemplate.Palette, sm spriteStyleMaps) []cheatSheetEntry {
	entries := make([]cheatSheetEntry, 0, len(merged.Terrain)+len(merged.Props)+len(merged.NPCs))

	for _, cm := range sortedMappings(merged.Terrain) {
		entries = append(entries, cheatSheetEntry{
			Char:     cm.Char,
			Category: "地形",
			Name:     cm.Value,
			Style:    template.CSS(sm.tile[cm.Value]),
		})
	}
	for _, cm := range sortedEntryMappings(merged.Props) {
		entries = append(entries, cheatSheetEntry{
			Char:     cm.Char,
			Category: "置物",
			Name:     cm.Value,
			Style:    template.CSS(sm.prop[cm.Value]),
		})
	}
	for _, cm := range sortedEntryMappings(merged.NPCs) {
		entries = append(entries, cheatSheetEntry{
			Char:     cm.Char,
			Category: "NPC",
			Name:     cm.Value,
			Style:    template.CSS(sm.npc[cm.Value]),
		})
	}
	return entries
}

// sortedEntryMappings はPaletteEntryマップをソート済みcharMappingに変換する
func sortedEntryMappings(m map[string]maptemplate.PaletteEntry) []charMapping {
	entries := make([]charMapping, 0, len(m))
	for k, v := range m {
		entries = append(entries, charMapping{Char: k, Value: v.ID, Tile: v.Tile})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Char < entries[j].Char
	})
	return entries
}

// buildPreviewDataFromCells は解決済みセル配列からプレビューデータを構築する
func (s *Server) buildPreviewDataFromCells(mapCells [][]maptemplate.MapCell) previewData {
	return s.buildPreviewDataFromCellsWith(mapCells, s.buildSpriteStyleMaps())
}

// buildPreviewDataFromCellsWith はスプライトマップを受け取ってプレビューデータを構築する
func (s *Server) buildPreviewDataFromCellsWith(mapCells [][]maptemplate.MapCell, sm spriteStyleMaps) previewData {
	cols := 0
	for _, row := range mapCells {
		if len(row) > cols {
			cols = len(row)
		}
	}
	var cells []previewCell
	for _, row := range mapCells {
		for _, mc := range row {
			cell := previewCell{
				Terrain: mc.Terrain,
				Prop:    mc.Prop,
				NPC:     mc.NPC,
			}
			if style, found := sm.tile[mc.Terrain]; found {
				cell.Sprites = append(cell.Sprites, previewSprite{Style: template.CSS(style)})
			}
			if mc.Prop != "" {
				if style, found := sm.prop[mc.Prop]; found {
					cell.Sprites = append(cell.Sprites, previewSprite{Style: template.CSS(style)})
				}
			}
			if mc.NPC != "" {
				if style, found := sm.npc[mc.NPC]; found {
					cell.Sprites = append(cell.Sprites, previewSprite{Style: template.CSS(style)})
				}
			}
			cells = append(cells, cell)
		}
	}

	return previewData{Cols: cols, Cells: cells}
}

// resolveSpriteStyle はスプライトシート名とキーからCSS文字列を返す
func (s *Server) resolveSpriteStyle(sheetName, spriteKey string) string {
	frames, ok := s.sprites[sheetName]
	if !ok {
		return ""
	}
	f, ok := frames[spriteKey]
	if !ok {
		f, ok = frames[spriteKey+"_0"]
		if !ok {
			return ""
		}
	}
	size, ok := s.sheetSizes[sheetName]
	if !ok {
		return ""
	}
	return fmt.Sprintf(
		"background:url('/sprites/%s') -%dpx -%dpx;background-size:%dpx %dpx;",
		sheetName, f.X, f.Y, size.W, size.H,
	)
}
