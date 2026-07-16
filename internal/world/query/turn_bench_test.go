package query_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/testscene"
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
			world, _ := testscene.InitDungeonWorld(b, 200, 100, 100)

			rng := rand.New(rand.NewPCG(1, 2))
			for range n {
				testscene.MustSpawnEnemy(b, world, rng.IntN(200), rng.IntN(200))
			}

			b.ResetTimer()
			for range b.N {
				require.NoError(b, query.RestoreAllActionPoints(world))
			}
		})
	}
}
