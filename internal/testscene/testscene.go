// Package testscene はテスト/ベンチ用のダンジョンシーン構築ヘルパを提供する。
// InitTestWorld（testutil）と lifecycle の両方に依存するため、循環を避けて両者と別の
// leaf パッケージに置く（testutil 自体は lifecycle の内部テストから使われるため
// testutil に lifecycle を import させられない）。
package testscene

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/require"
)

// InitDungeonWorld は mapSize×mapSize のマップと (playerX,playerY) のプレイヤーを持つ
// テスト/ベンチ用ワールドを返す。AI・移動・描画まわりのシーン構築の定型を集約する。
func InitDungeonWorld(tb testing.TB, mapSize, playerX, playerY int) (w.World, ecs.Entity) {
	tb.Helper()

	world := testutil.InitTestWorld(tb)
	d := world.Components.Dungeon.Get(world.Resources.SingletonEntity)
	d.Level = gc.Level{TileWidth: consts.Tile(mapSize), TileHeight: consts.Tile(mapSize)}

	player, err := lifecycle.SpawnPlayer(world, playerX, playerY, "Ash")
	require.NoError(tb, err)

	return world, player
}

// MustSpawnEnemy はテスト用の敵（"火の玉"）を1体生成する。エラー時は tb を失敗させる
func MustSpawnEnemy(tb testing.TB, world w.World, x, y int) ecs.Entity {
	tb.Helper()

	e, err := lifecycle.SpawnEnemy(world, x, y, "火の玉")
	require.NoError(tb, err)
	return e
}
