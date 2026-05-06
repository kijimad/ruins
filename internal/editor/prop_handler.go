package editor

import (
	"fmt"
	"image/color"
	"net/http"
	"strconv"
	"strings"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
)

type propItem struct {
	Index  int
	Prop   raw.PropRaw
	Active bool
}

type propEditData struct {
	Index      int
	Prop       raw.PropRaw
	SheetNames []string
}

type propsData struct {
	Items []propItem
	Edit  *propEditData
}

func (s *Server) handleProps(w http.ResponseWriter, r *http.Request) {
	selected := -1
	if v := r.URL.Query().Get("selected"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			selected = n
		}
	}
	s.renderProps(w, selected)
}

func (s *Server) renderProps(w http.ResponseWriter, activeIndex int) {
	props := s.store.Props()
	rows := make([]propItem, len(props))
	for i, p := range props {
		rows[i] = propItem{Index: i, Prop: p, Active: i == activeIndex}
	}
	data := propsData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(props) {
		data.Edit = &propEditData{Index: activeIndex, Prop: props[activeIndex], SheetNames: s.sheetNames()}
	}
	if err := s.templates.ExecuteTemplate(w, "props", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findPropIndex(name string) int {
	for i, p := range s.store.Props() {
		if p.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handlePropUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	prop, err := s.store.Prop(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	prop = parsePropForm(r, prop)
	if err := s.store.UpdateProp(index, prop); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/props?selected=%d", s.findPropIndex(prop.Name)), http.StatusSeeOther)
}

func (s *Server) handlePropCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}
	prop := raw.PropRaw{Name: name}
	if err := s.store.AddProp(prop); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/props?selected=%d", s.findPropIndex(name)), http.StatusSeeOther)
}

func (s *Server) handlePropDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteProp(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/props", http.StatusSeeOther)
}

func parsePropForm(r *http.Request, p raw.PropRaw) raw.PropRaw {
	p.Name = r.FormValue("name")
	p.Description = r.FormValue("description")
	p.BlockPass = r.FormValue("block_pass") == "on"
	p.BlockView = r.FormValue("block_view") == "on"
	p.SpriteRender.SpriteSheetName = r.FormValue("sprite_sheet_name")
	p.SpriteRender.SpriteKey = r.FormValue("sprite_key")

	animKeysStr := strings.TrimSpace(r.FormValue("anim_keys"))
	if animKeysStr != "" {
		keys := strings.Split(animKeysStr, ",")
		p.AnimKeys = make([]string, 0, len(keys))
		for _, k := range keys {
			k = strings.TrimSpace(k)
			if k != "" {
				p.AnimKeys = append(p.AnimKeys, k)
			}
		}
	} else {
		p.AnimKeys = nil
	}

	if r.FormValue("has_light") == "on" {
		if p.LightSource == nil {
			p.LightSource = &gc.LightSource{}
		}
		radius, _ := strconv.Atoi(r.FormValue("light_radius"))
		p.LightSource.Radius = consts.Tile(radius)
		p.LightSource.Enabled = r.FormValue("light_enabled") == "on"
		cr, _ := strconv.Atoi(r.FormValue("light_r"))
		cg, _ := strconv.Atoi(r.FormValue("light_g"))
		cb, _ := strconv.Atoi(r.FormValue("light_b"))
		ca, _ := strconv.Atoi(r.FormValue("light_a"))
		p.LightSource.Color = color.RGBA{R: clampUint8(cr), G: clampUint8(cg), B: clampUint8(cb), A: clampUint8(ca)}
	} else {
		p.LightSource = nil
	}

	if r.FormValue("has_door") == "on" {
		p.Door = &raw.DoorRaw{}
	} else {
		p.Door = nil
	}
	if r.FormValue("has_door_lock_trigger") == "on" {
		p.DoorLockTrigger = &raw.DoorLockTriggerRaw{}
	} else {
		p.DoorLockTrigger = nil
	}
	if r.FormValue("has_warp_next") == "on" {
		p.WarpNextTrigger = &raw.WarpNextTriggerRaw{}
	} else {
		p.WarpNextTrigger = nil
	}
	if r.FormValue("has_warp_escape") == "on" {
		p.WarpEscapeTrigger = &raw.WarpEscapeTriggerRaw{}
	} else {
		p.WarpEscapeTrigger = nil
	}
	if r.FormValue("has_dungeon_gate") == "on" {
		p.DungeonGateTrigger = &raw.DungeonGateTriggerRaw{}
	} else {
		p.DungeonGateTrigger = nil
	}

	return p
}
