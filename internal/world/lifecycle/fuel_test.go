package lifecycle_test

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddFuel_床の灯油は中身だけ燃えて同じタイルに残量0で残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	oil, err := lifecycle.SpawnFieldItem(world, "lamp_oil", 3, 4, 1)
	require.NoError(t, err)
	// 灯油は満タンで熱 500。半分なら 250
	world.Components.Charge.Get(oil).Current = 750
	fire := world.ECS.NewEntity()
	world.Components.Burning.Add(fire, &gc.Burning{})
	want := consts.Heat(250).BurnTurns(query.BurnEfficiency(world, fire))

	lifecycle.AddFuel(world, fire, oil)

	assert.Equal(t, want, world.Components.Burning.Get(fire).Remaining, "熱は燃やす前の残量で測る")
	require.True(t, world.ECS.Alive(oil))
	assert.Equal(t, 0, world.Components.Charge.Get(oil).Current)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 3, Y: 4}, world.Components.GridElement.Get(oil).Coord)
}

func TestAddCubeFuel_灯油は残量の割合の熱を足して残量0で残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	world.Components.Charge.Get(cube).Current = 0
	_, err = lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(oil).Current = 750

	require.True(t, lifecycle.AddCubeFuel(world, cube, oil))

	// 灯油は満タンで熱 500。半分なら 250
	assert.Equal(t, consts.Heat(250), query.CubeFuelTotal(world, cube))
	require.True(t, world.ECS.Alive(oil))
	assert.Equal(t, 0, world.Components.Charge.Get(oil).Current)
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
