package components

import (
	"testing"

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
		assert.Equal(t, "武器", c.CategoryOf(InventoryCategoryKey, entity))
	})

	t.Run("防具カテゴリ", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Wearable, &Wearable{})
		assert.Equal(t, "防具", c.CategoryOf(InventoryCategoryKey, entity))
	})

	t.Run("射撃武器カテゴリ", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Fire, &Fire{})
		assert.Equal(t, "武器", c.CategoryOf(InventoryCategoryKey, entity))
	})

	t.Run("道具カテゴリは本を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Book, &Book{})
		assert.Equal(t, "道具", c.CategoryOf(InventoryCategoryKey, entity))
	})

	t.Run("道具カテゴリは消耗品を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Consumable, &Consumable{})
		assert.Equal(t, "道具", c.CategoryOf(InventoryCategoryKey, entity))
	})

	t.Run("道具カテゴリは素材を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Material, &Material{})
		assert.Equal(t, "道具", c.CategoryOf(InventoryCategoryKey, entity))
	})

	t.Run("道具カテゴリは弾薬を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Ammo, &Ammo{})
		assert.Equal(t, "道具", c.CategoryOf(InventoryCategoryKey, entity))
	})

	t.Run("道具カテゴリは素材を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Material, nil)
		assert.Equal(t, "道具", c.CategoryOf(InventoryCategoryKey, entity))
	})

	t.Run("道具カテゴリは置物を含む", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Prop, nil)
		assert.Equal(t, "道具", c.CategoryOf(InventoryCategoryKey, entity))
	})

	t.Run("どのカテゴリにも属さない", func(t *testing.T) {
		t.Parallel()
		entity := manager.NewEntity()
		entity.AddComponent(c.Name, &Name{Name: "何か"})
		assert.Equal(t, "", c.CategoryOf(InventoryCategoryKey, entity))
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
