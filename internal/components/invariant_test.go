package components

import (
	"encoding/json"
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupComponents(t *testing.T) (*ecs.World, *Components) {
	t.Helper()
	world := ecs.NewWorld()
	c := &Components{}
	require.NoError(t, c.InitializeComponents(world))
	return world, c
}

func TestPred_Has(t *testing.T) {
	t.Parallel()
	manager, c := setupComponents(t)

	entity := manager.NewEntity()
	c.Melee.Add(entity, &Melee{})

	has := Has{"Melee", c.Melee}
	assert.True(t, has.Eval(entity))
	assert.Equal(t, "Melee", has.String())

	hasFail := Has{"Wearable", c.Wearable}
	assert.False(t, hasFail.Eval(entity))
}

func TestPred_Or(t *testing.T) {
	t.Parallel()
	manager, c := setupComponents(t)

	entity := manager.NewEntity()
	c.Melee.Add(entity, &Melee{})

	or := Or{Has{"Melee", c.Melee}, Has{"Fire", c.Fire}}
	assert.True(t, or.Eval(entity))
	assert.Equal(t, "(Melee OR Fire)", or.String())

	orFail := Or{Has{"Wearable", c.Wearable}, Has{"Fire", c.Fire}}
	assert.False(t, orFail.Eval(entity))
}

func TestPred_And(t *testing.T) {
	t.Parallel()
	manager, c := setupComponents(t)

	entity := manager.NewEntity()
	c.Melee.Add(entity, &Melee{})
	c.Fire.Add(entity, &Fire{})

	and := And{Has{"Melee", c.Melee}, Has{"Fire", c.Fire}}
	assert.True(t, and.Eval(entity))
	assert.Equal(t, "(Melee AND Fire)", and.String())

	andFail := And{Has{"Melee", c.Melee}, Has{"Wearable", c.Wearable}}
	assert.False(t, andFail.Eval(entity))
}

func TestPred_Not(t *testing.T) {
	t.Parallel()
	manager, c := setupComponents(t)

	entity := manager.NewEntity()
	c.Melee.Add(entity, &Melee{})

	not := Not{Has{"Wearable", c.Wearable}}
	assert.True(t, not.Eval(entity))
	assert.Equal(t, "NOT Wearable", not.String())

	notFail := Not{Has{"Melee", c.Melee}}
	assert.False(t, notFail.Eval(entity))
}

func TestCategory(t *testing.T) {
	t.Parallel()

	t.Run("武器カテゴリ", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Melee.Add(entity, &Melee{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "武器", cat)
	})

	t.Run("防具カテゴリ", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Wearable.Add(entity, &Wearable{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "防具", cat)
	})

	t.Run("射撃武器カテゴリ", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Fire.Add(entity, &Fire{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "武器", cat)
	})

	t.Run("道具カテゴリは本を含む", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Book.Add(entity, &Book{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("道具カテゴリは消耗品を含む", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Consumable.Add(entity, &Consumable{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("道具カテゴリは素材を含む", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Material.Add(entity, &Material{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("道具カテゴリは弾薬を含む", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Ammo.Add(entity, &Ammo{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("道具カテゴリは置物を含む", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Prop.Add(entity, &Prop{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("どのカテゴリにも属さない", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Name.Add(entity, &Name{Name: "何か"})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.False(t, ok)
		assert.Empty(t, cat)
	})

	t.Run("アイテム種別: 素材", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Material.Add(entity, &Material{})
		cat, ok := c.CategoryOf(ItemTypeCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "素材", cat)
	})

	t.Run("アイテム種別: 近接武器", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Melee.Add(entity, &Melee{})
		cat, ok := c.CategoryOf(ItemTypeCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "近接武器", cat)
	})

	t.Run("アイテム種別: 射撃武器", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Fire.Add(entity, &Fire{})
		cat, ok := c.CategoryOf(ItemTypeCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "射撃武器", cat)
	})

	t.Run("アイテム種別: Fire+Meleeは射撃武器になる", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Fire.Add(entity, &Fire{})
		c.Melee.Add(entity, &Melee{})
		cat, ok := c.CategoryOf(ItemTypeCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "射撃武器", cat)
	})

	t.Run("アイテム種別: 防具", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Wearable.Add(entity, &Wearable{})
		cat, ok := c.CategoryOf(ItemTypeCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "防具", cat)
	})

	t.Run("フィールド観察: 自分", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Player.Add(entity, &Player{})
		c.Faction.Add(entity, &Faction{Kind: FactionAlly})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "自分", cat)
	})

	t.Run("フィールド観察: 敵", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Faction.Add(entity, &Faction{Kind: FactionEnemy})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "敵", cat)
	})

	t.Run("フィールド観察: NPC(味方)", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Faction.Add(entity, &Faction{Kind: FactionAlly})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "NPC", cat)
	})

	t.Run("フィールド観察: NPC(中立)", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Faction.Add(entity, &Faction{Kind: FactionNeutral})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "NPC", cat)
	})

	t.Run("フィールド観察: 置物", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Prop.Add(entity, &Prop{})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "置物", cat)
	})

	t.Run("フィールド観察: タイル", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Tile.Add(entity, &Tile{})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "タイル", cat)
	})

	t.Run("フィールド観察: Player+FactionAllyは自分になる", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Player.Add(entity, &Player{})
		c.Faction.Add(entity, &Faction{Kind: FactionAlly})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "自分", cat)
	})

	t.Run("CategoryはPredとして使える", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		cats := c.Categories()
		weapon := cats[InventoryCategoryKey][1]
		entity := manager.NewEntity()
		c.Melee.Add(entity, &Melee{})
		assert.True(t, weapon.Eval(entity))
		assert.Equal(t, "武器", weapon.String())
	})
}

