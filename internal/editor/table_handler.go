package editor

import (
	"fmt"
	"log"
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

func (s *Server) handleCommandTables(w http.ResponseWriter, _ *http.Request) {
	s.renderCommandTables(w, -1)
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

func (s *Server) renderCommandTablePartial(w http.ResponseWriter, activeIndex int) {
	tables := s.store.CommandTables()
	rows := make([]commandTableItem, len(tables))
	for i, t := range tables {
		rows[i] = commandTableItem{Index: i, Table: t, Active: i == activeIndex}
	}
	data := commandTablesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(tables) {
		ed := commandTableEditData{Index: activeIndex, Table: tables[activeIndex], ItemOptions: s.itemOptions()}
		if err := s.templates.ExecuteTemplate(w, "command-table-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">コマンドテーブルを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "ct-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "ct-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleCommandTableEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	ct, err := s.store.CommandTable(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := commandTableEditData{Index: index, Table: ct, ItemOptions: s.itemOptions()}
	if err := s.templates.ExecuteTemplate(w, "command-table-edit", data); err != nil {
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
	s.renderCommandTablePartial(w, s.findCommandTableIndex(ct.Name))
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
	s.renderCommandTablePartial(w, s.findCommandTableIndex(name))
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
	s.renderCommandTablePartial(w, -1)
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

func (s *Server) handleDropTables(w http.ResponseWriter, _ *http.Request) {
	s.renderDropTables(w, -1)
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

func (s *Server) renderDropTablePartial(w http.ResponseWriter, activeIndex int) {
	tables := s.store.DropTables()
	rows := make([]dropTableItem, len(tables))
	for i, t := range tables {
		rows[i] = dropTableItem{Index: i, Table: t, Active: i == activeIndex}
	}
	data := dropTablesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(tables) {
		ed := dropTableEditData{Index: activeIndex, Table: tables[activeIndex], ItemOptions: s.itemOptions()}
		if err := s.templates.ExecuteTemplate(w, "drop-table-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">ドロップテーブルを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "dt-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "dt-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleDropTableEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	dt, err := s.store.DropTable(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := dropTableEditData{Index: index, Table: dt, ItemOptions: s.itemOptions()}
	if err := s.templates.ExecuteTemplate(w, "drop-table-edit", data); err != nil {
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
	s.renderDropTablePartial(w, s.findDropTableIndex(dt.Name))
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
	s.renderDropTablePartial(w, s.findDropTableIndex(name))
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
	s.renderDropTablePartial(w, -1)
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

func (s *Server) handleItemTables(w http.ResponseWriter, _ *http.Request) {
	s.renderItemTables(w, -1)
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

func (s *Server) renderItemTablePartial(w http.ResponseWriter, activeIndex int) {
	tables := s.store.ItemTables()
	rows := make([]itemTableItem, len(tables))
	for i, t := range tables {
		rows[i] = itemTableItem{Index: i, Table: t, Active: i == activeIndex}
	}
	data := itemTablesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(tables) {
		ed := itemTableEditData{Index: activeIndex, Table: tables[activeIndex], ItemOptions: s.itemOptions()}
		if err := s.templates.ExecuteTemplate(w, "item-table-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">アイテムテーブルを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "it-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "it-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleItemTableEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	it, err := s.store.ItemTable(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := itemTableEditData{Index: index, Table: it, ItemOptions: s.itemOptions()}
	if err := s.templates.ExecuteTemplate(w, "item-table-edit", data); err != nil {
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
	s.renderItemTablePartial(w, s.findItemTableIndex(it.Name))
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
	s.renderItemTablePartial(w, s.findItemTableIndex(name))
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
	s.renderItemTablePartial(w, -1)
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

func (s *Server) handleEnemyTables(w http.ResponseWriter, _ *http.Request) {
	s.renderEnemyTables(w, -1)
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

func (s *Server) renderEnemyTablePartial(w http.ResponseWriter, activeIndex int) {
	tables := s.store.EnemyTables()
	rows := make([]enemyTableItem, len(tables))
	for i, t := range tables {
		rows[i] = enemyTableItem{Index: i, Table: t, Active: i == activeIndex}
	}
	data := enemyTablesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(tables) {
		ed := enemyTableEditData{Index: activeIndex, Table: tables[activeIndex], MemberNames: s.memberNames()}
		if err := s.templates.ExecuteTemplate(w, "enemy-table-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">敵テーブルを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "et-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "et-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleEnemyTableEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	et, err := s.store.EnemyTable(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := enemyTableEditData{Index: index, Table: et, MemberNames: s.memberNames()}
	if err := s.templates.ExecuteTemplate(w, "enemy-table-edit", data); err != nil {
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
	s.renderEnemyTablePartial(w, s.findEnemyTableIndex(et.Name))
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
	s.renderEnemyTablePartial(w, s.findEnemyTableIndex(name))
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
	s.renderEnemyTablePartial(w, -1)
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
