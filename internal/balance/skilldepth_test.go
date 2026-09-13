package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillDepthProfileFor_実効と死んだティアで全段を分ける(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	prof, err := SkillDepthProfileFor(master, BaselineWeapon, "ruins_area", 20)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(prof.Tiers), 2)

	// 隣接ティアは実効か死んでいるかのどちらかで、合計は遷移数に一致する。
	transitions := len(prof.Tiers) - 1
	assert.Equal(t, transitions, prof.EffectiveSteps+prof.DeadTiers, "実効と死んだティアで全遷移を尽くす")

	// スキルが上がると武器ダメージは非減少で、撃破ターンは非増加。強くなるほど速く倒せる。
	for i := 1; i < len(prof.Tiers); i++ {
		assert.GreaterOrEqual(t, prof.Tiers[i].WeaponDamage, prof.Tiers[i-1].WeaponDamage, "ダメージは非減少")
		assert.LessOrEqual(t, prof.Tiers[i].PlayerTTK, prof.Tiers[i-1].PlayerTTK+1e-9, "撃破ターンは非増加")
	}
}

func TestSkillDepthProfileFor_弱い武器ほど死んだティアが多い(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	// 素手はダメージが小さく、+5%%/レベルの倍率が整数丸めで多く吸われるので死んだティアが多い。
	// ダメージの大きい鉄剣は同じ倍率でも刻みが立ちやすく、実効ティアが増える。
	bare, err := SkillDepthProfileFor(master, BaselineWeapon, "ruins_area", 20)
	require.NoError(t, err)
	iron, err := SkillDepthProfileFor(master, "iron_sword", "ruins_area", 20)
	require.NoError(t, err)
	assert.Greater(t, bare.DeadTiers, iron.DeadTiers, "弱い素手のほうが死んだティアが多い")
	assert.Less(t, bare.EffectiveSteps, iron.EffectiveSteps, "弱い素手のほうが実効ティアが少ない")
}
