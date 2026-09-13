package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepresentativeBuilds_倍率で武器が強くなる(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	builds, err := RepresentativeBuilds(master)
	require.NoError(t, err)
	require.Len(t, builds, 3, "下限・中盤・後半の3ビルド")
	// 段階が進むほど武器ダメージが強い
	assert.Less(t, builds[0].Weapon.Damage, builds[1].Weapon.Damage, "中盤は下限より強い")
	assert.Less(t, builds[1].Weapon.Damage, builds[2].Weapon.Damage, "後半は中盤より強い")
}

func TestBuildSpread_強ビルドほど戦力比が高い(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	builds, err := RepresentativeBuilds(master)
	require.NoError(t, err)
	spread, err := BuildSpread(master, builds, "ruins_area", 20)
	require.NoError(t, err)
	require.Len(t, spread, 20, "日数ぶんの行")
	for _, r := range spread {
		// 強武器ほど倒すのが速く戦力比が上がるので、最大は最小以上。幅は非負。
		assert.GreaterOrEqual(t, r.MaxRatio, r.MinRatio, "最強ビルドは下限以上の戦力比")
		assert.GreaterOrEqual(t, r.Spread(), 0.0, "幅は非負")
	}
	// 3ビルドあるので、少なくともどこかで幅が正になる
	var maxSpread float64
	for _, r := range spread {
		maxSpread = max(maxSpread, r.Spread())
	}
	assert.Positive(t, maxSpread, "ビルド差で戦力比に幅が出る")
}

func TestStageSpreads_段階ごとの幅と判定(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	builds, err := RepresentativeBuilds(master)
	require.NoError(t, err)
	stages, err := StageSpreads(master, builds, "ruins_area")
	require.NoError(t, err)
	require.Len(t, stages, 3, "序盤・中盤・終盤の3段階")
	for _, s := range stages {
		assert.GreaterOrEqual(t, s.MaxSpread, 0.0, s.Stage+" の最大幅は非負")
		assert.Positive(t, s.Tolerance, s.Stage+" の許容幅は正")
		// InRange は MaxSpread<=Tolerance と一致する
		assert.Equal(t, s.MaxSpread <= s.Tolerance, s.InRange())
	}
}
