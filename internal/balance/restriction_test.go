package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeaponRestrictionValues_不明なテーブルはエラー(t *testing.T) {
	t.Parallel()
	// 存在しない敵テーブルは基準の死亡確率が引けずエラーになる。
	master := loadTestMaster(t)
	_, _, err := WeaponRestrictionValues(master, "nonexistent", 20)
	require.Error(t, err)
}

func TestWeaponRestrictionValues_必須な武器ほど劣化量が大きい(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	values, baseDeath, err := WeaponRestrictionValues(master, "ruins_area", 20)
	require.NoError(t, err)
	require.NotEmpty(t, values)

	// 素手の死亡確率は正で、素手より強い武器は死亡確率を下げ劣化量が正になる。
	assert.Positive(t, baseDeath, "素手の死亡確率は正")
	// 劣化量降順に並ぶ。先頭は必須、末尾は素手と大差ないか劣る。
	for i := 1; i < len(values); i++ {
		assert.GreaterOrEqual(t, values[i-1].Degradation, values[i].Degradation, "劣化量降順")
	}
	// 先頭は素手より死亡確率が低い、すなわち劣化量が正の武器。
	assert.Positive(t, values[0].Degradation, "最も必須な武器は劣化量が正")
	// 劣化量は素手比の定義どおり baseDeath - DeathProbWith に一致する。
	for _, v := range values {
		assert.InDelta(t, baseDeath-v.DeathProbWith, v.Degradation, 1e-9, v.Element+" の劣化量の定義")
	}
}

func TestWeaponRestrictionValues_trap武器を解消済み(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	values, _, err := WeaponRestrictionValues(master, "ruins_area", 20)
	require.NoError(t, err)
	// バランス調整で、素手より有意に弱い装備可能武器(trap)を解消した。新たに trap を持ち込んだら
	// この回帰ゲートで検知する。劣化量が負なら素手より弱いことを意味する。
	for _, v := range values {
		assert.GreaterOrEqual(t, v.Degradation, -0.005, v.Element+" は素手より有意に弱くない")
	}
}
