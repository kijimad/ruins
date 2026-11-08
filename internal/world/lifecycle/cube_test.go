package lifecycle_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCubeFuel(t *testing.T) {
	t.Parallel()
	// 石炭は1kgで熱量800
	full := consts.CubeFuelCapacityBase
	tests := []struct {
		name      string
		current   consts.Heat
		wantOK    bool
		wantLeft  consts.Heat
		wantAlive bool
	}{
		{"熱量を足してアイテムを消す", 100, true, 900, false},
		{"あふれた分は捨てて上限で止める", full - 100, true, full, false},
		{"満タンならくべずアイテムを残す", full, false, full, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
			require.NoError(t, err)
			world.Components.FuelTank.Get(cube).Current = tt.current
			_, err = lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
			require.NoError(t, err)
			coal, err := lifecycle.SpawnBackpackItem(world, "coal", 1)
			require.NoError(t, err)

			assert.Equal(t, tt.wantOK, lifecycle.AddCubeFuel(world, cube, coal))
			assert.Equal(t, tt.wantLeft, query.CubeFuelTotal(world, cube))
			assert.Equal(t, tt.wantAlive, world.ECS.Alive(coal))
		})
	}
}

func TestConsumeCubeFuel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		amount   consts.Heat
		wantOK   bool
		wantLeft consts.Heat
	}{
		{"足りれば必要量だけ減らす", 9, true, 991},
		{"ちょうどなら0まで減らす", 1000, true, 0},
		{"足りなければ減らさない", 1001, false, 1000},
		{"0以下は減らさず成功", 0, true, 1000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
			require.NoError(t, err)
			world.Components.FuelTank.Get(cube).Current = 1000

			assert.Equal(t, tt.wantOK, lifecycle.ConsumeCubeFuel(world, cube, tt.amount))
			assert.Equal(t, tt.wantLeft, query.CubeFuelTotal(world, cube))
		})
	}
}
