package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDoorBehavior(t *testing.T) {
	t.Parallel()

	t.Run("閉じた扉を開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		// SpawnDoor は閉じた扉を実コンポーネント一式で生成する
		door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, gc.DoorOrientationHorizontal)
		require.NoError(t, err)

		result, err := Execute(NewOpenDoorActivity(door), player, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "扉を開くアクションが成功するべき")

		doorComp := world.Components.Door.Get(door)
		assert.True(t, doorComp.IsOpen, "扉が開いているべき")

		// BlockPassとBlockViewが削除されていることを確認
		assert.False(t, world.Components.BlockPass.Has(door), "BlockPassが削除されているべき")
		assert.False(t, world.Components.BlockView.Has(door), "BlockViewが削除されているべき")
	})

	t.Run("Doorコンポーネントがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 扉でないエンティティを対象にする
		wall := world.ECS.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.BlockPass.Add(wall, &gc.BlockPass{})

		result, err := Execute(NewOpenDoorActivity(wall), player, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success, "検証失敗で成功フラグがfalseであるべき")
		assert.Equal(t, gc.ActivityStateCanceled, result.State)
		assert.NotEmpty(t, result.Message)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// ゼロ値Entityは扉ではない
		result, err := Execute(NewOpenDoorActivity(gc.InvalidEntity), player, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success, "検証失敗で成功フラグがfalseであるべき")
		assert.Equal(t, gc.ActivityStateCanceled, result.State)
		assert.NotEmpty(t, result.Message)
	})
}

func TestCloseDoorBehavior(t *testing.T) {
	t.Parallel()

	// newOpenDoor はプレイヤーを扉の隣接マスに置き、doorCoord に開いた扉を用意する。
	// 本番と同じ spawn 関数で生成し、扉が実コンポーネント一式を持つ状態で検証する
	newOpenDoor := func(t *testing.T, world w.World, doorCoord consts.Coord[consts.Tile]) (player, door ecs.Entity) {
		t.Helper()
		var err error
		player, err = lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: doorCoord.X - 1, Y: doorCoord.Y}, "ash")
		require.NoError(t, err)
		door, err = lifecycle.SpawnDoor(world, doorCoord, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		// SpawnDoor は閉じた扉を作るため、閉扉を試すには開けておく
		require.NoError(t, lifecycle.OpenDoor(world, door))
		return player, door
	}

	t.Run("無人の扉は閉じられる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		doorCoord := consts.Coord[consts.Tile]{X: 11, Y: 10}
		player, door := newOpenDoor(t, world, doorCoord)

		result, err := Execute(NewCloseDoorActivity(door), player, world)

		require.NoError(t, err)
		assert.True(t, result.Success, "無人なら閉じられる。扉自身を占有物と誤検知しないこと")
		assert.False(t, world.Components.Door.Get(door).IsOpen, "扉が閉じているべき")
	})

	t.Run("扉のマスにキャラがいると閉じられない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		doorCoord := consts.Coord[consts.Tile]{X: 11, Y: 10}
		player, door := newOpenDoor(t, world, doorCoord)

		// 扉のマスに敵を置く。空間インデックスに反映させるため無効化する
		_, err := lifecycle.SpawnEnemy(world, doorCoord, "fireball")
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)

		result, err := Execute(NewCloseDoorActivity(door), player, world)

		require.NoError(t, err, "占有時は *UserError の no-op になる")
		assert.False(t, result.Success, "占有中は閉じられない")
		assert.Equal(t, gc.ActivityStateCanceled, result.State)
		assert.True(t, world.Components.Door.Get(door).IsOpen, "扉は開いたまま")
	})

	t.Run("扉のマスにアイテムがあると閉じられない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		doorCoord := consts.Coord[consts.Tile]{X: 11, Y: 10}
		player, door := newOpenDoor(t, world, doorCoord)

		// 扉のマスにフィールドアイテムを置く
		_, err := lifecycle.SpawnFieldItem(world, "bread", doorCoord.X, doorCoord.Y, 1)
		require.NoError(t, err)

		result, err := Execute(NewCloseDoorActivity(door), player, world)

		require.NoError(t, err, "占有時は *UserError の no-op になる")
		assert.False(t, result.Success, "アイテムがあると閉じられない")
		assert.Equal(t, gc.ActivityStateCanceled, result.State)
		assert.True(t, world.Components.Door.Get(door).IsOpen, "扉は開いたまま")
	})
}
