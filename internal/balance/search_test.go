package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolveScalar_単調関数の解を挟んで返す(t *testing.T) {
	t.Parallel()
	// eval(x)=2x は単調。target=3 の解 x=1.5 を [0,5] で挟んで返す。
	x, ok := SolveScalar(func(x float64) float64 { return 2 * x }, 3, 0, 5, 40)
	require.True(t, ok)
	assert.InDelta(t, 1.5, x, 1e-3)
}

func TestSolveScalar_範囲外は両立不能(t *testing.T) {
	t.Parallel()
	// target=20 は [0,5] の eval(x)=2x では最大10で届かない。ok=false を返す。
	_, ok := SolveScalar(func(x float64) float64 { return 2 * x }, 20, 0, 5, 40)
	assert.False(t, ok)
}

func TestWithScaledMeleeDamage_不在アイテムはそのまま実行する(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	before := ruinsDay20PowerRatio(master)
	called := false
	// 存在しないアイテムIDでは何も変更せず fn だけ呼ぶ。指標は変わらない
	withScaledMeleeDamage(master, "no_such_item", 2.0, func() {
		called = true
		assert.InDelta(t, before, ruinsDay20PowerRatio(master), 1e-9, "変更されない")
	})
	assert.True(t, called, "不在でも fn は呼ばれる")
}

func TestSolveScalar_敵武器スケールで目標戦力比を探す(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	// 敵武器 bite のダメージ倍率 k を動かし、廃墟 day20 戦力比を目標へ寄せる k を探す。
	// bite を強くするほど戦力比は下がるので単調。現状 day20 より少し下げる目標を狙う。
	base := ruinsDay20PowerRatio(master)
	target := base - 0.2
	eval := func(k float64) float64 {
		var got float64
		withScaledMeleeDamage(master, "bite", k, func() { got = ruinsDay20PowerRatio(master) })
		return got
	}
	k, ok := SolveScalar(eval, target, 1.0, 4.0, 40)
	require.True(t, ok, "現実的な倍率で目標を挟めること")
	assert.InDelta(t, target, eval(k), 0.03, "見つけた倍率で目標戦力比に収まる")
	// master を汚していないこと。探索後も base に戻る
	assert.InDelta(t, base, ruinsDay20PowerRatio(master), 1e-9)
}

func TestDominates_全目的以下かつ一部真に小で支配(t *testing.T) {
	t.Parallel()
	// 全目的で以下かつ少なくとも1つで真に小さいときだけ支配する。
	assert.True(t, dominates([]float64{0.1, 0.2}, []float64{0.1, 0.3}), "片方が小・他方が同じは支配")
	assert.False(t, dominates([]float64{0.1, 0.2}, []float64{0.1, 0.2}), "同一は支配しない")
	assert.False(t, dominates([]float64{0.1, 0.3}, []float64{0.2, 0.2}), "一勝一敗は支配しない")
}

func TestParetoFront_非劣集合を返す(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	base := ruinsDay20PowerRatio(master)
	// プレイヤー武器と敵武器を動かし、序盤day1・中盤day10・終盤day20の目標逸脱を目的にする。
	front := ParetoFront(master, []string{"bare_hands", "bite"}, []float64{0.8, 1.0, 1.5}, []int{1, 10, 20})
	require.NotEmpty(t, front, "非劣集合は空でない")
	for _, p := range front {
		assert.Len(t, p.Deviations, 3, "代表日3つ分の逸脱を持つ")
	}
	// 前線内の任意の2点は互いに支配しない
	for i := range front {
		for j := range front {
			if i != j {
				assert.False(t, dominates(front[i].Deviations, front[j].Deviations), "前線内は互いに非劣")
			}
		}
	}
	// 探索後も master を汚していない
	assert.InDelta(t, base, ruinsDay20PowerRatio(master), 1e-9)
}
