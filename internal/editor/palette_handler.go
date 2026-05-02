package editor

import (
	"fmt"
	"log"
	"net/http"
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

func (s *Server) handlePalettes(w http.ResponseWriter, _ *http.Request) {
	s.renderPalettes(w, "")
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
		PropEntries:    sortedMappings(p.Props),
		NPCEntries:     sortedMappings(p.NPCs),
		TileNames:      tileNames,
		PropNames:      propNames,
		NPCNames:       npcNames,
	}
}

func (s *Server) handlePaletteEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.paletteStore.Get(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := s.buildPaletteEditData(*p)
	if err := s.templates.ExecuteTemplate(w, "palette-edit", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	s.renderPalettePartial(w, id)
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
		Props:       map[string]string{},
		NPCs:        map[string]string{},
	}
	if err := s.paletteStore.Save(&p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderPalettePartial(w, id)
}

func (s *Server) handlePaletteDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.paletteStore.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderPalettePartial(w, "")
}

func (s *Server) renderPalettePartial(w http.ResponseWriter, activeID string) {
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
				ed := s.buildPaletteEditData(p)
				if err := s.templates.ExecuteTemplate(w, "palette-edit", ed); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				break
			}
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">パレットを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "pal-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "pal-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
}

// parsePaletteForm はフォームデータからPaletteを構築する
// terrain_char_0, terrain_value_0, ... の形式で送信される
func parsePaletteForm(r *http.Request, id string) maptemplate.Palette {
	desc := strings.TrimSpace(r.FormValue("description"))
	if desc == "" {
		desc = id
	}

	terrain := make(map[string]string)
	props := make(map[string]string)
	npcs := make(map[string]string)

	for i := 0; ; i++ {
		char := strings.TrimSpace(r.FormValue(fmt.Sprintf("terrain_char_%d", i)))
		value := strings.TrimSpace(r.FormValue(fmt.Sprintf("terrain_value_%d", i)))
		if char == "" && value == "" {
			break
		}
		if char != "" && value != "" {
			terrain[char] = value
		}
	}

	for i := 0; ; i++ {
		char := strings.TrimSpace(r.FormValue(fmt.Sprintf("prop_char_%d", i)))
		value := strings.TrimSpace(r.FormValue(fmt.Sprintf("prop_value_%d", i)))
		if char == "" && value == "" {
			break
		}
		if char != "" && value != "" {
			props[char] = value
		}
	}

	for i := 0; ; i++ {
		char := strings.TrimSpace(r.FormValue(fmt.Sprintf("npc_char_%d", i)))
		value := strings.TrimSpace(r.FormValue(fmt.Sprintf("npc_value_%d", i)))
		if char == "" && value == "" {
			break
		}
		if char != "" && value != "" {
			npcs[char] = value
		}
	}

	return maptemplate.Palette{
		ID:          id,
		Description: desc,
		Terrain:     terrain,
		Props:       props,
		NPCs:        npcs,
	}
}
