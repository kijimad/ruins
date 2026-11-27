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

		// 通れて見通せるようになっていることを確認
		assert.Equal(t, &gc.Passage{Passable: true}, world.Components.Passage.Get(door), "開いた扉は移動コスト無しで通れる")
		assert.True(t, world.Components.Visibility.Get(door).SeeThrough, "開いた扉は見通せる")
	})

	t.Run("Doorコンポーネントがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 扉でないエンティティを対象にする
		wall := world.ECS.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.Tile.Add(wall, &gc.Tile{})
		world.Components.Passage.Add(wall, &gc.Passage{})
		world.Components.Visibility.Add(wall, &gc.Visibility{})

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

	t.Run("Paramsの型が違う場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		odb := &OpenDoorBehavior{}
		comp := &gc.Activity{BehaviorName: gc.BehaviorOpenDoor, Params: &gc.CloseDoorParams{}}

		err := odb.Validate(comp, gc.InvalidEntity, world)

		require.ErrorIs(t, err, ErrParamsTypeMismatch)
	})
}

func TestOpenDoorBehavior_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("Paramsの型が違う場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		odb := &OpenDoorBehavior{}
		comp := &gc.Activity{BehaviorName: gc.BehaviorOpenDoor, Params: &gc.CloseDoorParams{}}

		err := odb.DoTurn(comp, gc.InvalidEntity, world)

		require.ErrorIs(t, err, ErrParamsTypeMismatch)
	})

	t.Run("対象がDoorコンポーネントを持たない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Validate を経由せず、Door を持たない対象で直接呼ぶ
		wall := world.ECS.NewEntity()
		odb := &OpenDoorBehavior{}
		comp := &gc.Activity{BehaviorName: gc.BehaviorOpenDoor, Params: &gc.OpenDoorParams{Target: wall}}

		err := odb.DoTurn(comp, gc.InvalidEntity, world)

		require.ErrorIs(t, err, errDoorTargetNotDoor)
	})

	t.Run("既に開いている扉に対しては何もせず完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		require.NoError(t, lifecycle.OpenDoor(world, door))
		// 準備で立った視界更新要求を捨て、DoTurn 自体が要求するかだけを見る
		query.GetVisionState(world).ConsumePendingUpdate()

		odb := &OpenDoorBehavior{}
		comp := &gc.Activity{BehaviorName: gc.BehaviorOpenDoor, Params: &gc.OpenDoorParams{Target: door}}

		err = odb.DoTurn(comp, gc.InvalidEntity, world)

		require.NoError(t, err)
		assert.True(t, world.Components.Door.Get(door).IsOpen, "既に開いた扉は開いたままであるべき")
		assert.True(t, IsCompleted(comp), "二重に開く操作でも完了扱いになるべき")
		assert.False(t, query.GetVisionState(world).ConsumePendingUpdate(), "状態が変わらなければ視界の再計算を要求しない")
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
		assert.Equal(t, query.T(world, "Something is in the doorway."), result.Message, "占有理由のメッセージが出る")
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
		assert.Equal(t, query.T(world, "Something is in the doorway."), result.Message, "占有理由のメッセージが出る")
		assert.True(t, world.Components.Door.Get(door).IsOpen, "扉は開いたまま")
	})

	t.Run("Paramsの型が違う場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		cdb := &CloseDoorBehavior{}
		comp := &gc.Activity{BehaviorName: gc.BehaviorCloseDoor, Params: &gc.OpenDoorParams{}}

		err := cdb.Validate(comp, gc.InvalidEntity, world)

		require.ErrorIs(t, err, ErrParamsTypeMismatch)
	})

	t.Run("対象がゼロ値エンティティの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		cdb := &CloseDoorBehavior{}
		comp := &gc.Activity{BehaviorName: gc.BehaviorCloseDoor, Params: &gc.CloseDoorParams{Target: gc.InvalidEntity}}

		err := cdb.Validate(comp, gc.InvalidEntity, world)

		require.ErrorIs(t, err, errDoorTargetNotAlive)
	})

	t.Run("対象がDoorコンポーネントを持たない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		wall := world.ECS.NewEntity()
		cdb := &CloseDoorBehavior{}
		comp := &gc.Activity{BehaviorName: gc.BehaviorCloseDoor, Params: &gc.CloseDoorParams{Target: wall}}

		err := cdb.Validate(comp, gc.InvalidEntity, world)

		require.ErrorIs(t, err, errDoorTargetNotDoor)
	})
}

func TestCloseDoorBehavior_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("Paramsの型が違う場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		cdb := &CloseDoorBehavior{}
		comp := &gc.Activity{BehaviorName: gc.BehaviorCloseDoor, Params: &gc.OpenDoorParams{}}

		err := cdb.DoTurn(comp, gc.InvalidEntity, world)

		require.ErrorIs(t, err, ErrParamsTypeMismatch)
	})

	t.Run("対象がDoorコンポーネントを持たない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		wall := world.ECS.NewEntity()
		cdb := &CloseDoorBehavior{}
		comp := &gc.Activity{BehaviorName: gc.BehaviorCloseDoor, Params: &gc.CloseDoorParams{Target: wall}}

		err := cdb.DoTurn(comp, gc.InvalidEntity, world)

		require.ErrorIs(t, err, errDoorTargetNotDoor)
	})

	t.Run("既に閉じている扉に対しては何もせず完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// SpawnDoor は閉じた扉を作る
		door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		// 準備で立った視界更新要求を捨て、DoTurn 自体が要求するかだけを見る
		query.GetVisionState(world).ConsumePendingUpdate()

		cdb := &CloseDoorBehavior{}
		comp := &gc.Activity{BehaviorName: gc.BehaviorCloseDoor, Params: &gc.CloseDoorParams{Target: door}}

		err = cdb.DoTurn(comp, gc.InvalidEntity, world)

		require.NoError(t, err)
		assert.False(t, world.Components.Door.Get(door).IsOpen, "既に閉じた扉は閉じたままであるべき")
		assert.True(t, IsCompleted(comp), "二重に閉じる操作でも完了扱いになるべき")
		assert.False(t, query.GetVisionState(world).ConsumePendingUpdate(), "状態が変わらなければ視界の再計算を要求しない")
	})
}
