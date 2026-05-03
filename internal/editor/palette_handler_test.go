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

func TestParseEntryFields(t *testing.T) {
	t.Parallel()

	t.Run("正常なエントリを構築する", func(t *testing.T) {
		t.Parallel()
		form := url.Values{
			"prop_char[]":  {"+", "B"},
			"prop_value[]": {"door", "bonfire"},
			"prop_tile[]":  {"floor", "dirt"},
		}
		result := parseEntryFields(form, "prop")

		require.Len(t, result, 2)
		assert.Equal(t, maptemplate.PaletteEntry{ID: "door", Tile: "floor"}, result["+"])
		assert.Equal(t, maptemplate.PaletteEntry{ID: "bonfire", Tile: "dirt"}, result["B"])
	})

	t.Run("空文字のcharはスキップする", func(t *testing.T) {
		t.Parallel()
		form := url.Values{
			"prop_char[]":  {"", "B"},
			"prop_value[]": {"door", "bonfire"},
			"prop_tile[]":  {"floor", "floor"},
		}
		result := parseEntryFields(form, "prop")

		require.Len(t, result, 1)
		assert.Equal(t, "bonfire", result["B"].ID)
	})

	t.Run("空文字のvalueはスキップする", func(t *testing.T) {
		t.Parallel()
		form := url.Values{
			"prop_char[]":  {"+", "B"},
			"prop_value[]": {"", "bonfire"},
			"prop_tile[]":  {"floor", "floor"},
		}
		result := parseEntryFields(form, "prop")

		require.Len(t, result, 1)
		assert.Equal(t, "bonfire", result["B"].ID)
	})

	t.Run("tileが不足している場合は空文字になる", func(t *testing.T) {
		t.Parallel()
		form := url.Values{
			"prop_char[]":  {"+"},
			"prop_value[]": {"door"},
			// tile[] なし
		}
		result := parseEntryFields(form, "prop")

		require.Len(t, result, 1)
		assert.Equal(t, maptemplate.PaletteEntry{ID: "door", Tile: ""}, result["+"])
	})

	t.Run("フィールドがない場合は空マップを返す", func(t *testing.T) {
		t.Parallel()
		form := url.Values{}
		result := parseEntryFields(form, "prop")

		assert.Empty(t, result)
	})
}

