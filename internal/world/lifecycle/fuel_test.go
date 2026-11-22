package lifecycle_test

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

func TestEmptyFuel_置き場所を保って空の物に置き換える(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		spawn func(world w.World, owner ecs.Entity) (ecs.Entity, error)
		check func(t *testing.T, world w.World, owner, empty ecs.Entity)
	}{
		{name: "持ち物", spawn: func(world w.World, _ ecs.Entity) (ecs.Entity, error) {
			return lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
		}, check: func(t *testing.T, world w.World, _, empty ecs.Entity) {
			t.Helper()
			player, err := query.GetPlayerEntity(world)
			require.NoError(t, err)
			assert.Equal(t, player, world.Components.LocationInBackpack.Get(empty).Owner)
		}},
		{name: "収納", spawn: func(world w.World, owner ecs.Entity) (ecs.Entity, error) {
			return lifecycle.SpawnStorageItem(world, "lamp_oil", 1, owner)
		}, check: func(t *testing.T, world w.World, owner, empty ecs.Entity) {
			t.Helper()
			assert.Equal(t, owner, world.Components.LocationInStorage.Get(empty).Owner)
		}},
		{name: "床", spawn: func(world w.World, _ ecs.Entity) (ecs.Entity, error) {
			oil, err := lifecycle.SpawnFieldItem(world, "lamp_oil", 3, 4, 1)
			if err == nil {
				world.Components.StageBound.Add(oil, &gc.StageBound{Key: gc.StageKey{Name: "camp"}})
			}
			return oil, err
		}, check: func(t *testing.T, world w.World, _, empty ecs.Entity) {
			t.Helper()
			assert.True(t, world.Components.LocationOnField.Has(empty))
			assert.Equal(t, consts.Coord[consts.Tile]{X: 3, Y: 4}, world.Components.GridElement.Get(empty).Coord)
			assert.Equal(t, gc.StageKey{Name: "camp"}, world.Components.StageBound.Get(empty).Key)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			storage := world.ECS.NewEntity()
			oil, err := tt.spawn(world, storage)
			require.NoError(t, err)

			require.True(t, lifecycle.EmptyFuel(world, oil))

			assert.False(t, world.ECS.Alive(oil))
			var empties []ecs.Entity
			q := ecs.NewFilter1[gc.RawID](world.ECS).Query()
			for q.Next() {
				if world.Components.RawID.Get(q.Entity()).ID == "empty_oil_can" {
					empties = append(empties, q.Entity())
				}
			}
			require.Len(t, empties, 1)
			tt.check(t, world, storage, empties[0])
		})
	}
}

func TestEmptyFuel_燃料でない物には何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	lantern, err := lifecycle.SpawnFieldItem(world, "hand_lantern", 3, 4, 1)
	require.NoError(t, err)

	assert.False(t, lifecycle.EmptyFuel(world, lantern))
	assert.True(t, world.ECS.Alive(lantern))
}

func TestAddFuel_床の灯油に火をくべると同じタイルに空の缶が残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	oil, err := lifecycle.SpawnFieldItem(world, "lamp_oil", 3, 4, 1)
	require.NoError(t, err)
	world.Components.Charge.Get(oil).Current = 750
	fire := world.ECS.NewEntity()
	world.Components.Burning.Add(fire, &gc.Burning{})
	require.Equal(t, consts.Heat(250), query.HeatContent(world, oil))
	want := consts.Heat(250).BurnTurns(query.BurnEfficiency(world, fire))

	lifecycle.AddFuel(world, fire, oil)

	assert.Equal(t, want, world.Components.Burning.Get(fire).Remaining, "熱は置き換える前の灯油で測る")
	assert.False(t, world.ECS.Alive(oil))
	var cans []consts.Coord[consts.Tile]
	q := ecs.NewFilter2[gc.LocationOnField, gc.RawID](world.ECS).Query()
	for q.Next() {
		if world.Components.RawID.Get(q.Entity()).ID == "empty_oil_can" {
			cans = append(cans, world.Components.GridElement.Get(q.Entity()).Coord)
		}
	}
	assert.Equal(t, []consts.Coord[consts.Tile]{{X: 3, Y: 4}}, cans)
}

func TestAddCubeFuel_灯油は残量の割合の熱を足して空の缶を残す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	world.Components.FuelTank.Get(cube).Current = 0
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(oil).Current = 750

	require.True(t, lifecycle.AddCubeFuel(world, cube, oil))

	// 灯油は満タンで熱 500。半分なら 250
	assert.Equal(t, consts.Heat(250), query.CubeFuelTotal(world, cube))
	assert.False(t, world.ECS.Alive(oil))
	_, ok := query.FindInBackpack(world, player, "empty_oil_can")
	assert.True(t, ok)
}
