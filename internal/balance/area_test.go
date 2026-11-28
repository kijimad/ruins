package balance

import (
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeasureAreaSupply_食べ物の栄養と食べ物でない物の熱を足す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	bread, err := lifecycle.SpawnFieldItem(world, "garlic_bread", 1, 1, 1)
	require.NoError(t, err)
	_, err = lifecycle.SpawnFieldItem(world, "garlic_bread", 2, 1, 1)
	require.NoError(t, err)
	biscuit, err := lifecycle.SpawnFieldItem(world, "biscuit", 3, 1, 1)
	require.NoError(t, err)
	sword, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 4, 1, 1)
	require.NoError(t, err)
	breadAmount := world.Components.ProvidesNutrition.Get(bread).Amount
	biscuitAmount := world.Components.ProvidesNutrition.Get(biscuit).Amount
	require.Less(t, world.Components.Perishable.Get(bread).StageLength, consts.Turn(stayDays)*gc.TurnsPerDay, "前提: パンは留まる間に傷む")
	require.GreaterOrEqual(t, world.Components.Perishable.Get(biscuit).StageLength, consts.Turn(stayDays)*gc.TurnsPerDay, "前提: ビスケットは留まる間は新鮮なまま")
	require.Positive(t, query.HeatContent(world, bread), "前提: パンも燃える")

	got := MeasureAreaSupply(world)

	assert.Equal(t, AreaSupply{
		Nutrition:        2*breadAmount + biscuitAmount,
		KeepingNutrition: biscuitAmount,
		Heat:             query.HeatContent(world, sword),
	}, got, "食べ物は燃料に数えない")
}

func TestFoodDays_1日に減る満腹度で割る(t *testing.T) {
	t.Parallel()
	// 実際の値と切り離した仮の値。1日に500減る
	p := Params{TurnsPerDay: 1500, HungerDrainTurns: 3}
	assert.InDelta(t, 2.0, FoodDays(p, 1000), 1e-9, "1日に500減るので1000で2日")
}

func TestFuelNights_地面の火で夜の500ターンを何晩燃やせるか(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 1.0, FuelNights(consts.Heat(1000)), 1e-9, "地面の燃焼効率50%で500ターン燃える")
}

func TestRenderAreaSupplyMarkdown_深さごとに中央値と幅を並べる(t *testing.T) {
	t.Parallel()
	// 実際の値と切り離した仮の値。1日に500減る
	p := Params{TurnsPerDay: 1500, HungerDrainTurns: 3}
	md := RenderAreaSupplyMarkdown(p, []AreaSupplySample{
		{Seed: 1, Depth: 40, Supply: AreaSupply{Nutrition: 500}},
		{Seed: 1, Depth: 0, Supply: AreaSupply{Nutrition: 1000, KeepingNutrition: 500, Heat: 1000}},
		{Seed: 2, Depth: 0, Supply: AreaSupply{Nutrition: 3000, KeepingNutrition: 1500, Heat: 1000}},
	})

	var rows [][]string
	for line := range strings.SplitSeq(md, "\n") {
		if !strings.HasPrefix(line, "|") || strings.HasPrefix(line, "|-") || strings.Contains(line, "北の深さ") {
			continue
		}
		var cells []string
		for c := range strings.SplitSeq(strings.Trim(line, "|"), "|") {
			cells = append(cells, strings.TrimSpace(c))
		}
		rows = append(rows, cells)
	}
	assert.Equal(t, [][]string{
		{"0", "4.0", "2.0〜6.0", "2.0", "1.0〜3.0", "1.0", "1.0〜1.0"},
		{"40", "1.0", "1.0〜1.0", "0.0", "0.0〜0.0", "0.0", "0.0〜0.0"},
	}, rows)
}
