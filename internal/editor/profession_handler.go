package editor

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/kijimaD/ruins/internal/raw"
)

type professionItem struct {
	Index      int
	Profession raw.Profession
	Active     bool
}

type professionEditData struct {
	Index       int
	Profession  raw.Profession
	ItemOptions []itemSelectOption
}

type professionsData struct {
	Items []professionItem
	Edit  *professionEditData
}

func (s *Server) handleProfessions(w http.ResponseWriter, _ *http.Request) {
	s.renderProfessions(w, -1)
}

func (s *Server) renderProfessions(w http.ResponseWriter, activeIndex int) {
	profs := s.store.Professions()
	rows := make([]professionItem, len(profs))
	for i, p := range profs {
		rows[i] = professionItem{Index: i, Profession: p, Active: i == activeIndex}
	}
	data := professionsData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(profs) {
		data.Edit = &professionEditData{Index: activeIndex, Profession: profs[activeIndex], ItemOptions: s.itemOptions()}
	}
	if err := s.templates.ExecuteTemplate(w, "professions", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderProfessionPartial(w http.ResponseWriter, activeIndex int) {
	profs := s.store.Professions()
	rows := make([]professionItem, len(profs))
	for i, p := range profs {
		rows[i] = professionItem{Index: i, Profession: p, Active: i == activeIndex}
	}
	data := professionsData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(profs) {
		ed := professionEditData{Index: activeIndex, Profession: profs[activeIndex], ItemOptions: s.itemOptions()}
		if err := s.templates.ExecuteTemplate(w, "profession-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">職業を選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "prof-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "prof-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleProfessionEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	prof, err := s.store.Profession(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := professionEditData{Index: index, Profession: prof, ItemOptions: s.itemOptions()}
	if err := s.templates.ExecuteTemplate(w, "profession-edit", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findProfessionIndex(id string) int {
	for i, p := range s.store.Professions() {
		if p.ID == id {
			return i
		}
	}
	return -1
}

func (s *Server) handleProfessionUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	prof := parseProfessionForm(r)
	if err := s.store.UpdateProfession(index, prof); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderProfessionPartial(w, s.findProfessionIndex(prof.ID))
}

func (s *Server) handleProfessionCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	if id == "" {
		http.Error(w, "IDは必須です", http.StatusBadRequest)
		return
	}
	prof := raw.Profession{ID: id, Name: r.FormValue("name")}
	if err := s.store.AddProfession(prof); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderProfessionPartial(w, s.findProfessionIndex(id))
}

func (s *Server) handleProfessionDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteProfession(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderProfessionPartial(w, -1)
}

func parseProfessionForm(r *http.Request) raw.Profession {
	prof := raw.Profession{
		ID:          r.FormValue("id"),
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}
	prof.Abilities.Vitality, _ = strconv.Atoi(r.FormValue("vitality"))
	prof.Abilities.Strength, _ = strconv.Atoi(r.FormValue("strength"))
	prof.Abilities.Sensation, _ = strconv.Atoi(r.FormValue("sensation"))
	prof.Abilities.Dexterity, _ = strconv.Atoi(r.FormValue("dexterity"))
	prof.Abilities.Agility, _ = strconv.Atoi(r.FormValue("agility"))
	prof.Abilities.Defense, _ = strconv.Atoi(r.FormValue("defense"))

	for i := 0; ; i++ {
		skillID := strings.TrimSpace(r.FormValue(fmt.Sprintf("skill_id_%d", i)))
		if skillID == "" {
			break
		}
		value, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("skill_value_%d", i)))
		prof.Skills = append(prof.Skills, raw.ProfessionSkill{ID: skillID, Value: value})
	}

	for i := 0; ; i++ {
		itemName := strings.TrimSpace(r.FormValue(fmt.Sprintf("prof_item_name_%d", i)))
		if itemName == "" {
			break
		}
		count, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("prof_item_count_%d", i)))
		if count <= 0 {
			count = 1
		}
		prof.Items = append(prof.Items, raw.ProfessionItem{Name: itemName, Count: count})
	}

	for i := 0; ; i++ {
		equipName := strings.TrimSpace(r.FormValue(fmt.Sprintf("equip_name_%d", i)))
		if equipName == "" {
			break
		}
		slot := r.FormValue(fmt.Sprintf("equip_slot_%d", i))
		prof.Equips = append(prof.Equips, raw.ProfessionEquip{Name: equipName, Slot: slot})
	}

	return prof
}
