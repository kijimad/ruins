package editor

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
)

type tileItem struct {
	Index  int
	Tile   raw.TileRaw
	Active bool
}

type tileEditData struct {
	Index      int
	Tile       raw.TileRaw
	SheetNames []string
}

type tilesData struct {
	Items []tileItem
	Edit  *tileEditData
}

func (s *Server) handleTiles(w http.ResponseWriter, r *http.Request) {
	selected := parseSelectedIndex(r)
	s.renderTiles(w, selected)
}

func (s *Server) renderTiles(w http.ResponseWriter, activeIndex int) {
	tiles := s.store.Tiles()
	rows := make([]tileItem, len(tiles))
	for i, t := range tiles {
		rows[i] = tileItem{Index: i, Tile: t, Active: i == activeIndex}
	}
	data := tilesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(tiles) {
		data.Edit = &tileEditData{Index: activeIndex, Tile: tiles[activeIndex], SheetNames: s.sheetNames()}
	}
	if err := s.templates.ExecuteTemplate(w, "tiles", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findTileIndex(name string) int {
	for i, t := range s.store.Tiles() {
		if t.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleTileUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	tile, err := s.store.Tile(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	tile = parseTileForm(r, tile)
	if err := s.store.UpdateTile(index, tile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/tiles?selected=%d", s.findTileIndex(tile.Name)), http.StatusSeeOther)
}

func (s *Server) handleTileCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}
	tile := raw.TileRaw{Name: name}
	if err := s.store.AddTile(tile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/tiles?selected=%d", s.findTileIndex(name)), http.StatusSeeOther)
}

func (s *Server) handleTileDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteTile(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/tiles", http.StatusSeeOther)
}

func parseTileForm(r *http.Request, t raw.TileRaw) raw.TileRaw {
	t.Name = r.FormValue("name")
	t.Description = r.FormValue("description")
	t.BlockPass = r.FormValue("block_pass") == "on"
	t.BlockView = r.FormValue("block_view") == "on"
	t.SpriteRender.SpriteSheetName = r.FormValue("sprite_sheet_name")
	t.SpriteRender.SpriteKey = r.FormValue("sprite_key")
	shelter, _ := strconv.Atoi(r.FormValue("shelter"))
	t.Shelter = gc.ShelterType(shelter)
	water, _ := strconv.Atoi(r.FormValue("water"))
	t.Water = gc.WaterType(water)
	foliage, _ := strconv.Atoi(r.FormValue("foliage"))
	t.Foliage = gc.FoliageType(foliage)
	return t
}
