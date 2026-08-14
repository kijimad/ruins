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

	t.Run("フィールド観察: 自分", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Player.Add(entity, &Player{})
		c.FactionAlly.Add(entity, &FactionAlly{})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "Self", cat)
	})

	t.Run("フィールド観察: 敵", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.FactionEnemy.Add(entity, &FactionEnemy{})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "Enemy", cat)
	})

	t.Run("フィールド観察: NPC(味方)", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.FactionAlly.Add(entity, &FactionAlly{})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "NPC", cat)
	})

	t.Run("フィールド観察: NPC(中立)", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.FactionNeutral.Add(entity, &FactionNeutral{})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "NPC", cat)
	})

	t.Run("フィールド観察: 固定物", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Fixed.Add(entity, &Fixed{})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "Fixed object", cat)
	})

	t.Run("フィールド観察: タイル", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Tile.Add(entity, &Tile{})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "Tile", cat)
	})

	t.Run("フィールド観察: Player+FactionAllyは自分になる", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		entity := manager.NewEntity()
		c.Player.Add(entity, &Player{})
		c.FactionAlly.Add(entity, &FactionAlly{})
		cat, ok := c.CategoryOf(FieldLookCategoryKey, entity)
		assert.True(t, ok)
		assert.Equal(t, "Self", cat)
	})

	t.Run("CategoryはPredとして使える", func(t *testing.T) {
		t.Parallel()
		manager, c := setupComponents(t)
		cats := c.Categories()
		enemy := cats[FieldLookCategoryKey][1]
		entity := manager.NewEntity()
		c.FactionEnemy.Add(entity, &FactionEnemy{})
		assert.True(t, enemy.Eval(entity))
		assert.Equal(t, "Enemy", enemy.String())
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
