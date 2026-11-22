package lifecycle_test

import (
	"math/rand/v2"
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

func TestSpawnFieldLoot_燃料と灯りは残量が4段階のどれかになる(t *testing.T) {
	t.Parallel()
	tests := []struct {
		rawID string
		want  []int
	}{
		{rawID: "battery", want: []int{250, 500, 750, 1000}},
		{rawID: "flashlight", want: []int{250, 500, 750, 1000}},
		{rawID: "lamp_oil", want: []int{375, 750, 1125, 1500}},
	}
	for _, tt := range tests {
		t.Run(tt.rawID, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)

			require.NoError(t, lifecycle.SpawnFieldLoot(world, tt.rawID, 3, 4, 40, rand.New(rand.NewPCG(1, 2))))

			seen := map[int]bool{}
			q := ecs.NewFilter1[gc.Charge](world.ECS).Query()
			for q.Next() {
				seen[q.Get().Current] = true
			}
			got := make([]int, 0, len(seen))
			for v := range seen {
				got = append(got, v)
			}
			assert.ElementsMatch(t, tt.want, got, "40個がそれぞれ別に引かれ、4段階がすべて出る")
		})
	}
}

func TestSpawnFieldLoot_同じ場所に先に置いた物は書き換えない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	first, err := lifecycle.SpawnFieldItem(world, "battery", 3, 4, 1)
	require.NoError(t, err)

	require.NoError(t, lifecycle.SpawnFieldLoot(world, "battery", 3, 4, 20, rand.New(rand.NewPCG(1, 2))))

	assert.Equal(t, 1000, world.Components.Charge.Get(first).Current)
}

func TestSpawnFieldLoot_燃料を受けない物は満タンのまま(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	require.NoError(t, lifecycle.SpawnFieldLoot(world, "torch", 3, 4, 5, rand.New(rand.NewPCG(1, 2))))

	var got []int
	q := ecs.NewFilter1[gc.Charge](world.ECS).Query()
	for q.Next() {
		got = append(got, q.Get().Current)
	}
	assert.Equal(t, []int{600, 600, 600, 600, 600}, got)
}

func TestSpawnStorageLoot_同じ乱数なら同じ残量になる(t *testing.T) {
	t.Parallel()
	charges := func() []int {
		world := testutil.InitTestWorld(t)
		storage := world.ECS.NewEntity()
		require.NoError(t, lifecycle.SpawnStorageLoot(world, "battery", 6, storage, rand.New(rand.NewPCG(7, 8))))
		var got []int
		q := ecs.NewFilter1[gc.Charge](world.ECS).Query()
		for q.Next() {
			got = append(got, q.Get().Current)
		}
		return got
	}

	first, second := charges(), charges()
	assert.Len(t, first, 6)
	assert.Equal(t, first, second)
}

func TestEmptyFuel_置き場所の無い燃料は空の物を作らずに消す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	owner := world.ECS.NewEntity()
	oil, err := lifecycle.SpawnFieldItem(world, "lamp_oil", 3, 4, 1)
	require.NoError(t, err)
	lifecycle.MoveToEquip(world, oil, owner, gc.SlotHand1)

	require.True(t, lifecycle.EmptyFuel(world, oil))

	assert.False(t, world.ECS.Alive(oil))
	var ids []string
	q := ecs.NewFilter1[gc.RawID](world.ECS).Query()
	for q.Next() {
		ids = append(ids, world.Components.RawID.Get(q.Entity()).ID)
	}
	assert.Empty(t, ids)
}
