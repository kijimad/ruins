package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployedFuelSystem(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		deployed  bool
		suspended bool
		turn      consts.Turn
		missing   int
		want      int
	}{
		{"展開中は間隔のターンに1戻る", true, false, 20, 5, 1},
		{"経過ターン0も間隔のターンに数える", true, false, 0, 5, 1},
		{"間隔でないターンは戻らない", true, false, 21, 5, 0},
		{"容量を超えない", true, false, 20, 0, 0},
		{"容量を超えていても減らさない", true, false, 20, -3, 0},
		{"畳んだキューブは戻らない", false, false, 20, 5, 0},
		{"退避中のステージのキューブは戻らない", true, true, 20, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 3, Y: 3})
			require.NoError(t, err)
			if tt.deployed {
				world.Components.Deployed.Add(cube, &gc.Deployed{})
			}
			if tt.suspended {
				world.Components.Suspended.Add(cube, &gc.Suspended{})
			}
			tank := world.Components.Charge.Get(cube)
			tank.Current = tank.Max - tt.missing
			before := tank.Current
			query.GetGameTime(world).TotalTurns = tt.turn

			require.NoError(t, (&DeployedFuelSystem{}).Update(world))
			assert.Equal(t, before+tt.want, world.Components.Charge.Get(cube).Current)
		})
	}
}

func TestRunTurnEndSystems_登録したDeployedFuelSystemが展開中のキューブの燃料を戻す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 3, Y: 3})
	require.NoError(t, err)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	tank := world.Components.Charge.Get(cube)
	tank.Current = tank.Max - 5
	before := tank.Current
	query.GetGameTime(world).TotalTurns = deployedFuelTurns

	updaters, _ := InitializeSystems(world)
	name := (&DeployedFuelSystem{}).String()
	sys, ok := updaters[name].(*DeployedFuelSystem)
	require.True(t, ok, "InitializeSystems に登録する")
	world.Updaters = map[string]w.Updater{name: sys}

	require.NoError(t, runTurnEndSystems(world))
	assert.Equal(t, before+1, world.Components.Charge.Get(cube).Current)
}