func TestParsePaletteForm(t *testing.T) {
	t.Parallel()

	t.Run("全フィールドをパースする", func(t *testing.T) {
		t.Parallel()
		form := url.Values{
			"description":     {"テスト説明"},
			"terrain_char[]":  {"#", "."},
			"terrain_value[]": {"wall", "floor"},
			"prop_char[]":     {"+"},
			"prop_value[]":    {"door"},
			"prop_tile[]":     {"floor"},
			"npc_char[]":      {"M"},
			"npc_value[]":     {"boss"},
			"npc_tile[]":      {"floor"},
		}
		r := httptest.NewRequest(http.MethodPost, "/palettes/test_pal", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		require.NoError(t, r.ParseForm())

		p := parsePaletteForm(r, "test_pal")

		assert.Equal(t, "test_pal", p.ID)
		assert.Equal(t, "テスト説明", p.Description)
		assert.Equal(t, "wall", p.Terrain["#"])
		assert.Equal(t, "floor", p.Terrain["."])
		assert.Equal(t, maptemplate.PaletteEntry{ID: "door", Tile: "floor"}, p.Props["+"])
		assert.Equal(t, maptemplate.PaletteEntry{ID: "boss", Tile: "floor"}, p.NPCs["M"])
	})

	t.Run("説明が空の場合はIDがセットされる", func(t *testing.T) {
		t.Parallel()
		form := url.Values{
			"description": {""},
		}
		r := httptest.NewRequest(http.MethodPost, "/palettes/mypal", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		require.NoError(t, r.ParseForm())

		p := parsePaletteForm(r, "mypal")

		assert.Equal(t, "mypal", p.Description)
	})

	t.Run("行を削除してもギャップなくパースされる", func(t *testing.T) {
		t.Parallel()
		// 3行あったうち中間を削除して2行になったケースを模倣する
		form := url.Values{
			"terrain_char[]":  {"#", "."},
			"terrain_value[]": {"wall", "floor"},
		}
		r := httptest.NewRequest(http.MethodPost, "/palettes/test", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		require.NoError(t, r.ParseForm())

		p := parsePaletteForm(r, "test")

		require.Len(t, p.Terrain, 2)
		assert.Equal(t, "wall", p.Terrain["#"])
		assert.Equal(t, "floor", p.Terrain["."])
	})
}

func TestSortedEntryMappings(t *testing.T) {
	t.Parallel()

	m := map[string]maptemplate.PaletteEntry{
		"C": {ID: "chair", Tile: "floor"},
		"A": {ID: "apple", Tile: "dirt"},
		"B": {ID: "bench", Tile: "floor"},
	}
	result := sortedEntryMappings(m)

	require.Len(t, result, 3)
	assert.Equal(t, "A", result[0].Char)
	assert.Equal(t, "apple", result[0].Value)
	assert.Equal(t, "dirt", result[0].Tile)
	assert.Equal(t, "B", result[1].Char)
	assert.Equal(t, "C", result[2].Char)
}

func setupPaletteTest(t *testing.T) *Server {
	t.Helper()
	tmpDir := t.TempDir()
	paletteDir := filepath.Join(tmpDir, "palettes")
	require.NoError(t, os.MkdirAll(paletteDir, 0o755))

	paletteTOML := `[palette]
id = "pal1"
description = "テスト"
[palette.terrain]
"#" = "wall"
"." = "floor"
[palette.props]
"+" = { id = "door", tile = "floor" }
[palette.npcs]
"M" = { id = "boss", tile = "floor" }
`
	require.NoError(t, os.WriteFile(filepath.Join(paletteDir, "pal1.toml"), []byte(paletteTOML), 0o644))

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

	return NewServer(store, WithPaletteStore(paletteStore))
}

func TestHandlePalettes(t *testing.T) {
	t.Parallel()
	srv := setupPaletteTest(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/palettes", nil)
	srv.handlePalettes(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "pal1")
	assert.Contains(t, body, "1 palettes")
}

func TestHandlePaletteEdit(t *testing.T) {
	t.Parallel()
	srv := setupPaletteTest(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/palettes/pal1/edit", nil)
	r.SetPathValue("id", "pal1")
	srv.handlePaletteEdit(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "pal1")
	assert.Contains(t, body, "wall")
	assert.Contains(t, body, "door")
	assert.Contains(t, body, "boss")
}

func TestHandlePaletteEdit_NotFound(t *testing.T) {
	t.Parallel()
	srv := setupPaletteTest(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/palettes/nonexistent/edit", nil)
	r.SetPathValue("id", "nonexistent")
	srv.handlePaletteEdit(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandlePaletteCreate(t *testing.T) {
	t.Parallel()
	srv := setupPaletteTest(t)

	form := url.Values{"id": {"new_pal"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/palettes/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handlePaletteCreate(w, r)

	// 空パレットでも保存自体は成功する
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandlePaletteCreate_EmptyID(t *testing.T) {
	t.Parallel()
	srv := setupPaletteTest(t)

	form := url.Values{"id": {""}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/palettes/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handlePaletteCreate(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandlePaletteUpdate(t *testing.T) {
	t.Parallel()
	srv := setupPaletteTest(t)

	form := url.Values{
		"description":     {"更新後の説明"},
		"terrain_char[]":  {"#"},
		"terrain_value[]": {"wall"},
		"prop_char[]":     {"D"},
		"prop_value[]":    {"desk"},
		"prop_tile[]":     {"floor"},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/palettes/pal1", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("id", "pal1")
	srv.handlePaletteUpdate(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	// 更新されたパレットを確認する
	p, err := srv.paletteStore.Get("pal1")
	require.NoError(t, err)
	assert.Equal(t, "更新後の説明", p.Description)
	assert.Equal(t, "wall", p.Terrain["#"])
	assert.Equal(t, maptemplate.PaletteEntry{ID: "desk", Tile: "floor"}, p.Props["D"])
	assert.Empty(t, p.NPCs)
}

func TestHandlePaletteDelete(t *testing.T) {
	t.Parallel()
	srv := setupPaletteTest(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/palettes/pal1", nil)
	r.SetPathValue("id", "pal1")
	srv.handlePaletteDelete(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	// 削除されたことを確認する
	_, err := srv.paletteStore.Get("pal1")
	assert.Error(t, err)
}
