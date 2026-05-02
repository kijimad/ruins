package editor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
)

// spriteFrame はスプライト1枚の座標情報
type spriteFrame struct {
	X, Y, W, H int
}

// Server はエディタのHTTPサーバー
type Server struct {
	store        *Store
	paletteStore *PaletteStore
	layoutStore  *LayoutStore
	templates    *template.Template
	assetsFS     fs.FS
	outputDir    string
	// sheetName → spriteKey → frame
	sprites map[string]map[string]spriteFrame
	// sheetName → PNG相対パス
	sheetPNGPaths map[string]string
	// sheetName → シート全体サイズ
	sheetSizes map[string]asepriteSize
	// アップロードされたスプライトシート画像（一時保持）
	uploadedSheet image.Image
}

// indexItem はテンプレートに渡すアイテム行データ
type indexItem struct {
	Index  int
	Item   raw.Item
	Active bool
}

// editData はアイテム編集テンプレートに渡すデータ
type editData struct {
	Index      int
	Item       raw.Item
	SheetNames []string
}

// indexData はindexテンプレートに渡すデータ
type indexData struct {
	Items []indexItem
	Edit  *editData
}

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

// recipeItem はテンプレートに渡すレシピ行データ
type recipeItem struct {
	Index  int
	Recipe raw.Recipe
	Active bool
}

// recipeEditData はレシピ編集テンプレートに渡すデータ
type recipeEditData struct {
	Index       int
	Recipe      raw.Recipe
	ItemOptions []itemSelectOption
}

// recipesData はレシピ一覧テンプレートに渡すデータ
type recipesData struct {
	Items []recipeItem
	Edit  *recipeEditData
}

// spriteGridData はスプライトグリッドテンプレートに渡すデータ
type spriteGridData struct {
	SheetName string
	Keys      []string
}

// cutterCell はスプライトカッターの1セル分のデータ
type cutterCell struct {
	Index int
	Row   int
	Col   int
}

// cutterData はスプライトカッターテンプレートに渡すデータ
type cutterData struct {
	Uploaded bool
	Cols     int
	Rows     int
	CellSize int
	Cells    []cutterCell
}

