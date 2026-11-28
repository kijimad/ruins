package balance

import (
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeasureAreaSupply_全てのアイテムの栄養と熱を足す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	bread, err := lifecycle.SpawnFieldItem(world, "garlic_bread", 1, 1, 1)
	require.NoError(t, err)
	_, err = lifecycle.SpawnFieldItem(world, "garlic_bread", 2, 1, 1)
	require.NoError(t, err)
	sword, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 3, 1, 1)
	require.NoError(t, err)
	nutrition := world.Components.ProvidesNutrition.Get(bread).Amount
	require.Positive(t, nutrition, "前提: パンは栄養を持つ")
	require.Positive(t, query.HeatContent(world, sword), "前提: 木の剣は燃える")

	got := MeasureAreaSupply(world)

	assert.Equal(t, AreaSupply{
		Nutrition: 2 * nutrition,
		Heat:      2*query.HeatContent(world, bread) + query.HeatContent(world, sword),
	}, got)
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
		{Seed: 1, Depth: 0, Supply: AreaSupply{Nutrition: 1000, Heat: 1000}},
		{Seed: 2, Depth: 0, Supply: AreaSupply{Nutrition: 3000, Heat: 1000}},
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
		{"0", "4.0", "2.0〜6.0", "1.0", "1.0〜1.0"},
		{"40", "1.0", "1.0〜1.0", "0.0", "0.0〜0.0"},
	}, rows)
}
