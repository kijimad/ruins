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
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/require"
)

// BenchmarkFindNearest は全走査とインデックス経由を、マップ上のタイル数を変えて比較する。
//
// 本作は床・壁も1タイル=1エンティティのため、全走査はマップ全タイルを毎回舐める。
// キャラクター探索はインデックスの Characters に絞ることで、タイル数に依存しなくなる。
func BenchmarkFindNearest(b *testing.B) {
	for _, tiles := range []int{0, 2500} {
		world := testutil.InitTestWorld(b)
		d := world.Components.Dungeon.Get(world.Resources.SingletonEntity)
		d.Level = gc.Level{TileWidth: consts.Tile(200), TileHeight: consts.Tile(200)}
		_, err := lifecycle.SpawnPlayer(world, 100, 100, "Ash")
		require.NoError(b, err)

		// キャラクター（敵）を散らす
		rng := rand.New(rand.NewPCG(1, 2))
		var self ecs.Entity
		for i := range 20 {
			e, err := lifecycle.SpawnEnemy(world, rng.IntN(200), rng.IntN(200), "火の玉")
			require.NoError(b, err)
			if i == 0 {
				self = e
			}
		}

		// 床タイルを模した GridElement エンティティ（キャラクターではない＝インデックス Characters に入らない）
		for range tiles {
			world.Components.GridElement.NewEntity(&gc.GridElement{
				X: consts.Tile(rng.IntN(200)), Y: consts.Tile(rng.IntN(200)),
			})
		}

		selfGrid := world.Components.GridElement.Get(self)
		match := func(e ecs.Entity) bool {
			return world.Components.SoloAI.Has(e) || world.Components.Player.Has(e)
		}
		// インデックスを構築しておく
		require.NotNil(b, query.GetSpatialIndex(world))

		b.Run(fmt.Sprintf("scan/tiles=%d", tiles), func(b *testing.B) {
			for range b.N {
				query.FindNearestEntity(world, self, selfGrid, match)
			}
		})
		b.Run(fmt.Sprintf("index/tiles=%d", tiles), func(b *testing.B) {
			for range b.N {
				query.FindNearestCharacter(world, self, selfGrid, match)
			}
		})
	}
}
