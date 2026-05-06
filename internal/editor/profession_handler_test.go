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

func TestHandleProfessions(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddProfession(raw.Profession{ID: "test", Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/professions", nil)
	srv.handleProfessions(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "テスト")
}

func TestHandleProfessionCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"id": {"new_prof"}, "name": {"新職業"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/professions/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleProfessionCreate(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)

	profs := srv.store.Professions()
	require.Len(t, profs, 1)
	assert.Equal(t, "new_prof", profs[0].ID)
	assert.Equal(t, "新職業", profs[0].Name)
}

func TestHandleProfessionDelete(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddProfession(raw.Profession{ID: "test", Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/professions/0/delete", nil)
	r.SetPathValue("index", "0")
	srv.handleProfessionDelete(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Empty(t, srv.store.Professions())
}
