package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ベースラインの凍結ゲート。導出した指標を現在値で pin し、パラメータや式の変更で値が動いたら
// make check で検知する。これは「値が正しい」の主張ではなく「今この値だ」という事実の固定と、
// 変化の検知である。意図した調整で値が動いたら、この期待値を更新する。golden と同じ運用。
//
// 目標帯そのものを assert しないのは、現状の後半カーブが目標外で、目標で縛ると CI が常時赤に
// なり運用できないため。目標との差は baseline.md が見せ、ゲートは意図しない変化だけを止める。

func TestBaselineSnapshot_序盤戦闘(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	player, err := LoadCombatantFromMember(master, "ash")
	require.NoError(t, err)
	weapon, err := LoadWeaponFromItem(master, "bare_hands")
	require.NoError(t, err)
	curve, err := DifficultyCurve(master, player, weapon, "ruins_area", 21)
	require.NoError(t, err)
	assert.InDelta(t, 2.59, curve[0].PowerRatio, 0.05, "廃墟 day1 の戦力比")
	assert.InDelta(t, 1.58, curve[19].PowerRatio, 0.05, "廃墟 day20 の戦力比。傾き調整で後半が下がった")

	forest, err := DifficultyCurve(master, player, weapon, "forest", 21)
	require.NoError(t, err)
	assert.InDelta(t, 2.50, forest[0].PowerRatio, 0.05, "森 day1 の戦力比。全日を目標帯へ寄せた")
	assert.InDelta(t, 1.24, forest[19].PowerRatio, 0.05, "森 day20 の戦力比")

	cave, err := DifficultyCurve(master, player, weapon, "cave", 21)
	require.NoError(t, err)
	assert.InDelta(t, 2.52, cave[0].PowerRatio, 0.05, "洞窟 day1 の戦力比。全日を目標帯へ寄せた")
	assert.InDelta(t, 1.16, cave[19].PowerRatio, 0.05, "洞窟 day20 の戦力比")
}

func TestBaselineSnapshot_生存圧(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 0.67, DaysUntilStarving(), 0.02, "栄養失調まで日数")
	assert.InDelta(t, 1.00, DaysUntilHungerEmpty(), 0.02, "満腹度が尽きるまで日数")
	assert.InDelta(t, 10, TurnsToHypothermia(0), 0.5, "0度で低体温まで")
}

func TestBaselineSnapshot_物流(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 980, DriveRangeAllFuel(oapi.OIL, consts.CubeWeightCapacityKg), 1, "OIL満載の航続")
	assert.InDelta(t, 196, DriveRangeAllFuel(oapi.WOOD, consts.CubeWeightCapacityKg), 1, "WOOD満載の航続")
	assert.Equal(t, 5000, FuelBurnTurns(oapi.OIL, 10), "OIL 10kg の燃焼ターン")
	assert.Equal(t, 1000, FuelBurnTurns(oapi.WOOD, 10), "WOOD 10kg の燃焼ターン")
}

func TestBaselineSnapshot_気候(t *testing.T) {
	t.Parallel()
	// 世界温度の季節変動。1年32日で春5→夏ピーク22→冬底付近へ巡る。
	assert.Equal(t, 5, WorldTemperatureAtDay(1), "春の中点")
	assert.Equal(t, 22, WorldTemperatureAtDay(8), "夏ピーク")
	assert.Equal(t, -29, WorldTemperatureAtDay(24), "冬底付近")
}

func TestBaselineSnapshot_身体と経済(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 4, HPDrainPerTurnAtBlood(0), "血液0でのHP減")
	assert.InDelta(t, 1.07, DaysUntilExhausted(), 0.02, "過労までの日数")
	assert.InDelta(t, 0.855, AuctionTakeHomeRate(1000, 1), 0.001, "競売の手取り率")
	assert.InDelta(t, 0.25, SleepTimeFraction(), 0.001, "釣り合いに要する睡眠時間の割合")
	assert.InDelta(t, 666.67, SleepTurnsToFullRecover(), 1, "満タンから睡眠で回復し切るターン数")
}

func TestBaselineSnapshot_探索収入(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	assert.InDelta(t, 47, ExpectedNetLootValue(master, "ruins_area", 8), 2, "危険度8の廃墟で拾える1個あたりの期待手取り")
	assert.InDelta(t, 2565, ExpectedRunLootIncome(master, "ruins_area", 5), 50, "廃墟5層探索の期待収入")
}

func TestBaselineSnapshot_進行成長(t *testing.T) {
	t.Parallel()
	// 能力値0の下限で、スキルを上げるのに要する攻撃回数。減衰で高レベルほど急に増える。
	assert.Equal(t, 210, AttacksToSkillLevel(0, 10), "Lv10到達の攻撃回数")
	assert.Equal(t, 1626, AttacksToSkillLevel(0, 30), "Lv30到達の攻撃回数")
}

func TestTargetCheck_InRange_帯の内外(t *testing.T) {
	t.Parallel()
	assert.True(t, TargetCheck{Value: 1.0, Lo: 0.8, Hi: 1.3}.InRange(), "帯内は真")
	assert.False(t, TargetCheck{Value: 1.5, Lo: 0.8, Hi: 1.3}.InRange(), "上限超過は偽")
	assert.False(t, TargetCheck{Value: 0.5, Lo: 0.8, Hi: 1.3}.InRange(), "下限未満は偽")
}

func TestDomainTargets_全ドメインが健全な帯を持つ(t *testing.T) {
	t.Parallel()
	// 目標帯そのものは assert しない。ドメインが揃い帯が妥当な向きかだけを検証する。
	checks := DomainTargets()
	assert.Len(t, checks, 6, "戦闘以外のスカラー6ドメイン")
	for _, c := range checks {
		assert.Less(t, c.Lo, c.Hi, c.Domain+" の目標帯は下限<上限")
	}
}
