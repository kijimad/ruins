package editor

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kijimaD/ruins/internal/raw"
)

// ================== コマンドテーブル ==================

type commandTableItem struct {
	Index  int
	Table  raw.CommandTable
	Active bool
}

type commandTableEditData struct {
	Index       int
	Table       raw.CommandTable
	ItemOptions []itemSelectOption
}

type commandTablesData struct {
	Items []commandTableItem
	Edit  *commandTableEditData
}

func (s *Server) handleCommandTables(w http.ResponseWriter, r *http.Request) {
	selected := -1
	if v := r.URL.Query().Get("selected"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			selected = n
		}
	}
	s.renderCommandTables(w, selected)
}

func (s *Server) renderCommandTables(w http.ResponseWriter, activeIndex int) {
	tables := s.store.CommandTables()
	rows := make([]commandTableItem, len(tables))
	for i, t := range tables {
		rows[i] = commandTableItem{Index: i, Table: t, Active: i == activeIndex}
	}
	data := commandTablesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(tables) {
		data.Edit = &commandTableEditData{Index: activeIndex, Table: tables[activeIndex], ItemOptions: s.itemOptions()}
	}
	if err := s.templates.ExecuteTemplate(w, "command-tables", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findCommandTableIndex(name string) int {
	for i, t := range s.store.CommandTables() {
		if t.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleCommandTableUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	ct := parseCommandTableForm(r)
	if err := s.store.UpdateCommandTable(index, ct); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/command-tables?selected=%d", s.findCommandTableIndex(ct.Name)), http.StatusSeeOther)
}

func (s *Server) handleCommandTableCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}
	ct := raw.CommandTable{Name: name}
	if err := s.store.AddCommandTable(ct); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/command-tables?selected=%d", s.findCommandTableIndex(name)), http.StatusSeeOther)
}

func (s *Server) handleCommandTableDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteCommandTable(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/command-tables", http.StatusSeeOther)
}

func parseCommandTableForm(r *http.Request) raw.CommandTable {
	ct := raw.CommandTable{Name: r.FormValue("name")}
	for i := 0; ; i++ {
		weapon := strings.TrimSpace(r.FormValue(fmt.Sprintf("entry_weapon_%d", i)))
		if weapon == "" {
			break
		}
		weight, _ := strconv.ParseFloat(r.FormValue(fmt.Sprintf("entry_weight_%d", i)), 64)
		ct.Entries = append(ct.Entries, raw.CommandTableEntry{Weapon: weapon, Weight: weight})
	}
	return ct
}

// ================== ドロップテーブル ==================

type dropTableItem struct {
	Index  int
	Table  raw.DropTable
	Active bool
}

type dropTableEditData struct {
	Index       int
	Table       raw.DropTable
	ItemOptions []itemSelectOption
}

type dropTablesData struct {
	Items []dropTableItem
	Edit  *dropTableEditData
}

func (s *Server) handleDropTables(w http.ResponseWriter, r *http.Request) {
	selected := -1
	if v := r.URL.Query().Get("selected"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			selected = n
		}
	}
	s.renderDropTables(w, selected)
}

func (s *Server) renderDropTables(w http.ResponseWriter, activeIndex int) {
	tables := s.store.DropTables()
	rows := make([]dropTableItem, len(tables))
	for i, t := range tables {
		rows[i] = dropTableItem{Index: i, Table: t, Active: i == activeIndex}
	}
	data := dropTablesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(tables) {
		data.Edit = &dropTableEditData{Index: activeIndex, Table: tables[activeIndex], ItemOptions: s.itemOptions()}
	}
	if err := s.templates.ExecuteTemplate(w, "drop-tables", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findDropTableIndex(name string) int {
	for i, t := range s.store.DropTables() {
		if t.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleDropTableUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	dt := parseDropTableForm(r)
	if err := s.store.UpdateDropTable(index, dt); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/drop-tables?selected=%d", s.findDropTableIndex(dt.Name)), http.StatusSeeOther)
}

func (s *Server) handleDropTableCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}
	dt := raw.DropTable{Name: name}
	if err := s.store.AddDropTable(dt); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/drop-tables?selected=%d", s.findDropTableIndex(name)), http.StatusSeeOther)
}

func (s *Server) handleDropTableDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteDropTable(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/drop-tables", http.StatusSeeOther)
}

func parseDropTableForm(r *http.Request) raw.DropTable {
	dt := raw.DropTable{Name: r.FormValue("name")}
	for i := 0; ; i++ {
		material := strings.TrimSpace(r.FormValue(fmt.Sprintf("entry_material_%d", i)))
		if material == "" {
			break
		}
		weight, _ := strconv.ParseFloat(r.FormValue(fmt.Sprintf("entry_weight_%d", i)), 64)
		dt.Entries = append(dt.Entries, raw.DropTableEntry{Material: material, Weight: weight})
	}
	return dt
}

// ================== アイテムテーブル ==================

type itemTableItem struct {
	Index  int
	Table  raw.ItemTable
	Active bool
}

type itemTableEditData struct {
	Index       int
	Table       raw.ItemTable
	ItemOptions []itemSelectOption
}

type itemTablesData struct {
	Items []itemTableItem
	Edit  *itemTableEditData
}

func (s *Server) handleItemTables(w http.ResponseWriter, r *http.Request) {
	selected := -1
	if v := r.URL.Query().Get("selected"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			selected = n
		}
	}
	s.renderItemTables(w, selected)
}

