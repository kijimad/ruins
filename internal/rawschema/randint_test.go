package rawschema_test

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestRandInt_Rollは基本の数から幅未満を足した範囲を引く(t *testing.T) {
	t.Parallel()
	r := rawschema.RandInt{Base: 5, Spread: 4}
	rng := rand.New(rand.NewPCG(1, 0))
	seen := map[int]bool{}
	for range 200 {
		seen[r.Roll(rng)] = true
	}
	assert.Equal(t, map[int]bool{5: true, 6: true, 7: true, 8: true}, seen)
}

func TestRandInt_Meanは基本の数に乱数の幅の平均を足す(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 10.0, rawschema.RandInt{Base: 8, Spread: 5}.Mean())
	assert.Equal(t, 6.5, rawschema.RandInt{Base: 5, Spread: 4}.Mean())
}
