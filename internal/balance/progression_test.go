package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpectedSkillLevelAtDay_攻撃頻度と日数で単調に伸びる(t *testing.T) {
	t.Parallel()
	// 攻撃頻度が同じなら日が進むほどスキルは非減少。
	prev := 0
	for day := 1; day <= 21; day++ {
		lv := ExpectedSkillLevelAtDay(0, DefaultAttacksPerDay, day)
		assert.GreaterOrEqual(t, lv, prev, "日が進むとスキルは非減少")
		prev = lv
	}
	// 想定攻撃頻度で20日戦えば素手スキルはおおむね Lv30 前後へ達する。
	assert.InDelta(t, 30, ExpectedSkillLevelAtDay(0, DefaultAttacksPerDay, 20), 3, "20日でLv30前後")
}

func TestProgressionCurve_想定は床より安全(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	require.NoError(t, err)
	weapon, err := LoadWeaponFromItem(master, BaselineWeapon)
	require.NoError(t, err)
	curve, err := ProgressionCurve(master, player, weapon, BaselineAreaTable, BaselineDays, DefaultAttacksPerDay)
	require.NoError(t, err)
	require.Len(t, curve, BaselineDays)

	for _, d := range curve {
		// 成長は攻撃力だけを上げるので、想定プレイヤーは床以上に安全で、決着も速い。
		assert.LessOrEqual(t, d.DeathExpected, d.DeathFloor+1e-9, "想定は床より死ににくい")
		assert.LessOrEqual(t, d.TurnsExpected, d.TurnsFloor+1e-9, "想定は床より速く倒す")
	}
	// 終盤は床では危険だが、想定プレイヤーは成長で危険をほぼ無効化する。進行度調整の効き具合を固定する。
	last := curve[BaselineDays-1]
	assert.Greater(t, last.DeathFloor, 0.10, "終盤の床は危険")
	assert.Less(t, last.DeathExpected, 0.02, "終盤の想定は成長でほぼ無害")
}
