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

func TestParseItemForm_BasicFields(t *testing.T) {
	t.Parallel()
	form := url.Values{
		"name":               {"テスト剣"},
		"description":        {"説明文"},
		"sprite_sheet_name":  {"items"},
		"sprite_key":         {"sword"},
		"value":              {"100"},
		"weight":             {"2.5"},
		"inflicts_damage":    {"10"},
		"provides_nutrition": {""},
	}
	r := httptest.NewRequest(http.MethodPost, "/items/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	item := parseItemForm(r, raw.Item{})

	assert.Equal(t, "テスト剣", item.Name)
	assert.Equal(t, "説明文", item.Description)
	assert.Equal(t, "items", item.SpriteSheetName)
	assert.Equal(t, "sword", item.SpriteKey)
	assert.Equal(t, 100, item.Value)
	assert.InDelta(t, 2.5, *item.Weight, 0.001)
	assert.Equal(t, 10, *item.InflictsDamage)
	assert.Nil(t, item.ProvidesNutrition)
}

func TestParseItemForm_Melee(t *testing.T) {
	t.Parallel()
	form := url.Values{
		"name":                  {"剣"},
		"has_melee":             {"on"},
		"melee_accuracy":        {"80"},
		"melee_damage":          {"15"},
		"melee_attack_count":    {"1"},
		"melee_cost":            {"3"},
		"melee_element":         {"NONE"},
		"melee_attack_category": {"BLADE"},
		"melee_target_group":    {"ENEMY"},
		"melee_target_num":      {"SINGLE"},
	}
	r := httptest.NewRequest(http.MethodPost, "/items/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	item := parseItemForm(r, raw.Item{})

	require.NotNil(t, item.Melee)
	assert.Equal(t, 80, item.Melee.Accuracy)
	assert.Equal(t, 15, item.Melee.Damage)
	assert.Equal(t, 1, item.Melee.AttackCount)
	assert.Equal(t, 3, item.Melee.Cost)
	assert.Equal(t, "BLADE", item.Melee.AttackCategory)
	assert.NotNil(t, item.Weapon, "近接攻撃を有効にするとWeaponも設定される")
}

func TestParseItemForm_MeleeUnchecked(t *testing.T) {
	t.Parallel()
	existing := raw.Item{
		Name:  "剣",
		Melee: &raw.MeleeRaw{Accuracy: 80},
	}
	form := url.Values{
		"name": {"剣"},
		// has_melee が送信されない = チェックボックスOFF
	}
	r := httptest.NewRequest(http.MethodPost, "/items/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	item := parseItemForm(r, existing)

	assert.Nil(t, item.Melee, "チェックボックスOFFでMeleeがnilになる")
}

func TestParseItemForm_Fire(t *testing.T) {
	t.Parallel()
	form := url.Values{
		"name":               {"銃"},
		"has_fire":           {"on"},
		"fire_accuracy":      {"70"},
		"fire_damage":        {"20"},
		"fire_attack_count":  {"1"},
		"fire_cost":          {"5"},
		"fire_magazine_size": {"6"},
		"fire_reload_effort": {"2"},
		"fire_ammo_tag":      {"9mm"},
		"fire_target_group":  {"ENEMY"},
		"fire_target_num":    {"SINGLE"},
	}
	r := httptest.NewRequest(http.MethodPost, "/items/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	item := parseItemForm(r, raw.Item{})

	require.NotNil(t, item.Fire)
	assert.Equal(t, 70, item.Fire.Accuracy)
	assert.Equal(t, 20, item.Fire.Damage)
	assert.Equal(t, 6, item.Fire.MagazineSize)
	assert.Equal(t, "9mm", item.Fire.AmmoTag)
	assert.NotNil(t, item.Weapon)
}

func TestParseItemForm_Consumable(t *testing.T) {
	t.Parallel()
	form := url.Values{
		"name":                    {"回復薬"},
		"has_consumable":          {"on"},
		"consumable_usable_scene": {"ANY"},
		"consumable_target_group": {"ALLY"},
		"consumable_target_num":   {"SINGLE"},
	}
	r := httptest.NewRequest(http.MethodPost, "/items/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	item := parseItemForm(r, raw.Item{})

	require.NotNil(t, item.Consumable)
	assert.Equal(t, "ANY", item.Consumable.UsableScene)
	assert.Equal(t, "ALLY", item.Consumable.TargetGroup)
	assert.Equal(t, "SINGLE", item.Consumable.TargetNum)
}

func TestParseItemForm_Wearable(t *testing.T) {
	t.Parallel()
	form := url.Values{
		"name":                        {"鎧"},
		"has_wearable":                {"on"},
		"wearable_defense":            {"10"},
		"wearable_equipment_category": {"BODY"},
		"wearable_insulation_cold":    {"3"},
		"wearable_insulation_heat":    {"1"},
	}
	r := httptest.NewRequest(http.MethodPost, "/items/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	item := parseItemForm(r, raw.Item{})

	require.NotNil(t, item.Wearable)
	assert.Equal(t, 10, item.Wearable.Defense)
	assert.Equal(t, "BODY", item.Wearable.EquipmentCategory)
	assert.Equal(t, 3, item.Wearable.InsulationCold)
	assert.Equal(t, 1, item.Wearable.InsulationHeat)
}

func TestParseItemForm_Stackable(t *testing.T) {
	t.Parallel()
	form := url.Values{
		"name":      {"弾薬"},
		"stackable": {"on"},
	}
	r := httptest.NewRequest(http.MethodPost, "/items/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	require.NoError(t, r.ParseForm())

	item := parseItemForm(r, raw.Item{})
	require.NotNil(t, item.Stackable)
	assert.True(t, *item.Stackable)
}

func TestHandleIndex(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{
		{Name: "剣", Description: "鋭い剣"},
		{Name: "盾", Description: "頑丈な盾"},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/items", nil)
	srv.handleIndex(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "剣")
	assert.Contains(t, body, "盾")
	assert.Contains(t, body, "2 items")
}

func TestHandleItemRow(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{
		{Name: "剣", Description: "鋭い剣"},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/items/0", nil)
	r.SetPathValue("index", "0")
	srv.handleItemRow(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "剣")
}

func TestHandleItemRow_NotFound(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/items/99", nil)
	r.SetPathValue("index", "99")
	srv.handleItemRow(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleItemEdit(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{
		{Name: "剣", Melee: &raw.MeleeRaw{Accuracy: 80, Damage: 10}},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/items/0/edit", nil)
	r.SetPathValue("index", "0")
	srv.handleItemEdit(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "剣")
	assert.Contains(t, body, "80")
}

func TestHandleItemUpdate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{
		{Name: "剣"},
	})

	form := url.Values{"name": {"改良剣"}, "description": {"改良された剣"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/items/0", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("index", "0")
	srv.handleItemUpdate(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "改良剣")

	// Storeにも反映されていることを確認する
	item, err := srv.store.Item(0)
	require.NoError(t, err)
	assert.Equal(t, "改良剣", item.Name)
}

func TestHandleItemCreate(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {"新アイテム"}, "description": {"新しいアイテム"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/items/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleItemCreate(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	items := srv.store.Items()
	require.Len(t, items, 1)
	assert.Equal(t, "新アイテム", items[0].Name)
}

func TestHandleItemCreate_EmptyName(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{})

	form := url.Values{"name": {""}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/items/new", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.handleItemCreate(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, srv.store.Items())
}

func TestHandleItemDelete(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, []raw.Item{
		{Name: "剣"},
		{Name: "盾"},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/items/0", nil)
	r.SetPathValue("index", "0")
	srv.handleItemDelete(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	items := srv.store.Items()
	require.Len(t, items, 1)
	assert.Equal(t, "盾", items[0].Name)
}
