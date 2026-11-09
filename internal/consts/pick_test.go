package consts

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedRoll float64

func (f fixedRoll) Float64() float64 { return float64(f) }

func identity(w float64) float64 { return w }

func TestPick(t *testing.T) {
	t.Parallel()
	weights := []float64{1, 0, 3}
	tests := []struct {
		name string
		roll float64
		want int
	}{
		{"先頭の重みの範囲は先頭", 0, 0},
		{"先頭の重みの境界は次の正の重み", 0.25, 2},
		{"末尾の重みの範囲は末尾", 0.99, 2},
		{"丸めで合計に届いても末尾の正の重み", 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := Pick(fixedRoll(tt.roll), weights, identity)
			require.True(t, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPick_選べないとfalse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		weights []float64
	}{
		{"空", nil},
		{"重みの合計が0", []float64{0, 0}},
		{"負の重みだけ", []float64{-1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, ok := Pick(fixedRoll(0.5), tt.weights, identity)
			assert.False(t, ok)
		})
	}
}

func TestPick_重みの比で選ぶ(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(1, 2))
	counts := make([]int, 2)
	const n = 10000
	for range n {
		i, ok := Pick(rng, []float64{1, 3}, identity)
		require.True(t, ok)
		counts[i]++
	}
	assert.InDelta(t, 0.25, float64(counts[0])/n, 0.02)
}

func TestPickUniform(t *testing.T) {
	t.Parallel()
	got, ok := PickUniform(fixedRoll(0.5), []string{"a", "b", "c", "d"})
	require.True(t, ok)
	assert.Equal(t, 2, got)

	_, ok = PickUniform(fixedRoll(0.5), []string{})
	assert.False(t, ok)
}

func TestHashRoll_Float64(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.0, HashRoll(0).Float64())
	assert.Equal(t, 0.5, HashRoll(1<<63).Float64())
	assert.Less(t, HashRoll(^uint64(0)).Float64(), 1.0)
}
