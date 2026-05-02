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
"+" = "door"
[palette.npcs]
"M" = "boss"
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
Str = 10
Vit = 10
Dex = 10
Int = 10
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

	// 編集フォームにhx-trigger="load"付きプレビューコンテナがある
	assert.Contains(t, body, `hx-trigger="load"`)
	assert.Contains(t, body, "/layouts/layouts/test/3x3_test/preview")
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

func TestBuildPreviewData(t *testing.T) {
	t.Parallel()
	server := setupLayoutTest(t)

	palette := &maptemplate.Palette{
		ID:      "test",
		Terrain: map[string]string{"#": "wall", ".": "floor"},
		Props:   map[string]string{"+": "door"},
		NPCs:    map[string]string{"M": "boss"},
	}

	data := server.buildPreviewData("##\n.M", palette)

	assert.Equal(t, 2, data.Cols)
	assert.Equal(t, 4, len(data.Cells))

	// 1行目: # #
	assert.Equal(t, "#", data.Cells[0].Char)
	assert.Equal(t, "wall", data.Cells[0].Terrain)
	assert.Equal(t, 1, len(data.Cells[0].Sprites))
	assert.Contains(t, data.Cells[0].Sprites[0].Style, "/sprites/sheet1")

	// 2行目: . M
	assert.Equal(t, ".", data.Cells[2].Char)
	assert.Equal(t, "floor", data.Cells[2].Terrain)

	assert.Equal(t, "M", data.Cells[3].Char)
	assert.Equal(t, "boss", data.Cells[3].NPC)
	assert.Equal(t, 1, len(data.Cells[3].Sprites))
}