// NewServer は新しいエディタサーバーを作成する。
// assetsFS が nil の場合はスプライト表示を無効にする
func NewServer(store *Store, opts ...ServerOption) *Server {
	s := &Server{
		store:         store,
		sprites:       make(map[string]map[string]spriteFrame),
		sheetPNGPaths: make(map[string]string),
		sheetSizes:    make(map[string]asepriteSize),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.assetsFS != nil {
		s.loadSpriteSheets()
	}
	funcMap := template.FuncMap{
		"derefInt": func(p *int) int {
			if p != nil {
				return *p
			}
			return 0
		},
		"derefFloat": func(p *float64) float64 {
			if p != nil {
				return *p
			}
			return 0
		},
		"derefBool": func(p *bool) bool {
			if p != nil {
				return *p
			}
			return false
		},
		"isNotNil": func(v any) bool {
			if v == nil {
				return false
			}
			rv := reflect.ValueOf(v)
			return !rv.IsNil()
		},
		"melee": func(item raw.Item) raw.MeleeRaw {
			if item.Melee != nil {
				return *item.Melee
			}
			return raw.MeleeRaw{}
		},
		"fire": func(item raw.Item) raw.FireRaw {
			if item.Fire != nil {
				return *item.Fire
			}
			return raw.FireRaw{}
		},
		"consumable": func(item raw.Item) raw.Consumable {
			if item.Consumable != nil {
				return *item.Consumable
			}
			return raw.Consumable{}
		},
		"wearable": func(item raw.Item) raw.Wearable {
			if item.Wearable != nil {
				return *item.Wearable
			}
			return raw.Wearable{}
		},
	}
	funcMap["mul"] = func(a, b int) int { return a * b }
	funcMap["printf"] = fmt.Sprintf
	funcMap["selectData"] = func(name, value string) map[string]string {
		return map[string]string{"Name": name, "Value": value}
	}
	funcMap["selected"] = func(current, value string) template.HTMLAttr {
		if current == value {
			return "selected"
		}
		return ""
	}
	funcMap["spriteStyle"] = func(sheetName, spriteKey string, scale int) template.CSS {
		frames, ok := s.sprites[sheetName]
		if !ok {
			return ""
		}
		f, ok := frames[spriteKey]
		if !ok {
			// オートタイルの場合、ベースキーに_0を付加して探す
			f, ok = frames[spriteKey+"_0"]
			if !ok {
				return ""
			}
		}
		size, ok := s.sheetSizes[sheetName]
		if !ok {
			return ""
		}
		return template.CSS(fmt.Sprintf(
			"width:%dpx;height:%dpx;background:url('/sprites/%s') -%dpx -%dpx;background-size:%dpx %dpx;display:inline-block;image-rendering:pixelated;",
			f.W*scale, f.H*scale, sheetName, f.X*scale, f.Y*scale, size.W*scale, size.H*scale,
		))
	}
	s.templates = template.Must(
		template.Must(
			template.Must(
				template.Must(template.New("").Funcs(funcMap).Parse(templateText)).Parse(templateTextExtra),
			).Parse(templateTextPalette),
		).Parse(templateTextLayout),
	)
	return s
}

// ServerOption はServer構築時のオプション
type ServerOption func(*Server)

// WithAssetsFS はスプライト表示用のアセットファイルシステムを設定する
func WithAssetsFS(fsys fs.FS) ServerOption {
	return func(s *Server) {
		s.assetsFS = fsys
	}
}

// WithOutputDir はスプライト保存先ディレクトリを設定する
func WithOutputDir(dir string) ServerOption {
	return func(s *Server) {
		s.outputDir = dir
	}
}

// WithPaletteStore はパレットストアを設定する
func WithPaletteStore(ps *PaletteStore) ServerOption {
	return func(s *Server) {
		s.paletteStore = ps
	}
}

// WithLayoutStore はレイアウトストアを設定する
func WithLayoutStore(ls *LayoutStore) ServerOption {
	return func(s *Server) {
		s.layoutStore = ls
	}
}

// asepriteJSON は Aseprite JSON のフレーム情報を読み取るための最小限の構造体
type asepriteJSON struct {
	Frames []asepriteFrame `json:"frames"`
	Meta   asepriteMeta    `json:"meta"`
}

type asepriteFrame struct {
	Filename string       `json:"filename"`
	Frame    asepriteRect `json:"frame"`
}

type asepriteRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type asepriteSize struct {
	W int `json:"w"`
	H int `json:"h"`
}

type asepriteMeta struct {
	Image string       `json:"image"`
	Size  asepriteSize `json:"size"`
}

func (s *Server) loadSpriteSheets() {
	sheets := s.store.SpriteSheets()
	for _, sheet := range sheets {
		bs, err := fs.ReadFile(s.assetsFS, sheet.Path)
		if err != nil {
			log.Printf("スプライトシートJSON読み込み失敗: %s: %v", sheet.Path, err)
			continue
		}
		var data asepriteJSON
		if err := json.Unmarshal(bs, &data); err != nil {
			log.Printf("スプライトシートJSONパース失敗: %s: %v", sheet.Path, err)
			continue
		}

		pngPath := filepath.Join(filepath.Dir(sheet.Path), data.Meta.Image)
		s.sheetPNGPaths[sheet.Name] = pngPath
		s.sheetSizes[sheet.Name] = data.Meta.Size

		frames := make(map[string]spriteFrame)
		for _, f := range data.Frames {
			key := strings.TrimSuffix(f.Filename, "_")
			frames[key] = spriteFrame{X: f.Frame.X, Y: f.Frame.Y, W: f.Frame.W, H: f.Frame.H}
		}
		s.sprites[sheet.Name] = frames
	}
}

// ListenAndServe はHTTPサーバーを起動する
func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /items/{index}/edit", s.handleItemEdit)
	mux.HandleFunc("GET /items/{index}", s.handleItemRow)
	mux.HandleFunc("POST /items/{index}", s.handleItemUpdate)
	mux.HandleFunc("POST /items/new", s.handleItemCreate)
	mux.HandleFunc("DELETE /items/{index}", s.handleItemDelete)

	mux.HandleFunc("GET /members", s.handleMembers)
	mux.HandleFunc("GET /members/{index}/edit", s.handleMemberEdit)
	mux.HandleFunc("POST /members/{index}", s.handleMemberUpdate)
	mux.HandleFunc("POST /members/new", s.handleMemberCreate)
	mux.HandleFunc("DELETE /members/{index}", s.handleMemberDelete)

	mux.HandleFunc("GET /recipes", s.handleRecipes)
	mux.HandleFunc("GET /recipes/{index}/edit", s.handleRecipeEdit)
	mux.HandleFunc("POST /recipes/{index}", s.handleRecipeUpdate)
	mux.HandleFunc("POST /recipes/new", s.handleRecipeCreate)
	mux.HandleFunc("DELETE /recipes/{index}", s.handleRecipeDelete)

	mux.HandleFunc("GET /command-tables", s.handleCommandTables)
	mux.HandleFunc("GET /command-tables/{index}/edit", s.handleCommandTableEdit)
	mux.HandleFunc("POST /command-tables/{index}", s.handleCommandTableUpdate)
	mux.HandleFunc("POST /command-tables/new", s.handleCommandTableCreate)
	mux.HandleFunc("DELETE /command-tables/{index}", s.handleCommandTableDelete)

	mux.HandleFunc("GET /drop-tables", s.handleDropTables)
	mux.HandleFunc("GET /drop-tables/{index}/edit", s.handleDropTableEdit)
	mux.HandleFunc("POST /drop-tables/{index}", s.handleDropTableUpdate)
	mux.HandleFunc("POST /drop-tables/new", s.handleDropTableCreate)
	mux.HandleFunc("DELETE /drop-tables/{index}", s.handleDropTableDelete)

	mux.HandleFunc("GET /item-tables", s.handleItemTables)
	mux.HandleFunc("GET /item-tables/{index}/edit", s.handleItemTableEdit)
	mux.HandleFunc("POST /item-tables/{index}", s.handleItemTableUpdate)
	mux.HandleFunc("POST /item-tables/new", s.handleItemTableCreate)
	mux.HandleFunc("DELETE /item-tables/{index}", s.handleItemTableDelete)

	mux.HandleFunc("GET /enemy-tables", s.handleEnemyTables)
	mux.HandleFunc("GET /enemy-tables/{index}/edit", s.handleEnemyTableEdit)
	mux.HandleFunc("POST /enemy-tables/{index}", s.handleEnemyTableUpdate)
	mux.HandleFunc("POST /enemy-tables/new", s.handleEnemyTableCreate)
	mux.HandleFunc("DELETE /enemy-tables/{index}", s.handleEnemyTableDelete)

	mux.HandleFunc("GET /tiles", s.handleTiles)
	mux.HandleFunc("GET /tiles/{index}/edit", s.handleTileEdit)
	mux.HandleFunc("POST /tiles/{index}", s.handleTileUpdate)
	mux.HandleFunc("POST /tiles/new", s.handleTileCreate)
	mux.HandleFunc("DELETE /tiles/{index}", s.handleTileDelete)

	mux.HandleFunc("GET /props", s.handleProps)
	mux.HandleFunc("GET /props/{index}/edit", s.handlePropEdit)
	mux.HandleFunc("POST /props/{index}", s.handlePropUpdate)
	mux.HandleFunc("POST /props/new", s.handlePropCreate)
	mux.HandleFunc("DELETE /props/{index}", s.handlePropDelete)

	mux.HandleFunc("GET /professions", s.handleProfessions)
	mux.HandleFunc("GET /professions/{index}/edit", s.handleProfessionEdit)
	mux.HandleFunc("POST /professions/{index}", s.handleProfessionUpdate)
	mux.HandleFunc("POST /professions/new", s.handleProfessionCreate)
	mux.HandleFunc("DELETE /professions/{index}", s.handleProfessionDelete)

	mux.HandleFunc("GET /sprite-sheets", s.handleSpriteSheets)
	mux.HandleFunc("GET /sprite-sheets/{index}/edit", s.handleSpriteSheetEdit)
	mux.HandleFunc("POST /sprite-sheets/{index}", s.handleSpriteSheetUpdate)
	mux.HandleFunc("POST /sprite-sheets/new", s.handleSpriteSheetCreate)
	mux.HandleFunc("DELETE /sprite-sheets/{index}", s.handleSpriteSheetDelete)

	mux.HandleFunc("GET /cutter", s.handleCutter)
	mux.HandleFunc("POST /cutter/upload", s.handleCutterUpload)
	mux.HandleFunc("GET /cutter/preview", s.handleCutterPreview)
	mux.HandleFunc("POST /cutter/save", s.handleCutterSave)

	if s.paletteStore != nil {
		mux.HandleFunc("GET /palettes", s.handlePalettes)
		mux.HandleFunc("GET /palettes/{id}/edit", s.handlePaletteEdit)
		mux.HandleFunc("POST /palettes/{id}", s.handlePaletteUpdate)
		mux.HandleFunc("POST /palettes/new", s.handlePaletteCreate)
		mux.HandleFunc("DELETE /palettes/{id}", s.handlePaletteDelete)
	}

	if s.layoutStore != nil {
		mux.HandleFunc("GET /layouts", s.handleLayouts)
		mux.HandleFunc("GET /layouts/{dir}/{file}/{chunk}/edit", s.handleLayoutEdit)
		mux.HandleFunc("POST /layouts/{dir}/{file}/{chunk}", s.handleLayoutUpdate)
		mux.HandleFunc("GET /layouts/{dir}/{file}/{chunk}/preview", s.handleLayoutPreview)
		mux.HandleFunc("POST /layouts/{dir}/{file}/{chunk}/preview", s.handleLayoutPreview)
	}

	if s.assetsFS != nil {
		mux.HandleFunc("GET /sprites/{name}", s.handleSpritePNG)
		mux.HandleFunc("GET /sprites/{name}/keys", s.handleSpriteKeys)
	}

	log.Printf("エディタを起動しました: http://%s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleSpritePNG(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	pngPath, ok := s.sheetPNGPaths[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	bs, err := fs.ReadFile(s.assetsFS, pngPath)
	if err != nil {
		http.Error(w, "画像読み込み失敗", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	if _, err := w.Write(bs); err != nil {
		log.Printf("スプライトPNG書き込み失敗: %v", err)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	s.renderIndex(w, -1)
}

func (s *Server) renderIndex(w http.ResponseWriter, activeIndex int) {
	items := s.store.Items()
	rows := make([]indexItem, len(items))
	for i, item := range items {
		rows[i] = indexItem{Index: i, Item: item, Active: i == activeIndex}
	}
	data := indexData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(items) {
		data.Edit = &editData{
			Index:      activeIndex,
			Item:       items[activeIndex],
			SheetNames: s.sheetNames(),
		}
	}
	if err := s.templates.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderPartial はedit-panel向けに編集フォームを返し、OOBでサイドバーリストも更新する。
// サイドバーのスクロール位置はinnerHTMLの更新では維持される
func (s *Server) renderPartial(w http.ResponseWriter, activeIndex int) {
	items := s.store.Items()
	rows := make([]indexItem, len(items))
	for i, item := range items {
		rows[i] = indexItem{Index: i, Item: item, Active: i == activeIndex}
	}
	data := indexData{Items: rows}

	// edit-panelの中身を返す
	if activeIndex >= 0 && activeIndex < len(items) {
		ed := editData{
			Index:      activeIndex,
			Item:       items[activeIndex],
			SheetNames: s.sheetNames(),
		}
		if err := s.templates.ExecuteTemplate(w, "item-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">アイテムを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}

	// OOBでサイドバーリストを更新する
	if err := s.templates.ExecuteTemplate(w, "item-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "item-count-oob", data); err != nil {
		log.Printf("アイテム数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleItemRow(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	item, err := s.store.Item(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := s.templates.ExecuteTemplate(w, "item-entry", indexItem{Index: index, Item: item}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleItemEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	item, err := s.store.Item(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := editData{
		Index:      index,
		Item:       item,
		SheetNames: s.sheetNames(),
	}
	if err := s.templates.ExecuteTemplate(w, "item-edit", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSpriteKeys(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	frames, ok := s.sprites[name]
	if !ok {
		http.Error(w, "スプライトシートが見つからない", http.StatusNotFound)
		return
	}
	keys := make([]string, 0, len(frames))
	for k := range frames {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	data := spriteGridData{SheetName: name, Keys: keys}
	if err := s.templates.ExecuteTemplate(w, "sprite-grid", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// findItemIndex は名前からアイテムのインデックスを返す。見つからなければ-1を返す
func (s *Server) findItemIndex(name string) int {
	for i, item := range s.store.Items() {
		if item.Name == name {
			return i
		}
	}
	return -1
}

// sheetNames は読み込み済みスプライトシート名の一覧をソート済みで返す
func (s *Server) sheetNames() []string {
	names := make([]string, 0, len(s.sprites))
	for name := range s.sprites {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *Server) handleItemUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}

	item, err := s.store.Item(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	item = parseItemForm(r, item)

	if err := s.store.UpdateItem(index, item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ソートによりインデックスが変わるため、更新後の名前でインデックスを探す
	newIndex := s.findItemIndex(item.Name)
	s.renderPartial(w, newIndex)
}

func (s *Server) handleItemCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}

	item := raw.Item{
		Name:        name,
		Description: r.FormValue("description"),
	}
	if err := s.store.AddItem(item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newIndex := s.findItemIndex(name)
	s.renderPartial(w, newIndex)
}

func (s *Server) handleItemDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteItem(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderPartial(w, -1)
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

// itemSelectOption はセレクトボックス用のアイテム選択肢
type itemSelectOption struct {
	Name  string
	Label string
}

// itemOptions はアイテムの選択肢を種別順・名前順で返す
func (s *Server) itemOptions() []itemSelectOption {
	items := s.store.Items()
	// sortItemsで種別順にソート済み（Store.load/save時にソートされている）
	opts := make([]itemSelectOption, len(items))
	for i, item := range items {
		opts[i] = itemSelectOption{
			Name:  item.Name,
			Label: itemTypeLabel(item) + item.Name,
		}
	}
	return opts
}

// itemTypeLabel はアイテムの種別をテキストラベルで返す
func itemTypeLabel(item raw.Item) string {
	var labels []string
	if item.Melee != nil {
		labels = append(labels, "近")
	}
	if item.Fire != nil {
		labels = append(labels, "射")
	}
	if item.Wearable != nil {
		labels = append(labels, "防")
	}
	if item.Consumable != nil {
		labels = append(labels, "消")
	}
	if item.Ammo != nil {
		labels = append(labels, "弾")
	}
	if item.Book != nil {
		labels = append(labels, "本")
	}
	if len(labels) == 0 {
		return ""
	}
	return "[" + strings.Join(labels, "") + "] "
}

func (s *Server) handleRecipes(w http.ResponseWriter, _ *http.Request) {
	s.renderRecipes(w, -1)
}

func (s *Server) renderRecipes(w http.ResponseWriter, activeIndex int) {
	recipes := s.store.Recipes()
	rows := make([]recipeItem, len(recipes))
	for i, r := range recipes {
		rows[i] = recipeItem{Index: i, Recipe: r, Active: i == activeIndex}
	}
	data := recipesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(recipes) {
		data.Edit = &recipeEditData{
			Index:       activeIndex,
			Recipe:      recipes[activeIndex],
			ItemOptions: s.itemOptions(),
		}
	}
	if err := s.templates.ExecuteTemplate(w, "recipes", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderRecipePartial(w http.ResponseWriter, activeIndex int) {
	recipes := s.store.Recipes()
	rows := make([]recipeItem, len(recipes))
	for i, r := range recipes {
		rows[i] = recipeItem{Index: i, Recipe: r, Active: i == activeIndex}
	}
	data := recipesData{Items: rows}

	if activeIndex >= 0 && activeIndex < len(recipes) {
		ed := recipeEditData{
			Index:       activeIndex,
			Recipe:      recipes[activeIndex],
			ItemOptions: s.itemOptions(),
		}
		if err := s.templates.ExecuteTemplate(w, "recipe-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">レシピを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}

	if err := s.templates.ExecuteTemplate(w, "recipe-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "recipe-count-oob", data); err != nil {
		log.Printf("レシピ数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleRecipeEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	recipe, err := s.store.Recipe(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := recipeEditData{
		Index:       index,
		Recipe:      recipe,
		ItemOptions: s.itemOptions(),
	}
	if err := s.templates.ExecuteTemplate(w, "recipe-edit", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findRecipeIndex(name string) int {
	for i, r := range s.store.Recipes() {
		if r.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleRecipeUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}

	recipe := parseRecipeForm(r)

	if err := s.store.UpdateRecipe(index, recipe); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newIndex := s.findRecipeIndex(recipe.Name)
	s.renderRecipePartial(w, newIndex)
}

func (s *Server) handleRecipeCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}

	recipe := raw.Recipe{Name: name}
	if err := s.store.AddRecipe(recipe); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newIndex := s.findRecipeIndex(name)
	s.renderRecipePartial(w, newIndex)
}

func (s *Server) handleRecipeDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteRecipe(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderRecipePartial(w, -1)
}

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

// ================== タイル ==================

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

func (s *Server) handleTiles(w http.ResponseWriter, _ *http.Request) {
	s.renderTiles(w, -1)
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

func (s *Server) renderTilePartial(w http.ResponseWriter, activeIndex int) {
	tiles := s.store.Tiles()
	rows := make([]tileItem, len(tiles))
	for i, t := range tiles {
		rows[i] = tileItem{Index: i, Tile: t, Active: i == activeIndex}
	}
	data := tilesData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(tiles) {
		ed := tileEditData{Index: activeIndex, Tile: tiles[activeIndex], SheetNames: s.sheetNames()}
		if err := s.templates.ExecuteTemplate(w, "tile-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">タイルを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "tile-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "tile-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleTileEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	tile, err := s.store.Tile(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := tileEditData{Index: index, Tile: tile, SheetNames: s.sheetNames()}
	if err := s.templates.ExecuteTemplate(w, "tile-edit", data); err != nil {
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
	s.renderTilePartial(w, s.findTileIndex(tile.Name))
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
	s.renderTilePartial(w, s.findTileIndex(name))
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
	s.renderTilePartial(w, -1)
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

// ================== 置物 ==================

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

func (s *Server) handleProps(w http.ResponseWriter, _ *http.Request) {
	s.renderProps(w, -1)
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

func (s *Server) renderPropPartial(w http.ResponseWriter, activeIndex int) {
	props := s.store.Props()
	rows := make([]propItem, len(props))
	for i, p := range props {
		rows[i] = propItem{Index: i, Prop: p, Active: i == activeIndex}
	}
	data := propsData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(props) {
		ed := propEditData{Index: activeIndex, Prop: props[activeIndex], SheetNames: s.sheetNames()}
		if err := s.templates.ExecuteTemplate(w, "prop-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">置物を選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "prop-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "prop-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handlePropEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	prop, err := s.store.Prop(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := propEditData{Index: index, Prop: prop, SheetNames: s.sheetNames()}
	if err := s.templates.ExecuteTemplate(w, "prop-edit", data); err != nil {
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
	s.renderPropPartial(w, s.findPropIndex(prop.Name))
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
	s.renderPropPartial(w, s.findPropIndex(name))
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
	s.renderPropPartial(w, -1)
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
		p.LightSource.Color = color.RGBA{R: uint8(cr), G: uint8(cg), B: uint8(cb), A: uint8(ca)}
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

// ================== 職業 ==================

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

// ================== スプライトシート ==================

type spriteSheetItem struct {
	Index  int
	Sheet  raw.SpriteSheet
	Active bool
}

type spriteSheetEditData struct {
	Index int
	Sheet raw.SpriteSheet
}

type spriteSheetsData struct {
	Items []spriteSheetItem
	Edit  *spriteSheetEditData
}

func (s *Server) handleSpriteSheets(w http.ResponseWriter, _ *http.Request) {
	s.renderSpriteSheets(w, -1)
}

func (s *Server) renderSpriteSheets(w http.ResponseWriter, activeIndex int) {
	sheets := s.store.SpriteSheets()
	rows := make([]spriteSheetItem, len(sheets))
	for i, sh := range sheets {
		rows[i] = spriteSheetItem{Index: i, Sheet: sh, Active: i == activeIndex}
	}
	data := spriteSheetsData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(sheets) {
		data.Edit = &spriteSheetEditData{Index: activeIndex, Sheet: sheets[activeIndex]}
	}
	if err := s.templates.ExecuteTemplate(w, "sprite-sheets", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderSpriteSheetPartial(w http.ResponseWriter, activeIndex int) {
	sheets := s.store.SpriteSheets()
	rows := make([]spriteSheetItem, len(sheets))
	for i, sh := range sheets {
		rows[i] = spriteSheetItem{Index: i, Sheet: sh, Active: i == activeIndex}
	}
	data := spriteSheetsData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(sheets) {
		ed := spriteSheetEditData{Index: activeIndex, Sheet: sheets[activeIndex]}
		if err := s.templates.ExecuteTemplate(w, "sprite-sheet-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">スプライトシートを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "ss-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "ss-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleSpriteSheetEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	ss, err := s.store.SpriteSheetByIndex(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := spriteSheetEditData{Index: index, Sheet: ss}
	if err := s.templates.ExecuteTemplate(w, "sprite-sheet-edit", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findSpriteSheetIndex(name string) int {
	for i, sh := range s.store.SpriteSheets() {
		if sh.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleSpriteSheetUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	ss := raw.SpriteSheet{Name: r.FormValue("name"), Path: r.FormValue("path")}
	if err := s.store.UpdateSpriteSheet(index, ss); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderSpriteSheetPartial(w, s.findSpriteSheetIndex(ss.Name))
}

func (s *Server) handleSpriteSheetCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}
	ss := raw.SpriteSheet{Name: name}
	if err := s.store.AddSpriteSheet(ss); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderSpriteSheetPartial(w, s.findSpriteSheetIndex(name))
}

func (s *Server) handleSpriteSheetDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteSpriteSheet(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderSpriteSheetPartial(w, -1)
}

// parseRecipeForm はHTTPフォームからRecipe構造体を構築する
func parseRecipeForm(r *http.Request) raw.Recipe {
	recipe := raw.Recipe{
		Name: r.FormValue("name"),
	}
	// 素材は input_name_0, input_amount_0, input_name_1, ... の形式
	for i := 0; ; i++ {
		name := strings.TrimSpace(r.FormValue(fmt.Sprintf("input_name_%d", i)))
		if name == "" {
			break
		}
		amount, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("input_amount_%d", i)))
		if amount <= 0 {
			amount = 1
		}
		recipe.Inputs = append(recipe.Inputs, raw.RecipeInput{Name: name, Amount: amount})
	}
	return recipe
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
		m.LightSource.Color = color.RGBA{R: uint8(cr), G: uint8(cg), B: uint8(cb), A: uint8(ca)}
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

func (s *Server) handleCutter(w http.ResponseWriter, _ *http.Request) {
	data := s.buildCutterData()
	if err := s.templates.ExecuteTemplate(w, "cutter", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleCutterUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "ファイルのパースに失敗", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("sheet")
	if err != nil {
		http.Error(w, "ファイルの読み込みに失敗", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("アップロードファイルのクローズに失敗: %v", err)
		}
	}()

	img, err := png.Decode(file)
	if err != nil {
		http.Error(w, "PNG画像のデコードに失敗", http.StatusBadRequest)
		return
	}
	s.uploadedSheet = img

	w.Header().Set("HX-Redirect", "/cutter")
	w.WriteHeader(http.StatusOK)
}

// handleCutterPreview はアップロード済み画像をPNGで返す
func (s *Server) handleCutterPreview(w http.ResponseWriter, _ *http.Request) {
	if s.uploadedSheet == nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	if err := png.Encode(w, s.uploadedSheet); err != nil {
		log.Printf("プレビュー画像の書き込みに失敗: %v", err)
	}
}

func (s *Server) handleCutterSave(w http.ResponseWriter, r *http.Request) {
	if s.uploadedSheet == nil {
		http.Error(w, "画像がアップロードされていません", http.StatusBadRequest)
		return
	}
	if s.outputDir == "" {
		http.Error(w, "出力先が設定されていません", http.StatusInternalServerError)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}

	bounds := s.uploadedSheet.Bounds()
	cellSize := 32
	cols := bounds.Dx() / cellSize
	rows := bounds.Dy() / cellSize
	saved := 0

	for row := range rows {
		for col := range cols {
			idx := row*cols + col
			name := strings.TrimSpace(r.FormValue(fmt.Sprintf("name_%d", idx)))
			if name == "" {
				continue
			}
			rect := image.Rect(col*cellSize, row*cellSize, (col+1)*cellSize, (row+1)*cellSize)
			cell := image.NewRGBA(image.Rect(0, 0, cellSize, cellSize))
			draw.Draw(cell, cell.Bounds(), s.uploadedSheet, rect.Min, draw.Src)

			// 完全に透明なセルはスキップする
			if isTransparent(cell) {
				continue
			}

			filename := name + "_.png"
			path := filepath.Join(s.outputDir, filename)
			if err := savePNG(path, cell); err != nil {
				http.Error(w, fmt.Sprintf("%s の保存に失敗: %v", name, err), http.StatusInternalServerError)
				return
			}
			saved++
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := fmt.Fprintf(w, `<div class="alert alert-success">%d 個のスプライトを保存しました</div>`, saved); err != nil {
		log.Printf("レスポンス書き込みに失敗: %v", err)
	}
}

func (s *Server) buildCutterData() cutterData {
	if s.uploadedSheet == nil {
		return cutterData{Uploaded: false}
	}
	bounds := s.uploadedSheet.Bounds()
	cellSize := 32
	cols := bounds.Dx() / cellSize
	rows := bounds.Dy() / cellSize
	cells := make([]cutterCell, 0, cols*rows)
	for row := range rows {
		for col := range cols {
			cells = append(cells, cutterCell{
				Index: row*cols + col,
				Row:   row,
				Col:   col,
			})
		}
	}
	return cutterData{
		Uploaded: true,
		Cols:     cols,
		Rows:     rows,
		CellSize: cellSize,
		Cells:    cells,
	}
}

// isTransparent は画像が完全に透明かどうかを判定する
func isTransparent(img *image.RGBA) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				return false
			}
		}
	}
	return true
}

func savePNG(path string, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// parseItemForm はHTTPフォームからItem構造体にフィールドを反映する
func parseItemForm(r *http.Request, item raw.Item) raw.Item {
	item.Name = r.FormValue("name")
	item.Description = r.FormValue("description")
	item.SpriteSheetName = r.FormValue("sprite_sheet_name")
	item.SpriteKey = r.FormValue("sprite_key")

	if v := r.FormValue("value"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			item.Value = n
		}
	} else {
		item.Value = 0
	}

	if v := r.FormValue("weight"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			item.Weight = &f
		}
	} else {
		item.Weight = nil
	}

	if v := r.FormValue("inflicts_damage"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			item.InflictsDamage = &n
		}
	} else {
		item.InflictsDamage = nil
	}

	if v := r.FormValue("provides_nutrition"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			item.ProvidesNutrition = &n
		}
	} else {
		item.ProvidesNutrition = nil
	}

	stackable := r.FormValue("stackable") == "on"
	if stackable {
		item.Stackable = &stackable
	} else {
		item.Stackable = nil
	}

	// Melee
	if r.FormValue("has_melee") == "on" {
		if item.Melee == nil {
			item.Melee = &raw.MeleeRaw{}
		}
		item.Melee.Accuracy, _ = strconv.Atoi(r.FormValue("melee_accuracy"))
		item.Melee.Damage, _ = strconv.Atoi(r.FormValue("melee_damage"))
		item.Melee.AttackCount, _ = strconv.Atoi(r.FormValue("melee_attack_count"))
		item.Melee.Cost, _ = strconv.Atoi(r.FormValue("melee_cost"))
		item.Melee.Element = r.FormValue("melee_element")
		item.Melee.AttackCategory = r.FormValue("melee_attack_category")
		item.Melee.TargetGroup = r.FormValue("melee_target_group")
		item.Melee.TargetNum = r.FormValue("melee_target_num")

		if item.Weapon == nil {
			item.Weapon = &raw.Weapon{}
		}
	} else {
		item.Melee = nil
	}

	// Fire
	if r.FormValue("has_fire") == "on" {
		if item.Fire == nil {
			item.Fire = &raw.FireRaw{}
		}
		item.Fire.Accuracy, _ = strconv.Atoi(r.FormValue("fire_accuracy"))
		item.Fire.Damage, _ = strconv.Atoi(r.FormValue("fire_damage"))
		item.Fire.AttackCount, _ = strconv.Atoi(r.FormValue("fire_attack_count"))
		item.Fire.Cost, _ = strconv.Atoi(r.FormValue("fire_cost"))
		item.Fire.Element = r.FormValue("fire_element")
		item.Fire.AttackCategory = r.FormValue("fire_attack_category")
		item.Fire.TargetGroup = r.FormValue("fire_target_group")
		item.Fire.TargetNum = r.FormValue("fire_target_num")
		item.Fire.MagazineSize, _ = strconv.Atoi(r.FormValue("fire_magazine_size"))
		item.Fire.ReloadEffort, _ = strconv.Atoi(r.FormValue("fire_reload_effort"))
		item.Fire.AmmoTag = r.FormValue("fire_ammo_tag")

		if item.Weapon == nil {
			item.Weapon = &raw.Weapon{}
		}
	} else {
		item.Fire = nil
	}

	// Consumable
	if r.FormValue("has_consumable") == "on" {
		if item.Consumable == nil {
			item.Consumable = &raw.Consumable{}
		}
		item.Consumable.UsableScene = r.FormValue("consumable_usable_scene")
		item.Consumable.TargetGroup = r.FormValue("consumable_target_group")
		item.Consumable.TargetNum = r.FormValue("consumable_target_num")
	} else {
		item.Consumable = nil
	}

	// Wearable
	if r.FormValue("has_wearable") == "on" {
		if item.Wearable == nil {
			item.Wearable = &raw.Wearable{}
		}
		item.Wearable.Defense, _ = strconv.Atoi(r.FormValue("wearable_defense"))
		item.Wearable.EquipmentCategory = r.FormValue("wearable_equipment_category")
		item.Wearable.InsulationCold, _ = strconv.Atoi(r.FormValue("wearable_insulation_cold"))
		item.Wearable.InsulationHeat, _ = strconv.Atoi(r.FormValue("wearable_insulation_heat"))
	} else {
		item.Wearable = nil
	}

	return item
}
