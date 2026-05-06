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

func TestParseRecipeForm(t *testing.T) {
	t.Parallel()
	form := url.Values{
		"name":           {"合成レシピ"},
		"input_name_0":   {"素材A"},
		"input_amount_0": {"3"},
		"input_name_1":   {"素材B"},
		"input_amount_1": {"5"},
	}
	r := httptest.NewRequest(http.MethodPost, "/recipes/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	recipe := parseRecipeForm(r)

	assert.Equal(t, "合成レシピ", recipe.Name)
	require.Len(t, recipe.Inputs, 2)
	assert.Equal(t, "素材A", recipe.Inputs[0].Name)
	assert.Equal(t, 3, recipe.Inputs[0].Amount)
	assert.Equal(t, "素材B", recipe.Inputs[1].Name)
	assert.Equal(t, 5, recipe.Inputs[1].Amount)
}

func TestHandleRecipes(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/recipes", nil)
	srv.handleRecipes(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleRecipeCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {"新レシピ"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/recipes/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleRecipeCreate(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)

	recipes := srv.store.Recipes()
	require.Len(t, recipes, 1)
	assert.Equal(t, "新レシピ", recipes[0].Name)
}

func TestHandleRecipeDelete(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})
	require.NoError(t, srv.store.AddRecipe(raw.Recipe{Name: "テスト"}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/recipes/0/delete", nil)
	r.SetPathValue("index", "0")
	srv.handleRecipeDelete(w, r)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Empty(t, srv.store.Recipes())
}
