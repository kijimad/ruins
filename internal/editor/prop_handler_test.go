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

func TestHandleProps(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddProp(raw.PropRaw{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/props", nil)
	srv.handleProps(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "テスト")
}

func TestHandlePropCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {"新置物"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/props/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handlePropCreate(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)

	props := srv.store.Props()
	require.Len(t, props, 1)
	assert.Equal(t, "新置物", props[0].Name)
}

func TestHandlePropDelete(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddProp(raw.PropRaw{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/props/0/delete", nil)
	r.SetPathValue("index", "0")
	srv.handlePropDelete(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Empty(t, srv.store.Props())
}
