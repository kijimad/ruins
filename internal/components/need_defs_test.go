package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

// testNeedDefs は components のテストが使う空腹と疲労の段階の表。値は raw の行と同じにする
var testNeedDefs = NeedDefs{
	Hunger: []HungerStage{
		{Level: rawschema.HungerStageIdSatiated, Name: "Full", From: 0.95},
		{Level: rawschema.HungerStageIdNormal, Name: "Normal", From: 0.66},
		{Level: rawschema.HungerStageIdHungry, Name: "Hungry", From: 0.33, Condition: SeverityMinor},
		{Level: rawschema.HungerStageIdStarving, Name: "Starving", From: 0, Condition: SeverityMedium},
	},
	Fatigue: []FatigueStage{
		{Level: rawschema.FatigueStageIdRested, Name: "Rested", From: 0},
		{Level: rawschema.FatigueStageIdNormal, Name: "Normal", From: 0.3},
		{Level: rawschema.FatigueStageIdTired, Name: "Tired", From: 0.5, Condition: SeverityMinor},
		{Level: rawschema.FatigueStageIdExhausted, Name: "Exhausted", From: 0.8, Condition: SeverityMedium},
	},
}

func TestNeedDefs_HungerStageOf(t *testing.T) {
	t.Parallel()

	t.Run("登録済みの段階を返す", func(t *testing.T) {
		t.Parallel()
		stage, ok := testNeedDefs.HungerStageOf(rawschema.HungerStageIdHungry)
		assert.True(t, ok)
		assert.Equal(t, HungerStage{Level: rawschema.HungerStageIdHungry, Name: "Hungry", From: 0.33, Condition: SeverityMinor}, stage)
	})

	t.Run("未登録はok=falseでゼロ値", func(t *testing.T) {
		t.Parallel()
		stage, ok := testNeedDefs.HungerStageOf(rawschema.HungerStageId("unknown"))
		assert.False(t, ok)
		assert.Equal(t, HungerStage{}, stage)
	})
}

func TestNeedDefs_FatigueStageOf(t *testing.T) {
	t.Parallel()

	t.Run("登録済みの段階を返す", func(t *testing.T) {
		t.Parallel()
		stage, ok := testNeedDefs.FatigueStageOf(rawschema.FatigueStageIdTired)
		assert.True(t, ok)
		assert.Equal(t, FatigueStage{Level: rawschema.FatigueStageIdTired, Name: "Tired", From: 0.5, Condition: SeverityMinor}, stage)
	})

	t.Run("未登録はok=falseでゼロ値", func(t *testing.T) {
		t.Parallel()
		stage, ok := testNeedDefs.FatigueStageOf(rawschema.FatigueStageId("unknown"))
		assert.False(t, ok)
		assert.Equal(t, FatigueStage{}, stage)
	})
}

func TestNeedDefs_FreshnessOf(t *testing.T) {
	t.Parallel()
	defs := NeedDefs{Freshness: []FreshnessDef{
		{Level: rawschema.Fresh, From: 0, Name: "Fresh", NutritionPercent: 100},
		{Level: rawschema.Stale, From: 1, Name: "Stale", NutritionPercent: 50},
	}}

	t.Run("登録済みの段階を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, FreshnessDef{Level: rawschema.Stale, From: 1, Name: "Stale", NutritionPercent: 50}, defs.FreshnessOf(rawschema.Stale))
	})

	t.Run("未登録はゼロ値", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, FreshnessDef{}, defs.FreshnessOf(rawschema.FreshnessStageId("unknown")))
	})
}
