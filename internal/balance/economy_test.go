package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheapestFoodCostPerNutrition_食料なしは0(t *testing.T) {
	t.Parallel()
	// 食料アイテムが1つも無ければ最安費用は0に落ちる。
	assert.Equal(t, 0.0, CheapestFoodCostPerNutrition(rawschema.Raws{}))
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

func TestExpectedLootValue_該当行の重みの総和が0なら0(t *testing.T) {
	t.Parallel()
	raws := rawschema.Raws{
		ItemTables: &[]rawschema.ItemTable{{
			Id:      "zero_table",
			Entries: []rawschema.ItemTableEntry{{Id: "junk", MinDanger: 1, MaxDanger: 10, Weight: 0}},
		}},
	}
	assert.Equal(t, 0.0, ExpectedLootValue(raws, "zero_table", 5), "危険度に該当する行はあるが重みの総和が0なら0で割らず0を返す")
}

func TestExpectedGroupValue_グループでもコレクションでもないならアイテムの価値を返す(t *testing.T) {
	t.Parallel()
	raws := rawschema.Raws{
		Items: &[]rawschema.Item{{Id: "gem", Name: "Gem", Value: 42}},
	}
	assert.Equal(t, 42.0, expectedGroupValue(raws, "gem"), "グループ・コレクションどちらでもない id はアイテムとして直接解決する")
}

func TestExpectedGroupValue_どれにも解決しないidは0(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.0, expectedGroupValue(rawschema.Raws{}, "no_such_id"), "アイテムグループ・コレクション・アイテムのどれでもない id は0")
}

func TestExpectedGroupValue_グループの参照先アイテムが見つからなければ0(t *testing.T) {
	t.Parallel()
	raws := rawschema.Raws{
		ItemGroups: &[]rawschema.ItemGroup{{
			Id:      "junk",
			Entries: []rawschema.ItemGroupEntry{{Id: "no_such_item", Weight: 1, Pack: "1d1"}},
		}},
	}
	assert.Equal(t, 0.0, expectedGroupValue(raws, "junk"), "参照先のアイテムが1件も見つからなければ個数の総和が0のまま0を返す")
}

func TestExpectedPack_解釈できないダイス表記は0(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.0, expectedPack(""))
	assert.Equal(t, 0.0, expectedPack("not-a-dice"))
}

func TestExpectedPack_ダイスの期待値はMinとMaxの平均(t *testing.T) {
	t.Parallel()
	// 2d6 は 2..12 なので期待値は (2+12)/2 = 7
	assert.Equal(t, 7.0, expectedPack("2d6"))
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
	ruins, err := raw.GetDungeon(master, BaselineDungeon)
	require.NoError(t, err)
	assert.Equal(t, 0.0, ExpectedRunLootIncome(master, ruins, 0), "0層は収入0")
	i1 := ExpectedRunLootIncome(master, ruins, 1)
	i5 := ExpectedRunLootIncome(master, ruins, 5)
	assert.Positive(t, i1, "1層でも期待収入は正")
	assert.Greater(t, i5, i1, "層が深いほど高危険度の loot で収入が増える")
}

func TestCostOfLivingPerDay_最安食料で満腹度減耗を賄う(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	perNut := CheapestFoodCostPerNutrition(master)
	assert.Positive(t, perNut, "栄養価を持つ食料があるので費用は正")
	// 1日の生活費 = 1日の満腹度減耗(turnsPerDay/HungerDrainTurns) × 満腹度1点あたりの最安食料費。
	p := DefaultParams(master)
	want := p.TurnsPerDay / p.HungerDrainTurns * perNut
	assert.InDelta(t, want, CostOfLivingPerDay(master, p), 1e-9, "生活費は減耗×最安食料費")
}

func TestEconomyProgression_進行で1日分の食費が軽くなる(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	curve := EconomyProgression(master, "ruins_area", BaselineDays, DefaultParams(master))
	require.Len(t, curve, BaselineDays)
	// 危険度が上がると loot 手取りが増えるので、1日分の食費を賄う loot 個数は序盤より終盤で減る。
	assert.Greater(t, curve[0].LootPerDayFood, curve[BaselineDays-1].LootPerDayFood, "終盤ほど食費が軽い")
	assert.Positive(t, curve[BaselineDays-1].LootPerDayFood, "個数は正")
}
