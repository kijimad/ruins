package query_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/require"
)

// BenchmarkRestoreAllActionPoints は毎ターン終了時に走る全エンティティの AP 回復コストを計測する。
//
// この処理はカリング対象外で全 TurnBased エンティティに CalculateMaxActionPoints/CalculateSpeed を
// 適用する。走行中（ProcessAll がカリングで軽くなる状況）では相対的に支配的な per-turn コストに
// なり得るため、敵総数に対するスケールを可視化する。
func BenchmarkRestoreAllActionPoints(b *testing.B) {
	for _, n := range []int{100, 400, 1000} {
		b.Run(fmt.Sprintf("enemies=%d", n), func(b *testing.B) {
			world := testutil.InitTestWorld(b)
			d := query.GetDungeon(world)
			d.Level = gc.Level{TileWidth: 200, TileHeight: 200}

			_, err := lifecycle.SpawnPlayer(world, 100, 100, "Ash")
			require.NoError(b, err)

			rng := rand.New(rand.NewPCG(1, 2))
			for range n {
				_, err := lifecycle.SpawnEnemy(world, rng.IntN(200), rng.IntN(200), "火の玉")
				require.NoError(b, err)
			}

			b.ResetTimer()
			for range b.N {
				require.NoError(b, query.RestoreAllActionPoints(world))
			}
		})
	}
}
