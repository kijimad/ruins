package raw

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectByWeightFunc_Empty(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(42, 0))
	result, err := SelectByWeightFunc(
		[]string{},
		func(_ string) float64 { return 1.0 },
		func(s string) string { return s },
		rng,
	)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestSelectByWeightFunc_AllZeroWeight(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(42, 0))
	result, err := SelectByWeightFunc(
		[]string{"a", "b"},
		func(_ string) float64 { return 0 },
		func(s string) string { return s },
		rng,
	)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestSelectByWeightFunc_SingleItem(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(42, 0))
	result, err := SelectByWeightFunc(
		[]string{"only"},
		func(_ string) float64 { return 1.0 },
		func(s string) string { return s },
		rng,
	)
	assert.NoError(t, err)
	assert.Equal(t, "only", result)
}

func TestSelectByWeightFunc_WeightedDistribution(t *testing.T) {
	t.Parallel()

	type item struct {
		name   string
		weight float64
	}
	items := []item{
		{"heavy", 100},
		{"light", 1},
	}

	rng := rand.New(rand.NewPCG(42, 0))
	counts := map[string]int{}
	for range 1000 {
		result, err := SelectByWeightFunc(
			items,
			func(it item) float64 { return it.weight },
			func(it item) string { return it.name },
			rng,
		)
		assert.NoError(t, err)
		counts[result]++
	}

	// 重み100:1なので期待値は約990回。シード固定の決定論的テストのため閾値900は十分な余裕がある
	assert.Greater(t, counts["heavy"], 900, "重みの大きい要素が多く選ばれる")
}

func TestPtrSlice(t *testing.T) {
	t.Parallel()

	t.Run("nilポインタ", func(t *testing.T) {
		t.Parallel()
		result := PtrSlice[int](nil)
		assert.Nil(t, result)
	})

	t.Run("有効なポインタ", func(t *testing.T) {
		t.Parallel()
		s := []int{1, 2, 3}
		result := PtrSlice(&s)
		assert.Equal(t, []int{1, 2, 3}, result)
	})

	t.Run("空スライスのポインタ", func(t *testing.T) {
		t.Parallel()
		s := []int{}
		result := PtrSlice(&s)
		assert.Empty(t, result)
	})
}
