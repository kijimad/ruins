package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
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

		// プレイヤーを作成
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		// 扉を作成（閉じている）
		door := world.ECS.NewEntity()
		world.Components.Door.Add(door, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationHorizontal})
		world.Components.GridElement.Add(door, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.BlockPass.Add(door, &gc.BlockPass{})
		world.Components.BlockView.Add(door, &gc.BlockView{})

		// OpenDoorBehaviorを実行
		result, err := Execute(NewOpenDoorActivity(door), player, world)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Success, "扉を開くアクションが成功するべき")

		// 扉が開いていることを確認
		doorComp := world.Components.Door.Get(door)
		assert.True(t, doorComp.IsOpen, "扉が開いているべき")

		// BlockPassとBlockViewが削除されていることを確認
		assert.False(t, world.Components.BlockPass.Has(door), "BlockPassが削除されているべき")
		assert.False(t, world.Components.BlockView.Has(door), "BlockViewが削除されているべき")

		world.ECS.RemoveEntity(player)
		world.ECS.RemoveEntity(door)
	})

	t.Run("Doorコンポーネントがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		// 普通の壁を作成（Doorコンポーネントなし）
		wall := world.ECS.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.BlockPass.Add(wall, &gc.BlockPass{})

		// OpenDoorBehaviorを実行
		result, err := Execute(NewOpenDoorActivity(wall), player, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success, "検証失敗で成功フラグがfalseであるべき")
		assert.Equal(t, gc.ActivityStateCanceled, result.State)
		assert.NotEmpty(t, result.Message)

		world.ECS.RemoveEntity(player)
		world.ECS.RemoveEntity(wall)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		// OpenDoorを実行（ゼロ値Entityは扉ではない）
		result, err := Execute(NewOpenDoorActivity(gc.InvalidEntity), player, world)

		require.Error(t, err)
		require.NotNil(t, result)
		assert.False(t, result.Success, "検証失敗で成功フラグがfalseであるべき")
		assert.Equal(t, gc.ActivityStateCanceled, result.State)
		assert.NotEmpty(t, result.Message)

		world.ECS.RemoveEntity(player)
	})
}

func TestCloseDoorBehavior(t *testing.T) {
	t.Parallel()

	// newOpenDoor はプレイヤーを扉の隣接マスに、開いた扉を doorCoord に置く
	newOpenDoor := func(world w.World, doorCoord consts.Coord[consts.Tile]) (player, door ecs.Entity) {
		player = world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: doorCoord.X - 1, Y: doorCoord.Y}})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		door = world.ECS.NewEntity()
		world.Components.Door.Add(door, &gc.Door{IsOpen: true, Orientation: gc.DoorOrientationHorizontal})
		world.Components.GridElement.Add(door, &gc.GridElement{Coord: doorCoord})
		// 実スポーンに合わせて扉自身も LocationOnField と Fixed を持つ。
		// 扉が自分を占有物と誤検知しないことを無人ケースで担保する
		world.Components.LocationOnField.Add(door, &gc.LocationOnField{})
		world.Components.Fixed.Add(door, &gc.Fixed{})
		return player, door
	}

	t.Run("無人の扉は閉じられる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		doorCoord := consts.Coord[consts.Tile]{X: 11, Y: 10}
		player, door := newOpenDoor(world, doorCoord)

		result, err := Execute(NewCloseDoorActivity(door), player, world)

		require.NoError(t, err)
		assert.True(t, result.Success, "無人なら閉じられる")
		assert.False(t, world.Components.Door.Get(door).IsOpen, "扉が閉じているべき")
	})

	t.Run("扉のマスにキャラがいると閉じられない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		doorCoord := consts.Coord[consts.Tile]{X: 11, Y: 10}
		player, door := newOpenDoor(world, doorCoord)

		// 扉のマスに敵を置く。空間インデックスに反映させるため無効化する
		enemy := world.ECS.NewEntity()
		world.Components.SoloAI.Add(enemy, &gc.SoloAI{})
		world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: doorCoord})
		query.InvalidateSpatialIndex(world)

		result, err := Execute(NewCloseDoorActivity(door), player, world)

		require.NoError(t, err, "ユーザ起因の失敗は err=nil の no-op になる")
		assert.False(t, result.Success, "占有中は閉じられない")
		assert.Equal(t, gc.ActivityStateCanceled, result.State)
		assert.True(t, world.Components.Door.Get(door).IsOpen, "扉は開いたまま")
	})

	t.Run("扉のマスにアイテムがあると閉じられない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		doorCoord := consts.Coord[consts.Tile]{X: 11, Y: 10}
		player, door := newOpenDoor(world, doorCoord)

		// 扉のマスにフィールドアイテムを置く
		item := world.ECS.NewEntity()
		world.Components.LocationOnField.Add(item, &gc.LocationOnField{})
		world.Components.GridElement.Add(item, &gc.GridElement{Coord: doorCoord})

		result, err := Execute(NewCloseDoorActivity(door), player, world)

		require.NoError(t, err)
		assert.False(t, result.Success, "アイテムがあると閉じられない")
		assert.Equal(t, gc.ActivityStateCanceled, result.State)
		assert.True(t, world.Components.Door.Get(door).IsOpen, "扉は開いたまま")
	})
}