func TestCategoryOfSpec(t *testing.T) {
	t.Parallel()
	_, c := setupComponents(t)

	tests := []struct {
		name string
		spec EntitySpec
		want string
	}{
		{"素材", EntitySpec{Material: &Material{}}, CategoryMaterial},
		{"弾薬", EntitySpec{Ammo: &Ammo{}}, CategoryAmmo},
		{"本", EntitySpec{Book: &Book{}}, CategoryBook},
		{"置物", EntitySpec{Prop: &Prop{}}, CategoryProp},
		{"消耗品", EntitySpec{Consumable: &Consumable{}}, CategoryConsumable},
		{"射撃武器", EntitySpec{Fire: &Fire{}}, CategoryFire},
		{"近接武器", EntitySpec{Melee: &Melee{}}, CategoryMelee},
		{"防具", EntitySpec{Wearable: &Wearable{}}, CategoryArmor},
		{"Fire+Meleeは射撃武器", EntitySpec{Fire: &Fire{}, Melee: &Melee{}}, CategoryFire},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cat, ok := c.CategoryOfSpec(ItemTypeCategoryKey, &tt.spec)
			assert.True(t, ok)
			assert.Equal(t, tt.want, cat)
		})
	}

	t.Run("該当なし", func(t *testing.T) {
		t.Parallel()
		spec := EntitySpec{Name: &Name{Name: "何か"}}
		cat, ok := c.CategoryOfSpec(ItemTypeCategoryKey, &spec)
		assert.False(t, ok)
		assert.Empty(t, cat)
	})
}

// categoryEntry はゴールデンテスト用のJSON構造体
type categoryEntry struct {
	Name string `json:"name"`
	Pred string `json:"pred"`
}

// categoriesToJSON はカテゴリ定義をJSON化する
func categoriesToJSON(cats map[CategoryGroupKey][]Category) ([]byte, error) {
	m := make(map[CategoryGroupKey][]categoryEntry, len(cats))
	for key, categories := range cats {
		entries := make([]categoryEntry, len(categories))
		for i, cat := range categories {
			entries[i] = categoryEntry{Name: cat.Name, Pred: cat.Pred.String()}
		}
		m[key] = entries
	}
	return json.MarshalIndent(m, "", "  ")
}

func TestCategoriesGolden(t *testing.T) {
	t.Parallel()
	_, c := setupComponents(t)

	got, err := categoriesToJSON(c.Categories())
	require.NoError(t, err)

	g := goldie.New(
		t,
		goldie.WithFixtureDir("testdata"),
		goldie.WithNameSuffix(".golden.json"),
		goldie.WithDiffEngine(goldie.ColoredDiff),
	)
	g.Assert(t, "categories", got)
}
