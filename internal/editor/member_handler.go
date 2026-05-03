package editor

import (
	"fmt"
	"image/color"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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

func (s *Server) handleMembers(w http.ResponseWriter, _ *http.Request) {
	s.renderMembers(w, -1)
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

// renderMemberPartial はedit-panel向けに編集フォームを返し、OOBでサイドバーリストも更新する
func (s *Server) renderMemberPartial(w http.ResponseWriter, activeIndex int) {
	members := s.store.Members()
	rows := make([]memberItem, len(members))
	for i, m := range members {
		rows[i] = memberItem{Index: i, Member: m, Active: i == activeIndex}
	}
	data := membersData{Items: rows}

	if activeIndex >= 0 && activeIndex < len(members) {
		ed := memberEditData{
			Index:             activeIndex,
			Member:            members[activeIndex],
			SheetNames:        s.sheetNames(),
			CommandTableNames: s.commandTableNames(),
			DropTableNames:    s.dropTableNames(),
		}
		if err := s.templates.ExecuteTemplate(w, "member-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">メンバーを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}

	if err := s.templates.ExecuteTemplate(w, "member-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "member-count-oob", data); err != nil {
		log.Printf("メンバー数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleMemberEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	member, err := s.store.Member(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := memberEditData{
		Index:             index,
		Member:            member,
		SheetNames:        s.sheetNames(),
		CommandTableNames: s.commandTableNames(),
		DropTableNames:    s.dropTableNames(),
	}
	if err := s.templates.ExecuteTemplate(w, "member-edit", data); err != nil {
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
	s.renderMemberPartial(w, newIndex)
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
	s.renderMemberPartial(w, newIndex)
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
	s.renderMemberPartial(w, -1)
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

	// AnimKeys
	animKeysStr := strings.TrimSpace(r.FormValue("anim_keys"))
	if animKeysStr != "" {
		keys := strings.Split(animKeysStr, ",")
		m.AnimKeys = make([]string, 0, len(keys))
		for _, k := range keys {
			k = strings.TrimSpace(k)
			if k != "" {
				m.AnimKeys = append(m.AnimKeys, k)
			}
		}
	} else {
		m.AnimKeys = nil
	}

	// LightSource
	if r.FormValue("has_light") == "on" {
		if m.LightSource == nil {
			m.LightSource = &gc.LightSource{}
		}
		radius, _ := strconv.Atoi(r.FormValue("light_radius"))
		m.LightSource.Radius = consts.Tile(radius)
		m.LightSource.Enabled = r.FormValue("light_enabled") == "on"
		cr, _ := strconv.Atoi(r.FormValue("light_r"))
		cg, _ := strconv.Atoi(r.FormValue("light_g"))
		cb, _ := strconv.Atoi(r.FormValue("light_b"))
		ca, _ := strconv.Atoi(r.FormValue("light_a"))
		m.LightSource.Color = color.RGBA{R: clampUint8(cr), G: clampUint8(cg), B: clampUint8(cb), A: clampUint8(ca)}
	} else {
		m.LightSource = nil
	}

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
