package lifecycle_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockAllDoors(t *testing.T) {
	t.Parallel()

	t.Run("全扉を閉じてロックする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door1, err := lifecycle.SpawnDoor(world, 5, 5, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		door2, err := lifecycle.SpawnDoor(world, 6, 6, gc.DoorOrientationVertical)
		require.NoError(t, err)

		locked := lifecycle.LockAllDoors(world)

		assert.Equal(t, 2, locked)
		assert.True(t, world.Components.Door.Get(door1).Locked)
		assert.True(t, world.Components.Door.Get(door2).Locked)
	})

	t.Run("開いた扉を閉じてからロックする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := lifecycle.SpawnDoor(world, 5, 5, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		require.NoError(t, lifecycle.OpenDoor(world, door))

		doorComp := world.Components.Door.Get(door)
		assert.True(t, doorComp.IsOpen)

		locked := lifecycle.LockAllDoors(world)

		assert.Equal(t, 1, locked)
		// LockAllDoors内のCloseDoorがarchetypeを変えるため取り直して検証する
		doorComp = world.Components.Door.Get(door)
		assert.False(t, doorComp.IsOpen, "扉が閉じられるべき")
		assert.True(t, doorComp.Locked, "扉がロックされるべき")
		// ロックした閉扉は視線を遮る（BlockView）べき。ボス部屋の扉が視線を通す不具合の回帰
		assert.True(t, world.Components.BlockView.Has(door), "ロックした閉扉はBlockViewを持つべき")
		// BlockView変化を視界システムへ通知するため視界更新フラグが立つべき
		assert.True(t, query.GetVisionState(world).NeedsForceUpdate, "視界の再計算が要求されるべき")
	})

	t.Run("既にロック済みの扉はスキップする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := lifecycle.SpawnDoor(world, 5, 5, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		world.Components.Door.Get(door).Locked = true

		locked := lifecycle.LockAllDoors(world)

		assert.Equal(t, 0, locked)
	})

	t.Run("扉がない場合は0を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		locked := lifecycle.LockAllDoors(world)

		assert.Equal(t, 0, locked)
	})
}

func TestUnlockAllDoors(t *testing.T) {
	t.Parallel()

	t.Run("全扉をアンロックして開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door1, err := lifecycle.SpawnDoor(world, 5, 5, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		door2, err := lifecycle.SpawnDoor(world, 6, 6, gc.DoorOrientationVertical)
		require.NoError(t, err)

		// ロックする
		world.Components.Door.Get(door1).Locked = true
		world.Components.Door.Get(door2).Locked = true

		opened := lifecycle.UnlockAllDoors(world)

		assert.Equal(t, 2, opened)
		doorComp1 := world.Components.Door.Get(door1)
		doorComp2 := world.Components.Door.Get(door2)
		assert.False(t, doorComp1.Locked)
		assert.True(t, doorComp1.IsOpen)
		assert.False(t, doorComp2.Locked)
		assert.True(t, doorComp2.IsOpen)
	})

	t.Run("既に開いている扉はカウントしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := lifecycle.SpawnDoor(world, 5, 5, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		require.NoError(t, lifecycle.OpenDoor(world, door))
		world.Components.Door.Get(door).Locked = true

		opened := lifecycle.UnlockAllDoors(world)

		assert.Equal(t, 0, opened)
		doorComp := world.Components.Door.Get(door)
		assert.False(t, doorComp.Locked, "アンロックされるべき")
		assert.True(t, doorComp.IsOpen, "開いたままであるべき")
	})
}

// TestSpawnDungeonEntrance_ダンジョンポータルと同じアニメフレームを持つ は、オーバーワールドの
// 遺跡入口がダンジョン内の階段ポータルと同じ回転アニメを持つことを固定する。入口はコードで
// 組むため、以前はアニメフレーム AnimKeys が抜けて静止していた。raw の warp_next を流用して揃える。
func TestSpawnDungeonEntrance_ダンジョンポータルと同じアニメフレームを持つ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e, err := lifecycle.SpawnDungeonEntrance(world, 5, 5, "亡者の森")
	require.NoError(t, err)

	require.True(t, world.Components.SpriteRender.Has(e), "スプライトを持つ")
	assert.NotEmpty(t, world.Components.SpriteRender.Get(e).AnimKeys, "入口はアニメフレームを持ちアニメーションする")

	// 相互作用は遺跡進入で、warp_next 由来の次階ポータルではない
	require.True(t, world.Components.Interactable.Has(e), "相互作用を持つ")
	assert.Contains(t, world.Components.Interactable.Get(e).Interactions, gc.InteractionDungeonEnter, "遺跡進入の相互作用")

	// オーバーワールドの地物として帯へ束縛される
	require.True(t, world.Components.StageBound.Has(e), "ステージへ束縛される")
	assert.Equal(t, gc.NewOverworldStage(), world.Components.StageBound.Get(e).Key, "オーバーワールド帯へ束縛される")

	// 遺跡定義名を運ぶ
	require.True(t, world.Components.DungeonEntrance.Has(e), "遺跡入口コンポーネントを持つ")
	assert.Equal(t, "亡者の森", world.Components.DungeonEntrance.Get(e).DefinitionName, "進入先の遺跡定義名を運ぶ")
}
