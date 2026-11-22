package lifecycle

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomizeLootCharge_上限が小さくても残量0にしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	items, err := spawnFieldItems(world, "battery", 3, 4, 20)
	require.NoError(t, err)
	for _, item := range items {
		world.Components.Charge.Get(item).Max = 2
	}

	randomizeLootCharge(world, items, rand.New(rand.NewPCG(1, 2)))

	seen := map[int]bool{}
	for _, item := range items {
		seen[world.Components.Charge.Get(item).Current] = true
	}
	assert.Equal(t, map[int]bool{1: true, 2: true}, seen, "上限2の 1/4 は切り捨てで0だが1に上げる")
}
