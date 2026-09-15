package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeaponViability_viableと罠と差を集計する(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	s, err := WeaponViability(master, "ruins_area", 20)
	require.NoError(t, err)
	require.Positive(t, s.Total, "近接武器を評価する")

	// viable は総数以下で、割合は0..1。
	assert.LessOrEqual(t, s.Viable, s.Total)
	assert.GreaterOrEqual(t, s.ViableRate(), 0.0)
	assert.LessOrEqual(t, s.ViableRate(), 1.0)
	// バランス調整で trap を解消したので罠は0。新たな trap を持ち込んだら検知する回帰ゲート。
	assert.Zero(t, s.Traps, "調整後は素手より弱い装備可能武器がない")
	// viable が存在するなら決着ターンの幅は非負で、その差が選択の意味になる。
	if s.Viable > 0 {
		assert.GreaterOrEqual(t, s.ViableTTKMax, s.ViableTTKMin)
	}
}

func TestViabilitySummary_symmetryとviabilityの判定(t *testing.T) {
	t.Parallel()
	// 幅が下限未満なら symmetry 寄り、超えれば viability ありと判定する。
	flat := ViabilitySummary{ViableTTKMin: 3.0, ViableTTKMax: 3.2}
	assert.False(t, flat.Distinct(), "幅が小さいと symmetry 寄り")
	varied := ViabilitySummary{ViableTTKMin: 1.8, ViableTTKMax: 4.4}
	assert.True(t, varied.Distinct(), "幅が大きいと viability あり")
}
