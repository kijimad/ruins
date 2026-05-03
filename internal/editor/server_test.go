package editor

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestStore はテスト用の最小限のraw.tomlを作成してStoreを返す
func setupTestStore(t *testing.T, items []raw.Item) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.toml")

	raws := raw.Raws{Items: items}
	f, err := os.Create(path)
	require.NoError(t, err)

	encoder := toml.NewEncoder(f)
	require.NoError(t, encoder.Encode(raws))
	require.NoError(t, f.Close())

	store, err := NewStore(path)
	require.NoError(t, err)
	return store
}

func newTestServer(t *testing.T, items []raw.Item) *Server {
	t.Helper()
	store := setupTestStore(t, items)
	return NewServer(store)
}

// createTestPNG は指定サイズのテスト用PNG画像を作成する
func createTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// 左上のセルだけ色を塗る
	for y := range 32 {
		for x := range 32 {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestHandleDashboard(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.handleDashboard(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Items")
	assert.Contains(t, body, "Palettes")
	assert.Contains(t, body, "Layouts")
}

func TestHandleSpriteKeys(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t, []raw.Item{})
	srv := NewServer(store)
	// スプライトデータを直接設定する
	srv.sprites["testsheet"] = map[string]spriteFrame{
		"sword": {X: 0, Y: 0, W: 16, H: 16},
		"axe":   {X: 16, Y: 0, W: 16, H: 16},
	}
	srv.sheetSizes["testsheet"] = asepriteSize{W: 64, H: 64}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sprites/testsheet/keys", nil)
	r.SetPathValue("name", "testsheet")
	srv.handleSpriteKeys(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "sword")
	assert.Contains(t, body, "axe")
}

func TestHandleSpriteKeys_NotFound(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sprites/unknown/keys", nil)
	r.SetPathValue("name", "unknown")
	srv.handleSpriteKeys(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
