package editor

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/kijimaD/ruins/internal/raw"
)

//go:embed templates/*.html
var templateFS embed.FS

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

// spriteGridData はスプライトグリッドテンプレートに渡すデータ
type spriteGridData struct {
	SheetName string
	Keys      []string
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
	funcMap["contains"] = func(slice []string, item string) bool {
		for _, s := range slice {
			if s == item {
				return true
			}
		}
		return false
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
		template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html"),
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
	mux.HandleFunc("GET /{$}", s.handleDashboard)
	mux.HandleFunc("GET /items", s.handleIndex)
	mux.HandleFunc("POST /items/{index}", s.handleItemUpdate)
	mux.HandleFunc("POST /items/new", s.handleItemCreate)
	mux.HandleFunc("POST /items/{index}/delete", s.handleItemDelete)

	mux.HandleFunc("GET /members", s.handleMembers)
	mux.HandleFunc("POST /members/{index}", s.handleMemberUpdate)
	mux.HandleFunc("POST /members/new", s.handleMemberCreate)
	mux.HandleFunc("POST /members/{index}/delete", s.handleMemberDelete)

	mux.HandleFunc("GET /recipes", s.handleRecipes)
	mux.HandleFunc("POST /recipes/{index}", s.handleRecipeUpdate)
	mux.HandleFunc("POST /recipes/new", s.handleRecipeCreate)
	mux.HandleFunc("POST /recipes/{index}/delete", s.handleRecipeDelete)

	mux.HandleFunc("GET /command-tables", s.handleCommandTables)
	mux.HandleFunc("POST /command-tables/{index}", s.handleCommandTableUpdate)
	mux.HandleFunc("POST /command-tables/new", s.handleCommandTableCreate)
	mux.HandleFunc("POST /command-tables/{index}/delete", s.handleCommandTableDelete)

	mux.HandleFunc("GET /drop-tables", s.handleDropTables)
	mux.HandleFunc("POST /drop-tables/{index}", s.handleDropTableUpdate)
	mux.HandleFunc("POST /drop-tables/new", s.handleDropTableCreate)
	mux.HandleFunc("POST /drop-tables/{index}/delete", s.handleDropTableDelete)

	mux.HandleFunc("GET /item-tables", s.handleItemTables)
	mux.HandleFunc("POST /item-tables/{index}", s.handleItemTableUpdate)
	mux.HandleFunc("POST /item-tables/new", s.handleItemTableCreate)
	mux.HandleFunc("POST /item-tables/{index}/delete", s.handleItemTableDelete)

	mux.HandleFunc("GET /enemy-tables", s.handleEnemyTables)
	mux.HandleFunc("POST /enemy-tables/{index}", s.handleEnemyTableUpdate)
	mux.HandleFunc("POST /enemy-tables/new", s.handleEnemyTableCreate)
	mux.HandleFunc("POST /enemy-tables/{index}/delete", s.handleEnemyTableDelete)

	mux.HandleFunc("GET /tiles", s.handleTiles)
	mux.HandleFunc("POST /tiles/{index}", s.handleTileUpdate)
	mux.HandleFunc("POST /tiles/new", s.handleTileCreate)
	mux.HandleFunc("POST /tiles/{index}/delete", s.handleTileDelete)

	mux.HandleFunc("GET /props", s.handleProps)
	mux.HandleFunc("POST /props/{index}", s.handlePropUpdate)
	mux.HandleFunc("POST /props/new", s.handlePropCreate)
	mux.HandleFunc("POST /props/{index}/delete", s.handlePropDelete)

	mux.HandleFunc("GET /professions", s.handleProfessions)
	mux.HandleFunc("POST /professions/{index}", s.handleProfessionUpdate)
	mux.HandleFunc("POST /professions/new", s.handleProfessionCreate)
	mux.HandleFunc("POST /professions/{index}/delete", s.handleProfessionDelete)

	mux.HandleFunc("GET /sprite-sheets", s.handleSpriteSheets)
	mux.HandleFunc("POST /sprite-sheets/{index}", s.handleSpriteSheetUpdate)
	mux.HandleFunc("POST /sprite-sheets/new", s.handleSpriteSheetCreate)
	mux.HandleFunc("POST /sprite-sheets/{index}/delete", s.handleSpriteSheetDelete)

	mux.HandleFunc("GET /cutter", s.handleCutter)
	mux.HandleFunc("POST /cutter/upload", s.handleCutterUpload)
	mux.HandleFunc("GET /cutter/preview", s.handleCutterPreview)
	mux.HandleFunc("POST /cutter/save", s.handleCutterSave)

	if s.paletteStore != nil {
		mux.HandleFunc("GET /palettes", s.handlePalettes)
		mux.HandleFunc("POST /palettes/{id}", s.handlePaletteUpdate)
		mux.HandleFunc("POST /palettes/new", s.handlePaletteCreate)
		mux.HandleFunc("POST /palettes/{id}/delete", s.handlePaletteDelete)
	}

	if s.layoutStore != nil {
		mux.HandleFunc("GET /layouts", s.handleLayouts)
		mux.HandleFunc("POST /layouts/new", s.handleLayoutCreate)
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

func (s *Server) handleDashboard(w http.ResponseWriter, _ *http.Request) {
	if err := s.templates.ExecuteTemplate(w, "dashboard", nil); err != nil {
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

// sheetNames は読み込み済みスプライトシート名の一覧をソート済みで返す
func (s *Server) sheetNames() []string {
	names := make([]string, 0, len(s.sprites))
	for name := range s.sprites {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
