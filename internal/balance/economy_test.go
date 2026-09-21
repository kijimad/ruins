package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheapestFoodCostPerNutrition_食料なしは0(t *testing.T) {
	t.Parallel()
	// 食料アイテムが1つも無ければ最安費用は0に落ちる。
	assert.Equal(t, 0.0, CheapestFoodCostPerNutrition(oapi.Raws{}))
}

func TestExpectedItemsPerFloor_奇数前提で厳密(t *testing.T) {
	t.Parallel()
	// rand(0..floorItemRandom-1) の期待値 (floorItemRandom-1)/2 が整数除算で厳密になるのは奇数のときだけ。
	// 偶数化すると expectedItemsPerFloor が無言で切り捨てに落ちるので、前提を機械で守る。
	assert.Equal(t, 1, floorItemRandom%2, "floorItemRandom は奇数")
}

func TestExpectedLootValue_危険度で上がり正の値(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	d1 := ExpectedLootValue(master, "ruins_area", 1)
	d8 := ExpectedLootValue(master, "ruins_area", 8)
	assert.Positive(t, d1, "危険度1でも拾える期待価値は正")
	assert.Greater(t, d8, d1, "危険度が上がると期待価値も上がる")
	assert.Equal(t, 0.0, ExpectedLootValue(master, "no_such_table", 1), "不明テーブルは0")
}

func TestExpectedNetLootValue_手取りは額面より小さく正(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	face := ExpectedLootValue(master, "ruins_area", 8)
	net := ExpectedNetLootValue(master, "ruins_area", 8)
	assert.Positive(t, net, "手取りは正")
	assert.Less(t, net, face, "売却倍率を掛けるので手取りは額面より小さい")
	assert.Equal(t, 0.0, ExpectedNetLootValue(master, "no_such_table", 1), "不明テーブルは0")
}

func TestExpectedRunLootIncome_層数で単調に増える(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	assert.Equal(t, 0.0, ExpectedRunLootIncome(master, "ruins_area", 0), "0層は収入0")
	i1 := ExpectedRunLootIncome(master, "ruins_area", 1)
	i5 := ExpectedRunLootIncome(master, "ruins_area", 5)
	assert.Positive(t, i1, "1層でも期待収入は正")
	assert.Greater(t, i5, i1, "層が深いほど高危険度の loot で収入が増える")
}

func TestCostOfLivingPerDay_最安食料で満腹度減耗を賄う(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	perNut := CheapestFoodCostPerNutrition(master)
	assert.Positive(t, perNut, "栄養価を持つ食料があるので費用は正")
	// 1日の生活費 = 1日の満腹度減耗(turnsPerDay/HungerDrainTurns) × 満腹度1点あたりの最安食料費。
	p := DefaultParams()
	want := p.TurnsPerDay / p.HungerDrainTurns * perNut
	assert.InDelta(t, want, CostOfLivingPerDay(master, p), 1e-9, "生活費は減耗×最安食料費")
}

func TestEconomyProgression_進行で1日分の食費が軽くなる(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	curve := EconomyProgression(master, "ruins_area", BaselineDays, DefaultParams())
	require.Len(t, curve, BaselineDays)
	// 危険度が上がると loot 手取りが増えるので、1日分の食費を賄う loot 個数は序盤より終盤で減る。
	assert.Greater(t, curve[0].LootPerDayFood, curve[BaselineDays-1].LootPerDayFood, "終盤ほど食費が軽い")
	assert.Positive(t, curve[BaselineDays-1].LootPerDayFood, "個数は正")
}
