package editor

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLayoutTest(t *testing.T) *Server {
	t.Helper()

	// テスト用ディレクトリを作成する
	tmpDir := t.TempDir()
	layoutDir := filepath.Join(tmpDir, "layouts")
	paletteDir := filepath.Join(tmpDir, "palettes")
	require.NoError(t, os.MkdirAll(layoutDir, 0o755))
	require.NoError(t, os.MkdirAll(paletteDir, 0o755))

	// テスト用パレットファイルを作成する
	paletteTOML := `[palette]
id = "test_pal"
description = "テスト用パレット"
[palette.terrain]
"#" = "wall"
"." = "floor"
[palette.props]
"+" = { id = "door", tile = "floor" }
[palette.npcs]
"M" = { id = "boss", tile = "floor" }
`
	require.NoError(t, os.WriteFile(filepath.Join(paletteDir, "test_pal.toml"), []byte(paletteTOML), 0o644))

	// テスト用レイアウトファイルを作成する
	layoutTOML := `[[chunk]]
name = "3x3_test"
palettes = ["test_pal"]
weight = 100
map = """
###
#.#
###
"""
`
	require.NoError(t, os.WriteFile(filepath.Join(layoutDir, "test.toml"), []byte(layoutTOML), 0o644))

	// raw.tomlの最小限ファイルを作成する
	rawTOML := `[[Tiles]]
Name = "wall"
Description = "壁"
BlockPass = true
BlockView = true
[Tiles.SpriteRender]
SpriteSheetName = "sheet1"
SpriteKey = "wall"

[[Tiles]]
Name = "floor"
Description = "床"
BlockPass = false
BlockView = false
[Tiles.SpriteRender]
SpriteSheetName = "sheet1"
SpriteKey = "floor"

[[Props]]
Name = "door"
Description = "ドア"
BlockPass = false
BlockView = false
[Props.SpriteRender]
SpriteSheetName = "sheet1"
SpriteKey = "door"

[[Members]]
Name = "boss"
SpriteSheetName = "sheet1"
SpriteKey = "boss"
[Members.Abilities]
Strength = 10
Vitality = 10
Dexterity = 10
Sensation = 10
`
	rawPath := filepath.Join(tmpDir, "raw.toml")
	require.NoError(t, os.WriteFile(rawPath, []byte(rawTOML), 0o644))

	store, err := NewStore(rawPath)
	require.NoError(t, err)

	paletteStore, err := NewPaletteStore(paletteDir)
	require.NoError(t, err)

	layoutStore, err := NewLayoutStore([]string{layoutDir})
	require.NoError(t, err)

	server := NewServer(store,
		WithPaletteStore(paletteStore),
		WithLayoutStore(layoutStore),
	)

	// テスト用スプライトデータを設定する
	server.sprites["sheet1"] = map[string]spriteFrame{
		"wall":  {X: 0, Y: 0, W: 32, H: 32},
		"floor": {X: 32, Y: 0, W: 32, H: 32},
		"door":  {X: 64, Y: 0, W: 32, H: 32},
		"boss":  {X: 96, Y: 0, W: 32, H: 32},
	}
	server.sheetSizes["sheet1"] = asepriteSize{W: 256, H: 256}

	return server
}

