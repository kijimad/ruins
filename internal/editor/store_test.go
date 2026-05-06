package editor

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreSortOnSave(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t, []raw.Item{
		{Name: "素材"}, // なし → "Z"
		{Name: "銃剣", Weapon: &raw.Weapon{}, Melee: &raw.MeleeRaw{}, Fire: &raw.FireRaw{}}, // 武器+近接+射撃 → "ABC"
		{Name: "鎧", Wearable: &raw.Wearable{}},                                            // 防具 → "D"
		{Name: "剣", Weapon: &raw.Weapon{}, Melee: &raw.MeleeRaw{}},                        // 武器+近接 → "AB"
		{Name: "拳銃", Weapon: &raw.Weapon{}, Fire: &raw.FireRaw{}},                         // 武器+射撃 → "AC"
		{Name: "回復薬", Consumable: &raw.Consumable{}},                                      // 消費 → "E"
		{Name: "刀", Weapon: &raw.Weapon{}, Melee: &raw.MeleeRaw{}},                        // 武器+近接 → "AB"
	})

	items := store.Items()
	require.Len(t, items, 7)
	// AB: 武器+近接が名前順で隣接する
	assert.Equal(t, "刀", items[0].Name)
	assert.Equal(t, "剣", items[1].Name)
	// ABC: 武器+近接+射撃
	assert.Equal(t, "銃剣", items[2].Name)
	// AC: 武器+射撃
	assert.Equal(t, "拳銃", items[3].Name)
	// D: 防具
	assert.Equal(t, "鎧", items[4].Name)
	// E: 消費
	assert.Equal(t, "回復薬", items[5].Name)
	// Z: なし
	assert.Equal(t, "素材", items[6].Name)
}

func TestStoreMemberCRUD(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t, []raw.Item{})

	// 初期状態は空
	assert.Empty(t, store.Members())

	// AddMember
	require.NoError(t, store.AddMember(raw.Member{Name: "テスト"}))
	members := store.Members()
	require.Len(t, members, 1)
	assert.Equal(t, "テスト", members[0].Name)

	// Member
	m, err := store.Member(0)
	require.NoError(t, err)
	assert.Equal(t, "テスト", m.Name)

	// UpdateMember
	m.Name = "更新済み"
	require.NoError(t, store.UpdateMember(0, m))
	updated, err := store.Member(0)
	require.NoError(t, err)
	assert.Equal(t, "更新済み", updated.Name)

	// DeleteMember
	require.NoError(t, store.DeleteMember(0))
	assert.Empty(t, store.Members())

	// 範囲外アクセスでエラーになる
	_, err = store.Member(0)
	assert.Error(t, err)
}

func TestStoreRecipeCRUD(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t, []raw.Item{})

	// 初期状態は空
	assert.Empty(t, store.Recipes())

	// AddRecipe
	require.NoError(t, store.AddRecipe(raw.Recipe{Name: "テスト"}))
	recipes := store.Recipes()
	require.Len(t, recipes, 1)
	assert.Equal(t, "テスト", recipes[0].Name)

	// Recipe
	r, err := store.Recipe(0)
	require.NoError(t, err)
	assert.Equal(t, "テスト", r.Name)

	// UpdateRecipe
	r.Name = "更新済み"
	require.NoError(t, store.UpdateRecipe(0, r))
	updated, err := store.Recipe(0)
	require.NoError(t, err)
	assert.Equal(t, "更新済み", updated.Name)

	// DeleteRecipe
	require.NoError(t, store.DeleteRecipe(0))
	assert.Empty(t, store.Recipes())

	// 範囲外アクセスでエラーになる
	_, err = store.Recipe(0)
	assert.Error(t, err)
}

func TestStoreCommandTableCRUD(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t, []raw.Item{})

	// 初期状態は空
	assert.Empty(t, store.CommandTables())

	// AddCommandTable
	require.NoError(t, store.AddCommandTable(raw.CommandTable{Name: "テスト"}))
	tables := store.CommandTables()
	require.Len(t, tables, 1)
	assert.Equal(t, "テスト", tables[0].Name)

	// DeleteCommandTable
	require.NoError(t, store.DeleteCommandTable(0))
	assert.Empty(t, store.CommandTables())
}

func TestStoreProfessionCRUD(t *testing.T) {
	t.Parallel()
	store := setupTestStore(t, []raw.Item{})

	// 初期状態は空
	assert.Empty(t, store.Professions())

	// AddProfession
	require.NoError(t, store.AddProfession(raw.Profession{ID: "test", Name: "テスト"}))
	profs := store.Professions()
	require.Len(t, profs, 1)
	assert.Equal(t, "テスト", profs[0].Name)

	// DeleteProfession
	require.NoError(t, store.DeleteProfession(0))
	assert.Empty(t, store.Professions())
}
