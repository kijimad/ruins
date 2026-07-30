package consts

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDice_表記をDiceへ変換する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want Dice
	}{
		{"1d3+1", Dice{Base: 1, Sides: 3, Bonus: 1}},
		{"2d6", Dice{Base: 2, Sides: 6, Bonus: 0}},
		{"d6", Dice{Base: 1, Sides: 6, Bonus: 0}},
		{"3d4-1", Dice{Base: 3, Sides: 4, Bonus: -1}},
		{"5", Dice{Base: 0, Sides: 0, Bonus: 5}},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			got, err := ParseDice(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseDice_不正な表記はエラーになる(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in       string
		contains string
	}{
		{"", "空"},
		{"abc", "数値が不正"},
		{"1dx", "面数が不正"},
		{"xd6", "個数が不正"},
		{"1d3+x", "ボーナスが不正"},
		{"1d0", "面数は1以上"},
		{"-1d6", "個数は1以上"},
		{"0d6", "個数は1以上"},  // d を書いたら個数1以上。定数は "5" と書く
		{"1D3", "数値が不正"},   // 大文字 D は受け付けない。d に統一する
		{" 2d4 ", "個数が不正"}, // 前後空白も厳密に弾く
		{"5 ", "数値が不正"},    // 定数も空白を許さない
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			_, err := ParseDice(tt.in)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.contains)
		})
	}
}

func TestDice_String_ParseDiceと往復する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		d    Dice
		want string
	}{
		{Dice{Base: 1, Sides: 3, Bonus: 1}, "1d3+1"},
		{Dice{Base: 2, Sides: 6}, "2d6"},
		{Dice{Base: 3, Sides: 4, Bonus: -1}, "3d4-1"},
		{Dice{Bonus: 5}, "5"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.d.String(), "正規化表記")
			back, err := ParseDice(tt.d.String())
			require.NoError(t, err)
			assert.Equal(t, tt.d, back, "String→ParseDice で元に戻る")
		})
	}
}

func TestDice_MinMax_取りうる範囲を返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		d        Dice
		min, max int
	}{
		{Dice{Base: 1, Sides: 3}, 1, 3},           // 1d3 は 1..3
		{Dice{Base: 1, Sides: 3, Bonus: 1}, 2, 4}, // 1d3+1 は 2..4
		{Dice{Base: 2, Sides: 6}, 2, 12},          // 2d6 は 2..12
		{Dice{Bonus: 5}, 5, 5},                    // 定数は 5..5
	}
	for _, tt := range tests {
		t.Run(tt.d.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.min, tt.d.Min(), "最小")
			assert.Equal(t, tt.max, tt.d.Max(), "最大")
		})
	}
}

func TestDice_Roll_MinとMaxの範囲に収まる(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(1, 2))
	d := MustParseDice("2d6")
	for range 1000 {
		got := d.Roll(rng)
		assert.GreaterOrEqual(t, got, d.Min())
		assert.LessOrEqual(t, got, d.Max())
	}
}

func TestDice_Roll_定数はBonusを返す(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(1, 2))
	d := MustParseDice("5")
	for range 10 {
		assert.Equal(t, 5, d.Roll(rng), "定数は常に Bonus")
	}
}

func TestDice_Roll_同じseedで同じ列を返す(t *testing.T) {
	t.Parallel()

	d := MustParseDice("3d6+1")
	roll := func() []int {
		rng := rand.New(rand.NewPCG(42, 7))
		out := make([]int, 20)
		for i := range out {
			out[i] = d.Roll(rng)
		}
		return out
	}
	first := roll()
	second := roll()
	assert.Equal(t, first, second, "同じ seed なら抽選列は決定的")
}

func TestDice_Roll_一様抽選と等価になる(t *testing.T) {
	t.Parallel()

	// 一様抽選 [a,b] は 1d(b-a+1)+(a-1) と等価。ここでは [2,4] = 1d3+1 を確認する
	d := MustParseDice("1d3+1")
	rng := rand.New(rand.NewPCG(1, 2))
	seen := map[int]bool{}
	for range 300 {
		seen[d.Roll(rng)] = true
	}
	assert.Equal(t, map[int]bool{2: true, 3: true, 4: true}, seen, "2..4 が一様に出る")
}
