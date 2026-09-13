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
	assert.InDelta(t, 1.88, curve[19].PowerRatio, 0.05, "廃墟 day20 の戦力比")
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
}
