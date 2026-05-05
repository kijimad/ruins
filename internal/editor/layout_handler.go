package editor

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
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
	Items []layoutItem
	Edit  *layoutEditData
}

// layoutEditData はチャンク編集テンプレートに渡すデータ
type layoutEditData struct {
	DirName    string
	FileName   string
	ChunkName  string
	FileKey    string
	Chunk      maptemplate.ChunkTemplate
	CheatSheet []cheatSheetEntry
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
	merged := s.mergePalettes(chunk.Palettes)
	if merged != nil {
		ed.CheatSheet = s.buildCheatSheet(merged)
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
	data := layoutsData{Items: items}
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

func (s *Server) handleLayoutEdit(w http.ResponseWriter, r *http.Request) {
	dirName := r.PathValue("dir")
	fileName := r.PathValue("file") + ".toml"
	chunkName := r.PathValue("chunk")

	chunk, err := s.layoutStore.GetChunk(dirName, fileName, chunkName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := s.buildLayoutEditData(dirName, fileName, chunkName, FileKey(dirName, fileName), chunk)
	if err := s.templates.ExecuteTemplate(w, "layout-edit", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleLayoutUpdate(w http.ResponseWriter, r *http.Request) {
	dirName := r.PathValue("dir")
	fileName := r.PathValue("file") + ".toml"
	chunkName := r.PathValue("chunk")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	mapContent := r.FormValue("map_content")

	// パレットに定義されていない文字がないか検証してから保存する
	validate := func(chunk *maptemplate.ChunkTemplate) error {
		merged := s.mergePalettes(chunk.Palettes)
		if merged == nil {
			return nil
		}
		return validateMapContent(mapContent, merged, chunk.Placements)
	}

	if err := s.layoutStore.SaveChunk(dirName, fileName, chunkName, mapContent, validate); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.renderLayoutPartial(w, dirName, fileName, chunkName)
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

func (s *Server) renderLayoutPartial(w http.ResponseWriter, activeDirName, activeFileName, activeChunkName string) {
	items, err := s.buildLayoutItems(activeDirName, activeFileName, activeChunkName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := layoutsData{Items: items}

	if activeChunkName != "" {
		chunk, err := s.layoutStore.GetChunk(activeDirName, activeFileName, activeChunkName)
		if err == nil {
			ed := s.buildLayoutEditData(activeDirName, activeFileName, activeChunkName, FileKey(activeDirName, activeFileName), chunk)
			if err := s.templates.ExecuteTemplate(w, "layout-edit", ed); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		if _, err := w.Write([]byte(`<div class="text-secondary mt-5 text-center">チャンクを選択してください</div>`)); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "layout-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "layout-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
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
	expandedMap := chunk.Map
	if r.Method == http.MethodPost {
		if parseErr := r.ParseForm(); parseErr != nil {
			http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
			return
		}
		if content := r.FormValue("map_content"); content != "" {
			expandedMap = content
		}
	}

	// placementsがある場合はチャンクを展開する
	if len(chunk.Placements) > 0 {
		loader, loaderErr := s.layoutStore.BuildTemplateLoader(s.paletteStore.Dir())
		if loaderErr != nil {
			http.Error(w, "テンプレートローダー構築エラー: "+loaderErr.Error(), http.StatusInternalServerError)
			return
		}
		// 一時的にチャンクのMapを差し替えて展開する
		tempChunk := *chunk
		tempChunk.Map = expandedMap
		expandedMap, err = tempChunk.ExpandWithPlacements(loader, 0)
		if err != nil {
			http.Error(w, "チャンク展開エラー: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	merged := s.mergePalettes(chunk.Palettes)
	data := s.buildPreviewData(expandedMap, merged)
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

// buildCheatSheet はマージ済みパレットからチートシートデータを構築する
func (s *Server) buildCheatSheet(merged *maptemplate.Palette) []cheatSheetEntry {
	sm := s.buildSpriteStyleMaps()
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

// buildPreviewData は展開済みマップとマージ済みパレットからプレビューデータを構築する
func (s *Server) buildPreviewData(expandedMap string, merged *maptemplate.Palette) previewData {
	sm := s.buildSpriteStyleMaps()

	lines := strings.Split(strings.TrimSpace(expandedMap), "\n")
	cols := 0
	for _, line := range lines {
		if len([]rune(line)) > cols {
			cols = len([]rune(line))
		}
	}
	var cells []previewCell
	for _, line := range lines {
		for _, ch := range line {
			charStr := string(ch)
			cell := previewCell{Char: charStr}
			if terrain, ok := merged.GetTerrain(charStr); ok {
				cell.Terrain = terrain
				if style, found := sm.tile[terrain]; found {
					cell.Sprites = append(cell.Sprites, previewSprite{Style: template.CSS(style)})
				}
			}
			if propName, ok := merged.GetProp(charStr); ok {
				cell.Prop = propName
				if style, found := sm.prop[propName]; found {
					cell.Sprites = append(cell.Sprites, previewSprite{Style: template.CSS(style)})
				}
			}
			if npcName, ok := merged.GetNPC(charStr); ok {
				cell.NPC = npcName
				if style, found := sm.npc[npcName]; found {
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
