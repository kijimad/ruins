package balance

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestCalcHealing_Numeral(t *testing.T) {
	t.Parallel()

	ph := &gc.ProvidesHealing{Kind: gc.HealNumeral, Numeral: 50}
	assert.Equal(t, 50, calcHealing(ph, 100))
}

func TestCalcHealing_Ratio(t *testing.T) {
	t.Parallel()

	ph := &gc.ProvidesHealing{Kind: gc.HealRatio, Ratio: 0.5}
	assert.Equal(t, 50, calcHealing(ph, 100))
}

func TestBattleStats_DPS(t *testing.T) {
	t.Parallel()

	t.Run("結果がない場合は0", func(t *testing.T) {
		t.Parallel()
		bs := BattleStats{}
		assert.Equal(t, 0.0, bs.DPS())
	})

	t.Run("ターンが0の場合は0", func(t *testing.T) {
		t.Parallel()
		bs := BattleStats{Results: []BattleResult{{DamageDealt: 10, Turns: 0}}}
		assert.Equal(t, 0.0, bs.DPS())
	})

	t.Run("正常な計算", func(t *testing.T) {
		t.Parallel()
		bs := BattleStats{Results: []BattleResult{
			{DamageDealt: 100, Turns: 10},
			{DamageDealt: 200, Turns: 20},
		}}
		assert.InDelta(t, 10.0, bs.DPS(), 0.01)
	})
}

func TestRunStats_DeathRate(t *testing.T) {
	t.Parallel()

	t.Run("結果がない場合は0", func(t *testing.T) {
		t.Parallel()
		s := RunStats{}
		assert.Equal(t, 0.0, s.DeathRate())
	})

	t.Run("全員死亡", func(t *testing.T) {
		t.Parallel()
		s := RunStats{Results: []RunResult{
			{Died: true}, {Died: true},
		}}
		assert.Equal(t, 1.0, s.DeathRate())
	})

	t.Run("半数死亡", func(t *testing.T) {
		t.Parallel()
		s := RunStats{Results: []RunResult{
			{Died: true}, {Died: false},
		}}
		assert.Equal(t, 0.5, s.DeathRate())
	})
}

func TestRunStats_MedianDepth(t *testing.T) {
	t.Parallel()

	s := RunStats{Results: []RunResult{
		{ReachedDepth: 3},
		{ReachedDepth: 5},
		{ReachedDepth: 7},
	}}
	assert.Equal(t, 5, s.MedianDepth())
}

func TestRunStats_HPAtDepth(t *testing.T) {
	t.Parallel()

	s := RunStats{Results: []RunResult{
		{HPByDepth: map[int]int{1: 80, 2: 60}},
		{HPByDepth: map[int]int{1: 90}},
	}}

	hps := s.HPAtDepth(1)
	assert.Len(t, hps, 2)

	hps3 := s.HPAtDepth(3)
	assert.Empty(t, hps3)
}

func TestRunStats_SuddenDeathRate(t *testing.T) {
	t.Parallel()

	t.Run("結果がない場合は0", func(t *testing.T) {
		t.Parallel()
		s := RunStats{}
		assert.Equal(t, 0.0, s.SuddenDeathRate(1))
	})

	t.Run("depth<1は1に補正される", func(t *testing.T) {
		t.Parallel()
		s := RunStats{Results: []RunResult{
			{ReachedDepth: 1, Died: true},
			{ReachedDepth: 5, Died: false},
		}}
		assert.InDelta(t, 0.5, s.SuddenDeathRate(0), 0.01)
	})

	t.Run("到達者がいない場合は0", func(t *testing.T) {
		t.Parallel()
		s := RunStats{Results: []RunResult{
			{ReachedDepth: 1, Died: true},
		}}
		assert.Equal(t, 0.0, s.SuddenDeathRate(5))
	})

	t.Run("突然死率の計算", func(t *testing.T) {
		t.Parallel()
		s := RunStats{Results: []RunResult{
			{ReachedDepth: 3, Died: true},
			{ReachedDepth: 5, Died: false},
			{ReachedDepth: 3, Died: true},
			{ReachedDepth: 10, Died: false},
		}}
		// 深度3に到達: 4人、深度3で死亡: 2人 → 0.5
		assert.InDelta(t, 0.5, s.SuddenDeathRate(3), 0.01)
	})
}

func TestRunStats_FloorDamagePercentile(t *testing.T) {
	t.Parallel()

	s := RunStats{Results: []RunResult{
		{
			HPByDepth:           map[int]int{1: 70},
			HPBeforeHealByDepth: map[int]int{1: 60},
		},
		{
			HPByDepth:           map[int]int{1: 80},
			HPBeforeHealByDepth: map[int]int{1: 50},
		},
	}}

	// 深度1: playerMaxHP=100から開始
	// ラン1: 100-60=40ダメージ、ラン2: 100-50=50ダメージ
	// ソート後 [40,50]、Percentile(p=0.5) は idx=int(1*0.5)=0 で 40 を返す
	result := s.MedianDamagePerFloor(1, 100)
	assert.Equal(t, 40, result)
}

func TestRunStats_FloorHealingPercentile(t *testing.T) {
	t.Parallel()

	s := RunStats{Results: []RunResult{
		{
			HPByDepth:           map[int]int{1: 80},
			HPBeforeHealByDepth: map[int]int{1: 60},
		},
	}}

	// 回復量: 80-60=20
	result := s.MedianHealingPerFloor(1)
	assert.Equal(t, 20, result)
}

func TestRunStats_FloorHealingPercentile_NoData(t *testing.T) {
	t.Parallel()

	s := RunStats{}
	assert.Equal(t, 0, s.MedianHealingPerFloor(1))
}

func TestRunStats_PercentileWrappers(t *testing.T) {
	t.Parallel()

	s := RunStats{Results: []RunResult{
		{
			ReachedDepth:        5,
			HPByDepth:           map[int]int{1: 80},
			HPBeforeHealByDepth: map[int]int{1: 70},
			WeaponDamageByDepth: map[int]int{1: 10},
			AvgKillTurnsByDepth: map[int]int{1: 3},
			HungerByDepth:       map[int]int{1: 400},
		},
	}}

	// 各ラッパーが値を返せることを確認する
	assert.Equal(t, 80, s.MedianHP(1))
	assert.Equal(t, 80, s.P5HP(1))
	assert.Equal(t, 80, s.P95HP(1))
	assert.Equal(t, 70, s.MedianHPBeforeHeal(1))
	assert.Equal(t, 70, s.P5HPBeforeHeal(1))
	assert.Equal(t, 70, s.P95HPBeforeHeal(1))
	assert.Equal(t, 10, s.MedianWeaponDamage(1))
	assert.Equal(t, 10, s.P5WeaponDamage(1))
	assert.Equal(t, 10, s.P95WeaponDamage(1))
	assert.Equal(t, 3, s.MedianKillTurns(1))
	assert.Equal(t, 3, s.P5KillTurns(1))
	assert.Equal(t, 3, s.P95KillTurns(1))
	assert.Equal(t, 400, s.MedianHunger(1))
	assert.Equal(t, 400, s.P5Hunger(1))
	assert.Equal(t, 400, s.P95Hunger(1))
}