func (s *Server) renderItemTables(w http.ResponseWriter, activeIndex int) {
	tables := s.store.ItemTables()
	rows := make([]itemTableItem, len(tables))
	for i, t := range tables {
		rows[i] = itemTableItem{Index: i, Table: t, Active: i == activeIndex}
	}
	data := itemTablesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(tables) {
		data.Edit = &itemTableEditData{Index: activeIndex, Table: tables[activeIndex], ItemOptions: s.itemOptions()}
	}
	if err := s.templates.ExecuteTemplate(w, "item-tables", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findItemTableIndex(name string) int {
	for i, t := range s.store.ItemTables() {
		if t.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleItemTableUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	it := parseItemTableForm(r)
	if err := s.store.UpdateItemTable(index, it); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/item-tables?selected=%d", s.findItemTableIndex(it.Name)), http.StatusSeeOther)
}

func (s *Server) handleItemTableCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}
	it := raw.ItemTable{Name: name}
	if err := s.store.AddItemTable(it); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/item-tables?selected=%d", s.findItemTableIndex(name)), http.StatusSeeOther)
}

func (s *Server) handleItemTableDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteItemTable(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/item-tables", http.StatusSeeOther)
}

func parseItemTableForm(r *http.Request) raw.ItemTable {
	it := raw.ItemTable{Name: r.FormValue("name")}
	for i := 0; ; i++ {
		itemName := strings.TrimSpace(r.FormValue(fmt.Sprintf("entry_item_%d", i)))
		if itemName == "" {
			break
		}
		weight, _ := strconv.ParseFloat(r.FormValue(fmt.Sprintf("entry_weight_%d", i)), 64)
		minDepth, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("entry_min_depth_%d", i)))
		maxDepth, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("entry_max_depth_%d", i)))
		it.Entries = append(it.Entries, raw.ItemTableEntry{ItemName: itemName, Weight: weight, MinDepth: minDepth, MaxDepth: maxDepth})
	}
	return it
}

// ================== 敵テーブル ==================

type enemyTableItem struct {
	Index  int
	Table  raw.EnemyTable
	Active bool
}

type enemyTableEditData struct {
	Index       int
	Table       raw.EnemyTable
	MemberNames []string
}

type enemyTablesData struct {
	Items []enemyTableItem
	Edit  *enemyTableEditData
}

func (s *Server) handleEnemyTables(w http.ResponseWriter, r *http.Request) {
	selected := -1
	if v := r.URL.Query().Get("selected"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			selected = n
		}
	}
	s.renderEnemyTables(w, selected)
}

func (s *Server) renderEnemyTables(w http.ResponseWriter, activeIndex int) {
	tables := s.store.EnemyTables()
	rows := make([]enemyTableItem, len(tables))
	for i, t := range tables {
		rows[i] = enemyTableItem{Index: i, Table: t, Active: i == activeIndex}
	}
	data := enemyTablesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(tables) {
		data.Edit = &enemyTableEditData{Index: activeIndex, Table: tables[activeIndex], MemberNames: s.memberNames()}
	}
	if err := s.templates.ExecuteTemplate(w, "enemy-tables", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findEnemyTableIndex(name string) int {
	for i, t := range s.store.EnemyTables() {
		if t.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleEnemyTableUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	et := parseEnemyTableForm(r)
	if err := s.store.UpdateEnemyTable(index, et); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/enemy-tables?selected=%d", s.findEnemyTableIndex(et.Name)), http.StatusSeeOther)
}

func (s *Server) handleEnemyTableCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}
	et := raw.EnemyTable{Name: name}
	if err := s.store.AddEnemyTable(et); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/enemy-tables?selected=%d", s.findEnemyTableIndex(name)), http.StatusSeeOther)
}

func (s *Server) handleEnemyTableDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteEnemyTable(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/enemy-tables", http.StatusSeeOther)
}

func parseEnemyTableForm(r *http.Request) raw.EnemyTable {
	et := raw.EnemyTable{Name: r.FormValue("name")}
	for i := 0; ; i++ {
		enemyName := strings.TrimSpace(r.FormValue(fmt.Sprintf("entry_enemy_%d", i)))
		if enemyName == "" {
			break
		}
		weight, _ := strconv.ParseFloat(r.FormValue(fmt.Sprintf("entry_weight_%d", i)), 64)
		minDepth, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("entry_min_depth_%d", i)))
		maxDepth, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("entry_max_depth_%d", i)))
		et.Entries = append(et.Entries, raw.EnemyTableEntry{EnemyName: enemyName, Weight: weight, MinDepth: minDepth, MaxDepth: maxDepth})
	}
	return et
}
