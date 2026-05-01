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
	store     *Store
	templates *template.Template
	assetsFS  fs.FS
	outputDir string
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
	Index      int
	Member     raw.Member
	SheetNames []string
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
	Index     int
	Recipe    raw.Recipe
	ItemNames []string
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
			return ""
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
	s.templates = template.Must(template.New("").Funcs(funcMap).Parse(templateText))
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

	mux.HandleFunc("GET /cutter", s.handleCutter)
	mux.HandleFunc("POST /cutter/upload", s.handleCutterUpload)
	mux.HandleFunc("GET /cutter/preview", s.handleCutterPreview)
	mux.HandleFunc("POST /cutter/save", s.handleCutterSave)

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
			Index:      activeIndex,
			Member:     members[activeIndex],
			SheetNames: s.sheetNames(),
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
			Index:      activeIndex,
			Member:     members[activeIndex],
			SheetNames: s.sheetNames(),
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
		Index:      index,
		Member:     member,
		SheetNames: s.sheetNames(),
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

// itemNames はアイテム名の一覧をソート済みで返す
func (s *Server) itemNames() []string {
	items := s.store.Items()
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	sort.Strings(names)
	return names
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
			Index:     activeIndex,
			Recipe:    recipes[activeIndex],
			ItemNames: s.itemNames(),
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
			Index:     activeIndex,
			Recipe:    recipes[activeIndex],
			ItemNames: s.itemNames(),
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
		Index:     index,
		Recipe:    recipe,
		ItemNames: s.itemNames(),
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
