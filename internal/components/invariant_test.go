package components

import (
	"encoding/json"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func setupComponents(t *testing.T) (*ecs.Manager, *Components) {
	t.Helper()
	manager := ecs.NewManager()
	c := &Components{}
	require.NoError(t, c.InitializeComponents(manager))
	return manager, c
}

func TestPred_Has(t *testing.T) {
	t.Parallel()
	manager, c := setupComponents(t)

	entity := manager.NewEntity()
	entity.AddComponent(c.Value, &Value{Value: 100})

	has := Has{"Value", c.Value}
	assert.True(t, has.Eval(entity))
	assert.Equal(t, "Value", has.String())

	notHas := Has{"Melee", c.Melee}
	assert.False(t, notHas.Eval(entity))
}

func TestPred_And(t *testing.T) {
	t.Parallel()
	manager, c := setupComponents(t)

	entity := manager.NewEntity()
	entity.AddComponent(c.Value, &Value{Value: 100})
	entity.AddComponent(c.Name, &Name{Name: "test"})

	and := And{Has{"Value", c.Value}, Has{"Name", c.Name}}
	assert.True(t, and.Eval(entity))
	assert.Equal(t, "(Value AND Name)", and.String())

	andFail := And{Has{"Value", c.Value}, Has{"Melee", c.Melee}}
	assert.False(t, andFail.Eval(entity))
}

func TestPred_Or(t *testing.T) {
	t.Parallel()
	manager, c := setupComponents(t)

	entity := manager.NewEntity()
	entity.AddComponent(c.Melee, &Melee{})

	or := Or{Has{"Melee", c.Melee}, Has{"Fire", c.Fire}}
	assert.True(t, or.Eval(entity))
	assert.Equal(t, "(Melee OR Fire)", or.String())

	orFail := Or{Has{"Value", c.Value}, Has{"Fire", c.Fire}}
	assert.False(t, orFail.Eval(entity))
}

func TestPred_Not(t *testing.T) {
	t.Parallel()
	manager, c := setupComponents(t)

	entity := manager.NewEntity()
	entity.AddComponent(c.Melee, &Melee{})

	not := Not{Has{"Wearable", c.Wearable}}
	assert.True(t, not.Eval(entity))
	assert.Equal(t, "NOT Wearable", not.String())

	notFail := Not{Has{"Melee", c.Melee}}
	assert.False(t, notFail.Eval(entity))
}

func TestCategory(t *testing.T) {
	t.Parallel()
	manager, c := setupComponents(t)

	t.Run("武器カテゴリ", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Melee, &Melee{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "武器", cat)
	})

	t.Run("防具カテゴリ", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Wearable, &Wearable{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "防具", cat)
	})

	t.Run("射撃武器カテゴリ", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Fire, &Fire{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "武器", cat)
	})

	t.Run("道具カテゴリは本を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Book, &Book{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("道具カテゴリは消耗品を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Consumable, &Consumable{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("道具カテゴリは素材を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Material, &Material{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("道具カテゴリは弾薬を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Ammo, &Ammo{})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("道具カテゴリは素材を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Material, nil)
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("道具カテゴリは置物を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Prop, nil)
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "道具", cat)
	})

	t.Run("どのカテゴリにも属さない", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Name, &Name{Name: "何か"})
		cat, ok := c.CategoryOf(InventoryCategoryKey, entity)
		assert.False(t, ok)
		assert.Equal(t, "", cat)
	})

	t.Run("アイテム種別: 素材", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Material, nil)
		cat, ok := c.CategoryOf(ItemTypeCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "素材", cat)
	})

	t.Run("アイテム種別: 近接武器", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Melee, &Melee{})
		cat, ok := c.CategoryOf(ItemTypeCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "近接武器", cat)
	})

	t.Run("アイテム種別: 射撃武器", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Fire, &Fire{})
		cat, ok := c.CategoryOf(ItemTypeCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "射撃武器", cat)
	})

	t.Run("アイテム種別: Fire+Meleeは射撃武器になる", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Fire, &Fire{})
		entity.AddComponent(c.Melee, &Melee{})
		cat, ok := c.CategoryOf(ItemTypeCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "射撃武器", cat)
	})

	t.Run("アイテム種別: 防具", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Wearable, &Wearable{})
		cat, ok := c.CategoryOf(ItemTypeCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "防具", cat)
	})

	t.Run("CategoryはPredとして使える", func(t *testing.T) {
		t.Parallel()
		cats := c.Categories()
		weapon := cats[InventoryCategoryKey][1]
		entity := manager.NewEntity()
		entity.AddComponent(c.Melee, &Melee{})
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
		assert.Equal(t, "", cat)
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
