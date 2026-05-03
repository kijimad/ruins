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

func TestParseMemberForm(t *testing.T) {
	t.Parallel()
	form := url.Values{
		"name":              {"テスト戦士"},
		"sprite_sheet_name": {"characters"},
		"faction_type":      {"ENEMY"},
		"vitality":          {"100"},
		"strength":          {"50"},
		"is_boss":           {"on"},
	}
	r := httptest.NewRequest(http.MethodPost, "/members/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	member := parseMemberForm(r, raw.Member{})

	assert.Equal(t, "テスト戦士", member.Name)
	assert.Equal(t, "characters", member.SpriteSheetName)
	assert.Equal(t, "ENEMY", member.FactionType)
	assert.Equal(t, 100, member.Abilities.Vitality)
	assert.Equal(t, 50, member.Abilities.Strength)
	assert.True(t, member.IsBoss)
}

func TestParseMemberForm_WithDialog(t *testing.T) {
	t.Parallel()
	form := url.Values{
		"name":               {"NPC"},
		"has_dialog":         {"on"},
		"dialog_message_key": {"greeting"},
	}
	r := httptest.NewRequest(http.MethodPost, "/members/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	member := parseMemberForm(r, raw.Member{})

	require.NotNil(t, member.Dialog)
	assert.Equal(t, "greeting", member.Dialog.MessageKey)
}

func TestHandleMembers(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddMember(raw.Member{Name: "テスト"}))
	require.NoError(t, srv.store.AddMember(raw.Member{Name: "仲間"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/members", nil)
	srv.handleMembers(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "テスト")
	assert.Contains(t, body, "仲間")
}

func TestHandleMemberEdit(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddMember(raw.Member{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/members/0/edit", nil)
	r.SetPathValue("index", "0")
	srv.handleMemberEdit(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "テスト")
}

func TestHandleMemberUpdate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddMember(raw.Member{Name: "テスト"}))

	form := url.Values{"name": {"更新済み"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/members/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("index", "0")
	srv.handleMemberUpdate(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	// Storeにも反映されていることを確認する
	member, err := srv.store.Member(0)
	require.NoError(t, err)
	assert.Equal(t, "更新済み", member.Name)
}

func TestHandleMemberCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {"テスト"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/members/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleMemberCreate(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	members := srv.store.Members()
	require.Len(t, members, 1)
	assert.Equal(t, "テスト", members[0].Name)
}

func TestHandleMemberCreate_EmptyName(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {""}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/members/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleMemberCreate(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, srv.store.Members())
}

func TestHandleMemberDelete(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddMember(raw.Member{Name: "テスト"}))
	require.NoError(t, srv.store.AddMember(raw.Member{Name: "仲間"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/members/0", nil)
	r.SetPathValue("index", "0")
	srv.handleMemberDelete(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	members := srv.store.Members()
	require.Len(t, members, 1)
	assert.Equal(t, "仲間", members[0].Name)
}
