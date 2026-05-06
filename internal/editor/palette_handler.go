package editor

import (
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/kijimaD/ruins/internal/maptemplate"
)

type paletteItem struct {
	Palette maptemplate.Palette
	Active  bool
}

type palettesData struct {
	Items []paletteItem
	Edit  *paletteEditData
}

type charMapping struct {
	Char  string
	Value string
	Tile  string
}

type paletteEditData struct {
	Palette        maptemplate.Palette
	TerrainEntries []charMapping
	PropEntries    []charMapping
	NPCEntries     []charMapping
	TileNames      []string
	PropNames      []string
	NPCNames       []string
}

func sortedMappings(m map[string]string) []charMapping {
	entries := make([]charMapping, 0, len(m))
	for k, v := range m {
		entries = append(entries, charMapping{Char: k, Value: v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Char < entries[j].Char
	})
	return entries
}

func (s *Server) handlePalettes(w http.ResponseWriter, r *http.Request) {
	selected := r.URL.Query().Get("selected")
	s.renderPalettes(w, selected)
}

func (s *Server) renderPalettes(w http.ResponseWriter, activeID string) {
	palettes, err := s.paletteStore.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows := make([]paletteItem, len(palettes))
	for i, p := range palettes {
		rows[i] = paletteItem{Palette: p, Active: p.ID == activeID}
	}
	data := palettesData{Items: rows}
	if activeID != "" {
		for _, p := range palettes {
			if p.ID == activeID {
				data.Edit = s.buildPaletteEditData(p)
				break
			}
		}
	}
	if err := s.templates.ExecuteTemplate(w, "palettes", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) buildPaletteEditData(p maptemplate.Palette) *paletteEditData {
	tiles := s.store.Tiles()
	tileNames := make([]string, len(tiles))
	for i, t := range tiles {
		tileNames[i] = t.Name
	}
	props := s.store.Props()
	propNames := make([]string, len(props))
	for i, pr := range props {
		propNames[i] = pr.Name
	}
	members := s.store.Members()
	npcNames := make([]string, len(members))
	for i, m := range members {
		npcNames[i] = m.Name
	}
	sort.Strings(tileNames)
	sort.Strings(propNames)
	sort.Strings(npcNames)
	return &paletteEditData{
		Palette:        p,
		TerrainEntries: sortedMappings(p.Terrain),
		PropEntries:    sortedEntryMappings(p.Props),
		NPCEntries:     sortedEntryMappings(p.NPCs),
		TileNames:      tileNames,
		PropNames:      propNames,
		NPCNames:       npcNames,
	}
}

func (s *Server) handlePaletteUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}

	p := parsePaletteForm(r, id)
	if err := s.paletteStore.Save(&p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/palettes?selected="+url.QueryEscape(id), http.StatusSeeOther)
}

func (s *Server) handlePaletteCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	if id == "" {
		http.Error(w, "IDは必須です", http.StatusBadRequest)
		return
	}
	p := maptemplate.Palette{
		ID:          id,
		Description: id,
		Terrain:     map[string]string{},
		Props:       map[string]maptemplate.PaletteEntry{},
		NPCs:        map[string]maptemplate.PaletteEntry{},
	}
	if err := s.paletteStore.Save(&p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/palettes?selected="+url.QueryEscape(id), http.StatusSeeOther)
}

func (s *Server) handlePaletteDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.paletteStore.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/palettes", http.StatusSeeOther)
}

// parsePaletteForm はフォームデータからPaletteを構築する。
// 同名の配列フィールド（terrain_char[], terrain_value[] など）で送信される
func parsePaletteForm(r *http.Request, id string) maptemplate.Palette {
	desc := strings.TrimSpace(r.FormValue("description"))
	if desc == "" {
		desc = id
	}

	terrain := make(map[string]string)
	tChars := r.Form["terrain_char[]"]
	tValues := r.Form["terrain_value[]"]
	for i := range tChars {
		char := strings.TrimSpace(tChars[i])
		value := ""
		if i < len(tValues) {
			value = strings.TrimSpace(tValues[i])
		}
		if char != "" && value != "" {
			terrain[char] = value
		}
	}

	props := parseEntryFields(r.Form, "prop")
	npcs := parseEntryFields(r.Form, "npc")

	return maptemplate.Palette{
		ID:          id,
		Description: desc,
		Terrain:     terrain,
		Props:       props,
		NPCs:        npcs,
	}
}

// parseEntryFields はフォームの配列フィールドからPaletteEntryマップを構築する
func parseEntryFields(form url.Values, prefix string) map[string]maptemplate.PaletteEntry {
	chars := form[prefix+"_char[]"]
	values := form[prefix+"_value[]"]
	tiles := form[prefix+"_tile[]"]
	result := make(map[string]maptemplate.PaletteEntry)
	for i := range chars {
		char := strings.TrimSpace(chars[i])
		value := ""
		if i < len(values) {
			value = strings.TrimSpace(values[i])
		}
		tile := ""
		if i < len(tiles) {
			tile = strings.TrimSpace(tiles[i])
		}
		if char != "" && value != "" {
			result[char] = maptemplate.PaletteEntry{ID: value, Tile: tile}
		}
	}
	return result
}
