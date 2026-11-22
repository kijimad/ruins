package overworld

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
)

func TestNewLootChargeRNG_同じ種のほかの流れと一致しない(t *testing.T) {
	t.Parallel()
	const seed uint64 = 12345
	draws := func(r *rand.Rand) []uint64 {
		out := make([]uint64, 4)
		for i := range out {
			out[i] = r.Uint64()
		}
		return out
	}
	charge := draws(lifecycle.NewLootChargeRNG(seed, 0x4))
	for _, other := range []*rand.Rand{
		rand.New(rand.NewPCG(seed, 0x4)),
		rand.New(rand.NewPCG(seed, 0x5)),
		rand.New(rand.NewPCG(seed, 0x6)),
		rand.New(rand.NewPCG(seed^outdoorLootItemChannel, 0x4)),
	} {
		assert.NotEqual(t, draws(other), charge)
	}
}
