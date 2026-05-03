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

func TestHandleCommandTables(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddCommandTable(raw.CommandTable{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/command-tables", nil)
	srv.handleCommandTables(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleCommandTableCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {"新コマンド"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/command-tables/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleCommandTableCreate(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	tables := srv.store.CommandTables()
	require.Len(t, tables, 1)
	assert.Equal(t, "新コマンド", tables[0].Name)
}

func TestHandleCommandTableDelete(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddCommandTable(raw.CommandTable{Name: "削除対象"}))
	require.NoError(t, srv.store.AddCommandTable(raw.CommandTable{Name: "残る"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/command-tables/0", nil)
	r.SetPathValue("index", "0")
	srv.handleCommandTableDelete(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	tables := srv.store.CommandTables()
	require.Len(t, tables, 1)
	assert.Equal(t, "残る", tables[0].Name)
}

func TestHandleDropTables(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddDropTable(raw.DropTable{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/drop-tables", nil)
	srv.handleDropTables(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleDropTableCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {"新ドロップ"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/drop-tables/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleDropTableCreate(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	tables := srv.store.DropTables()
	require.Len(t, tables, 1)
	assert.Equal(t, "新ドロップ", tables[0].Name)
}

func TestHandleDropTableDelete(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddDropTable(raw.DropTable{Name: "削除対象"}))
	require.NoError(t, srv.store.AddDropTable(raw.DropTable{Name: "残る"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/drop-tables/0", nil)
	r.SetPathValue("index", "0")
	srv.handleDropTableDelete(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	tables := srv.store.DropTables()
	require.Len(t, tables, 1)
	assert.Equal(t, "残る", tables[0].Name)
}

func TestHandleItemTables(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddItemTable(raw.ItemTable{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/item-tables", nil)
	srv.handleItemTables(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleItemTableCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {"新アイテムテーブル"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/item-tables/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleItemTableCreate(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	tables := srv.store.ItemTables()
	require.Len(t, tables, 1)
	assert.Equal(t, "新アイテムテーブル", tables[0].Name)
}

func TestHandleEnemyTables(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddEnemyTable(raw.EnemyTable{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/enemy-tables", nil)
	srv.handleEnemyTables(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleEnemyTableCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {"新エネミーテーブル"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/enemy-tables/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleEnemyTableCreate(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	tables := srv.store.EnemyTables()
	require.Len(t, tables, 1)
	assert.Equal(t, "新エネミーテーブル", tables[0].Name)
}