func TestHandleLayoutPreview(t *testing.T) {
	t.Parallel()
	server := setupLayoutTest(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /layouts/{dir}/{file}/{chunk}/preview", server.handleLayoutPreview)
	mux.HandleFunc("POST /layouts/{dir}/{file}/{chunk}/preview", server.handleLayoutPreview)

	t.Run("プレビューがHTMLを返す", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest("GET", "/layouts/layouts/test/3x3_test/preview", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()

		// グリッドが生成されている
		assert.Contains(t, body, "preview-grid")
		assert.Contains(t, body, "grid-template-columns:repeat(3, 32px)")

		// スプライトスタイルが含まれている
		assert.Contains(t, body, "sprite-layer")
		assert.Contains(t, body, "/sprites/sheet1")

		// ツールチップ情報が含まれている
		assert.Contains(t, body, "wall")
		assert.Contains(t, body, "floor")
	})

	t.Run("POSTでテキストエリアの内容をプレビューする", func(t *testing.T) {
		t.Parallel()
		form := strings.NewReader("map_content=" + url.QueryEscape("##\n.#"))
		req := httptest.NewRequest("POST", "/layouts/layouts/test/3x3_test/preview", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()

		// POSTで送った2列のグリッドになっている
		assert.Contains(t, body, "grid-template-columns:repeat(2, 32px)")
		assert.Contains(t, body, "floor")
	})

	t.Run("存在しないチャンクは404", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest("GET", "/layouts/layouts/test/nonexistent/preview", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandleLayoutEdit(t *testing.T) {
	t.Parallel()
	server := setupLayoutTest(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /layouts/{dir}/{file}/{chunk}/edit", server.handleLayoutEdit)

	req := httptest.NewRequest("GET", "/layouts/layouts/test/3x3_test/edit", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// インラインプレビューが含まれている
	assert.Contains(t, body, "preview-grid")
	// テキストエリアにマップが含まれている
	assert.Contains(t, body, "###")
	assert.Contains(t, body, "#.#")

	// チートシートにパレットのマッピングとスプライトが含まれている
	assert.Contains(t, body, "文字チートシート")
	assert.Contains(t, body, "<code>#</code>")
	assert.Contains(t, body, "wall")
	assert.Contains(t, body, "floor")
	assert.Contains(t, body, "door")
	assert.Contains(t, body, "boss")
	assert.Contains(t, body, "/sprites/sheet1")
}

func TestHandleLayoutUpdate_UndefinedChar(t *testing.T) {
	t.Parallel()
	server := setupLayoutTest(t)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /layouts/{dir}/{file}/{chunk}", server.handleLayoutUpdate)

	// パレットに定義されていない文字 "X" を含むマップを送信する
	form := strings.NewReader("map_content=" + url.QueryEscape("###\n#X#\n###"))
	req := httptest.NewRequest("POST", "/layouts/layouts/test/3x3_test", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "X")
	assert.Contains(t, w.Body.String(), "パレットに未定義の文字があります")
}

func TestHandleLayoutUpdate_ValidChars(t *testing.T) {
	t.Parallel()
	server := setupLayoutTest(t)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /layouts/{dir}/{file}/{chunk}", server.handleLayoutUpdate)

	// パレットに定義されている文字のみのマップを送信する
	form := strings.NewReader("map_content=" + url.QueryEscape("###\n#.#\n###"))
	req := httptest.NewRequest("POST", "/layouts/layouts/test/3x3_test", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/layouts/layouts/test/3x3_test/edit")
}

func TestHandleLayoutUpdate_PaletteChange(t *testing.T) {
	t.Parallel()
	server := setupLayoutTest(t)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /layouts/{dir}/{file}/{chunk}", server.handleLayoutUpdate)

	// パレットを変更して保存する
	formData := url.Values{}
	formData.Set("map_content", "###\n#.#\n###")
	formData.Add("palettes", "test_pal")
	form := strings.NewReader(formData.Encode())
	req := httptest.NewRequest("POST", "/layouts/layouts/test/3x3_test", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)

	// パレットが保存されたことを確認する
	chunk, err := server.layoutStore.GetChunk("layouts", "test.toml", "3x3_test")
	require.NoError(t, err)
	assert.Equal(t, []string{"test_pal"}, chunk.Palettes)
}

func TestValidateMapContent(t *testing.T) {
	t.Parallel()

	palette := &maptemplate.Palette{
		Terrain: map[string]string{"#": "wall", ".": "floor"},
		Props:   map[string]maptemplate.PaletteEntry{"+": {ID: "door", Tile: "floor"}},
		NPCs:    map[string]maptemplate.PaletteEntry{"M": {ID: "boss", Tile: "floor"}},
	}

	t.Run("定義済み文字のみならエラーなし", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, validateMapContent("###\n#.#\n#+M", palette, nil))
	})

	t.Run("未定義文字があればエラー", func(t *testing.T) {
		t.Parallel()
		err := validateMapContent("###\n#X#\n#Z#", palette, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "X")
		assert.Contains(t, err.Error(), "Z")
	})

	t.Run("空白と改行は無視する", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, validateMapContent("# #\n. .\n", palette, nil))
	})

	t.Run("空文字列はエラーなし", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, validateMapContent("", palette, nil))
	})

	t.Run("placementsのプレースホルダ文字はスキップする", func(t *testing.T) {
		t.Parallel()
		placements := []maptemplate.ChunkPlacement{
			{Chunks: []string{"5x5_room"}, ID: "A"},
			{Chunks: []string{"4x4_room"}, ID: "B"},
		}
		// @とA,Bはプレースホルダなのでエラーにならない
		assert.NoError(t, validateMapContent("#@@@@\n#@@@A\n#@@@B", palette, placements))
	})

	t.Run("placementsがあっても未定義文字はエラー", func(t *testing.T) {
		t.Parallel()
		placements := []maptemplate.ChunkPlacement{
			{Chunks: []string{"5x5_room"}, ID: "A"},
		}
		err := validateMapContent("#X@A", palette, placements)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "X")
		// @とAはエラーに含まれない
		assert.NotContains(t, err.Error(), "@")
		assert.NotContains(t, err.Error(), "A")
	})
}

func TestHandleLayoutPreview_WithPlacements(t *testing.T) {
	t.Parallel()

	// テスト用ディレクトリを作成する
	tmpDir := t.TempDir()
	layoutDir := filepath.Join(tmpDir, "layouts")
	chunkDir := filepath.Join(tmpDir, "chunks")
	paletteDir := filepath.Join(tmpDir, "palettes")
	require.NoError(t, os.MkdirAll(layoutDir, 0o755))
	require.NoError(t, os.MkdirAll(chunkDir, 0o755))
	require.NoError(t, os.MkdirAll(paletteDir, 0o755))

	paletteTOML := `[palette]
id = "test_pal"
description = "テスト用パレット"
[palette.terrain]
"#" = "wall"
"." = "floor"
`
	require.NoError(t, os.WriteFile(filepath.Join(paletteDir, "test_pal.toml"), []byte(paletteTOML), 0o644))

	// 子チャンク: 2x2の部屋
	childTOML := `[[chunk]]
name = "2x2_room"
palettes = ["test_pal"]
weight = 100
map = """
..
..
"""
`
	require.NoError(t, os.WriteFile(filepath.Join(chunkDir, "room.toml"), []byte(childTOML), 0o644))

	// 親レイアウト: 4x4のマップにplacementsで子チャンクを埋め込む
	parentTOML := `[[chunk]]
name = "4x4_parent"
palettes = ["test_pal"]
weight = 100
map = """
####
#@@#
#@A#
####
"""

[[chunk.placements]]
chunks = ["2x2_room"]
id = "A"
`
	require.NoError(t, os.WriteFile(filepath.Join(layoutDir, "parent.toml"), []byte(parentTOML), 0o644))

	rawTOML := `[[Tiles]]
Name = "wall"
Description = "壁"
BlockPass = true
BlockView = true
[Tiles.SpriteRender]
SpriteSheetName = "sheet1"
SpriteKey = "wall"

[[Tiles]]
Name = "floor"
Description = "床"
BlockPass = false
BlockView = false
[Tiles.SpriteRender]
SpriteSheetName = "sheet1"
SpriteKey = "floor"
`
	rawPath := filepath.Join(tmpDir, "raw.toml")
	require.NoError(t, os.WriteFile(rawPath, []byte(rawTOML), 0o644))

	store, err := NewStore(rawPath)
	require.NoError(t, err)
	paletteStore, err := NewPaletteStore(paletteDir)
	require.NoError(t, err)
	layoutStore, err := NewLayoutStore([]string{layoutDir, chunkDir})
	require.NoError(t, err)

	server := NewServer(store,
		WithPaletteStore(paletteStore),
		WithLayoutStore(layoutStore),
	)
	server.sprites["sheet1"] = map[string]spriteFrame{
		"wall":  {X: 0, Y: 0, W: 32, H: 32},
		"floor": {X: 32, Y: 0, W: 32, H: 32},
	}
	server.sheetSizes["sheet1"] = asepriteSize{W: 256, H: 256}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /layouts/{dir}/{file}/{chunk}/preview", server.handleLayoutPreview)
	mux.HandleFunc("POST /layouts/{dir}/{file}/{chunk}/preview", server.handleLayoutPreview)

	t.Run("GETでplacementsが展開される", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest("GET", "/layouts/layouts/parent/4x4_parent/preview", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		// 4列のグリッドになっている
		assert.Contains(t, body, "grid-template-columns:repeat(4, 32px)")
		// 展開後は@が残っていない（floorに置換されている）
		assert.Contains(t, body, "floor")
	})

	t.Run("POSTでもplacementsが展開される", func(t *testing.T) {
		t.Parallel()
		// 同じマップをPOSTで送る
		mapContent := "####\n#@@#\n#@A#\n####"
		form := strings.NewReader("map_content=" + url.QueryEscape(mapContent))
		req := httptest.NewRequest("POST", "/layouts/layouts/parent/4x4_parent/preview", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		assert.Contains(t, body, "grid-template-columns:repeat(4, 32px)")
		assert.Contains(t, body, "floor")
	})
}

func TestBuildPreviewDataFromCells(t *testing.T) {
	t.Parallel()
	server := setupLayoutTest(t)

	palette := &maptemplate.Palette{
		ID:      "test",
		Terrain: map[string]string{"#": "wall", ".": "floor"},
		Props:   map[string]maptemplate.PaletteEntry{"+": {ID: "door", Tile: "floor"}},
		NPCs:    map[string]maptemplate.PaletteEntry{"M": {ID: "boss", Tile: "floor"}},
	}

	cells := maptemplate.ResolveMapCells("##\n.M", palette)
	data := server.buildPreviewDataFromCells(cells)

	assert.Equal(t, 2, data.Cols)
	assert.Equal(t, 4, len(data.Cells))

	// 1行目: # #
	assert.Equal(t, "wall", data.Cells[0].Terrain)
	assert.Equal(t, 1, len(data.Cells[0].Sprites))
	assert.Contains(t, data.Cells[0].Sprites[0].Style, "/sprites/sheet1")

	// 2行目: . M
	assert.Equal(t, "floor", data.Cells[2].Terrain)

	assert.Equal(t, "floor", data.Cells[3].Terrain)
	assert.Equal(t, "boss", data.Cells[3].NPC)
	assert.Equal(t, 2, len(data.Cells[3].Sprites))
}

func TestHandleLayoutCreate(t *testing.T) {
	t.Parallel()
	server := setupLayoutTest(t)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /layouts/new", server.handleLayoutCreate)

	t.Run("新規チャンクを作成できる", func(t *testing.T) {
		t.Parallel()
		formData := url.Values{}
		formData.Set("dir", "layouts")
		formData.Set("file", "new_file")
		formData.Set("name", "5x3_test_room")
		req := httptest.NewRequest("POST", "/layouts/new", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusSeeOther, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/layouts/layouts/new_file/5x3_test_room/edit")

		// 作成されたチャンクを確認する
		chunk, err := server.layoutStore.GetChunk("layouts", "new_file.toml", "5x3_test_room")
		require.NoError(t, err)
		assert.Equal(t, 100, chunk.Weight)
		lines := strings.Split(strings.TrimSpace(chunk.Map), "\n")
		assert.Equal(t, 3, len(lines))
		assert.Equal(t, 5, len(lines[0]))
	})

	t.Run("不正なチャンク名はエラー", func(t *testing.T) {
		t.Parallel()
		formData := url.Values{}
		formData.Set("dir", "layouts")
		formData.Set("file", "bad")
		formData.Set("name", "invalid_name")
		req := httptest.NewRequest("POST", "/layouts/new", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("必須フィールドが空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		formData := url.Values{}
		formData.Set("dir", "")
		formData.Set("file", "")
		formData.Set("name", "")
		req := httptest.NewRequest("POST", "/layouts/new", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
