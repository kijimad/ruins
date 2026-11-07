package aiinput

import (
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

func TestPlanFollowAction(t *testing.T) {
	t.Parallel()
	// 空のキューブの1歩は基準の燃料ポイントだけかかる
	step := query.FollowFuelCost(0)
	tests := []struct {
		name         string
		cubeAt       consts.Coord[consts.Tile]
		fuel         consts.Heat
		ap           int
		wantBehavior gc.BehaviorName
		wantDest     consts.Coord[consts.Tile]
		wantFuel     consts.Heat
		wantMovement gc.SoloMovement
	}{
		{"離れていれば1歩近づき燃料を払う", consts.Coord[consts.Tile]{X: 8, Y: 5}, 100, 100, gc.BehaviorMove, consts.Coord[consts.Tile]{X: 7, Y: 5}, 100 - step, gc.SoloFollow},
		{"隣接していれば待ち燃料を払わない", consts.Coord[consts.Tile]{X: 6, Y: 5}, 100, 100, gc.BehaviorWait, consts.Coord[consts.Tile]{}, 100, gc.SoloFollow},
		{"燃料が足りなければ待機へ切り替える", consts.Coord[consts.Tile]{X: 8, Y: 5}, step - 1, 100, gc.BehaviorWait, consts.Coord[consts.Tile]{}, step - 1, gc.SoloStationary},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			cube, err := lifecycle.SpawnCube(world, tt.cubeAt)
			require.NoError(t, err)
			solo := world.Components.SoloAI.Get(cube)
			solo.Movement = gc.SoloFollow
			world.Components.FuelTank.Get(cube).Heat = tt.fuel
			world.Components.TurnBased.Get(cube).AP.Current = tt.ap

			rp := newSoloPlanner(newTestRNG())
			act := rp.planFollowAction(world, cube, solo, world.Components.GridElement.Get(cube))

			require.NotNil(t, act)
			assert.Equal(t, tt.wantBehavior, act.BehaviorName)
			if tt.wantBehavior == gc.BehaviorMove {
				p, ok := act.Params.(*gc.MoveParams)
				require.True(t, ok)
				assert.Equal(t, tt.wantDest, p.Destination.Coord)
			}
			assert.Equal(t, tt.wantFuel, query.CubeFuelTotal(world, cube))
			assert.Equal(t, tt.wantMovement, solo.Movement)
		})
	}
}

func TestPlanFollowAction_AP不足なら燃料を払わず行動を終える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 8, Y: 5})
	require.NoError(t, err)
	solo := world.Components.SoloAI.Get(cube)
	solo.Movement = gc.SoloFollow
	world.Components.TurnBased.Get(cube).AP.Current = consts.StandardActionCost - 1

	rp := newSoloPlanner(newTestRNG())
	act := rp.planFollowAction(world, cube, solo, world.Components.GridElement.Get(cube))

	assert.Nil(t, act)
	assert.Equal(t, consts.FuelGaugeFullHeat, query.CubeFuelTotal(world, cube))
}

func TestCullDistantSolo_追従中のキューブは圏外でも残す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
	require.NoError(t, err)
	far := consts.Coord[consts.Tile]{X: consts.Tile(activationRadius + 5), Y: 0}
	following, err := lifecycle.SpawnCube(world, far)
	require.NoError(t, err)
	world.Components.SoloAI.Get(following).Movement = gc.SoloFollow
	staying, err := lifecycle.SpawnCube(world, far.Add(consts.Coord[consts.Tile]{Y: 1}))
	require.NoError(t, err)

	kept, err := cullDistantSolo(world, []ecs.Entity{following, staying})
	require.NoError(t, err)
	assert.Equal(t, []ecs.Entity{following}, kept)
}
