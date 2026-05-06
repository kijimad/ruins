package editor

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/kijimaD/ruins/internal/raw"
)

// memberItem はテンプレートに渡すメンバー行データ
type memberItem struct {
	Index  int
	Member raw.Member
	Active bool
}

// memberEditData はメンバー編集テンプレートに渡すデータ
type memberEditData struct {
	Index             int
	Member            raw.Member
	SheetNames        []string
	CommandTableNames []string
	DropTableNames    []string
}

// membersData はメンバー一覧テンプレートに渡すデータ
type membersData struct {
	Items []memberItem
	Edit  *memberEditData
}

func (s *Server) handleMembers(w http.ResponseWriter, r *http.Request) {
	selected := parseSelectedIndex(r)
	s.renderMembers(w, selected)
}

func (s *Server) renderMembers(w http.ResponseWriter, activeIndex int) {
	members := s.store.Members()
	rows := make([]memberItem, len(members))
	for i, m := range members {
		rows[i] = memberItem{Index: i, Member: m, Active: i == activeIndex}
	}
	data := membersData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(members) {
		data.Edit = &memberEditData{
			Index:             activeIndex,
			Member:            members[activeIndex],
			SheetNames:        s.sheetNames(),
			CommandTableNames: s.commandTableNames(),
			DropTableNames:    s.dropTableNames(),
		}
	}
	if err := s.templates.ExecuteTemplate(w, "members", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// findMemberIndex は名前からメンバーのインデックスを返す。見つからなければ-1を返す
func (s *Server) findMemberIndex(name string) int {
	for i, m := range s.store.Members() {
		if m.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleMemberUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	member, err := s.store.Member(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	member = parseMemberForm(r, member)

	if err := s.store.UpdateMember(index, member); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newIndex := s.findMemberIndex(member.Name)
	http.Redirect(w, r, fmt.Sprintf("/members?selected=%d", newIndex), http.StatusSeeOther)
}

func (s *Server) handleMemberCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}

	member := raw.Member{Name: name}
	if err := s.store.AddMember(member); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newIndex := s.findMemberIndex(name)
	http.Redirect(w, r, fmt.Sprintf("/members?selected=%d", newIndex), http.StatusSeeOther)
}

func (s *Server) handleMemberDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteMember(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/members", http.StatusSeeOther)
}

// commandTableNames はコマンドテーブル名の一覧をソート済みで返す
func (s *Server) commandTableNames() []string {
	tables := s.store.CommandTables()
	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = t.Name
	}
	sort.Strings(names)
	return names
}

// dropTableNames はドロップテーブル名の一覧をソート済みで返す
func (s *Server) dropTableNames() []string {
	tables := s.store.DropTables()
	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = t.Name
	}
	sort.Strings(names)
	return names
}

// memberNames はメンバー名の一覧をソート済みで返す
func (s *Server) memberNames() []string {
	members := s.store.Members()
	names := make([]string, len(members))
	for i, m := range members {
		names[i] = m.Name
	}
	sort.Strings(names)
	return names
}

// parseMemberForm はHTTPフォームからMember構造体にフィールドを反映する
func parseMemberForm(r *http.Request, m raw.Member) raw.Member {
	m.Name = r.FormValue("name")
	m.SpriteSheetName = r.FormValue("sprite_sheet_name")
	m.SpriteKey = r.FormValue("sprite_key")
	m.FactionType = r.FormValue("faction_type")
	m.IsBoss = r.FormValue("is_boss") == "on"
	m.CommandTableName = r.FormValue("command_table_name")
	m.DropTableName = r.FormValue("drop_table_name")

	if r.FormValue("player") == "on" {
		t := true
		m.Player = &t
	} else {
		m.Player = nil
	}

	m.Abilities.Vitality, _ = strconv.Atoi(r.FormValue("vitality"))
	m.Abilities.Strength, _ = strconv.Atoi(r.FormValue("strength"))
	m.Abilities.Sensation, _ = strconv.Atoi(r.FormValue("sensation"))
	m.Abilities.Dexterity, _ = strconv.Atoi(r.FormValue("dexterity"))
	m.Abilities.Agility, _ = strconv.Atoi(r.FormValue("agility"))
	m.Abilities.Defense, _ = strconv.Atoi(r.FormValue("defense"))

	m.AnimKeys = parseAnimKeys(r)
	m.LightSource = parseLightSource(r, m.LightSource)

	// Dialog
	if r.FormValue("has_dialog") == "on" {
		if m.Dialog == nil {
			m.Dialog = &raw.DialogRaw{}
		}
		m.Dialog.MessageKey = r.FormValue("dialog_message_key")
	} else {
		m.Dialog = nil
	}

	return m
}
