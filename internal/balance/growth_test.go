package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttacksToSkillLevel_範囲外は0(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, AttacksToSkillLevel(0, 0), "Lv0以下は0")
	assert.Equal(t, 0, AttacksToSkillLevel(0, 101), "上限超過は0")
}

func TestAttacksToSkillLevel_レベルとともに増え能力で減る(t *testing.T) {
	t.Parallel()
	// スキル値が上がるほど減衰で1レベルに要する攻撃が増える。単調増加。
	assert.Less(t, AttacksToSkillLevel(0, 10), AttacksToSkillLevel(0, 30), "高レベルほど攻撃回数が増える")
	// 能力値が高いほど成長が速く、同じレベルに要する攻撃が減る。
	assert.Less(t, AttacksToSkillLevel(10, 30), AttacksToSkillLevel(0, 30), "能力が高いほど少ない攻撃で到達")
}

func TestSkillLevelAfterAttacks_逆関数と整合(t *testing.T) {
	t.Parallel()
	// Lv30 到達に要した攻撃回数を打てば、ちょうど Lv30 以上に届く。
	atks := AttacksToSkillLevel(0, 30)
	assert.GreaterOrEqual(t, SkillLevelAfterAttacks(0, atks), 30, "到達攻撃回数でLv30以上")
	// 1回少なければ Lv30 未満。境界の一貫性。
	assert.Less(t, SkillLevelAfterAttacks(0, atks-1), 30, "1回足りないとLv30未満")
}
