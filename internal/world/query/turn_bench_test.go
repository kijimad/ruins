package query_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/require"
)

// BenchmarkRestoreAllActionPoints は毎ターン終了時に走る全エンティティの AP 回復コストを計測する。
//
// この処理はカリング対象外で全 TurnBased エンティティに CalculateMaxActionPoints/CalculateSpeed を適用する。
func BenchmarkRestoreAllActionPoints(b *testing.B) {
	for _, n := range []int{100, 400, 1000} {
		b.Run(fmt.Sprintf("enemies=%d", n), func(b *testing.B) {
			world := testutil.InitTestWorld(b)
			testutil.SetStageLevel(world, gc.Level{TileWidth: consts.Tile(200), TileHeight: consts.Tile(200)})
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 100, Y: 100}, "Ash")
			require.NoError(b, err)

			rng := rand.New(rand.NewPCG(1, 2))
			for range n {
				_, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: consts.Tile(rng.IntN(200)), Y: consts.Tile(rng.IntN(200))}, "火の玉")
				require.NoError(b, err)
			}

			b.ResetTimer()
			for range b.N {
				require.NoError(b, query.RestoreAllActionPoints(world))
			}
		})
	}
}
