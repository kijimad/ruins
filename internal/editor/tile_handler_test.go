package editor

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleTiles(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddTile(raw.TileRaw{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tiles", nil)
	srv.handleTiles(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "テスト")
}

func TestHandleTileCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {"新タイル"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/tiles/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleTileCreate(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)

	tiles := srv.store.Tiles()
	require.Len(t, tiles, 1)
	assert.Equal(t, "新タイル", tiles[0].Name)
}

func TestHandleTileDelete(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddTile(raw.TileRaw{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/tiles/0/delete", nil)
	r.SetPathValue("index", "0")
	srv.handleTileDelete(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Empty(t, srv.store.Tiles())
}
