package lifecycle_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
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
		assert.False(t, doorComp.IsOpen, "扉が閉じられるべき")
		assert.True(t, doorComp.Locked, "扉がロックされるべき")
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
