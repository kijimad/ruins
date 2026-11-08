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
		subState     gc.AIStateSubState
		fuel         consts.Heat
		wantBehavior gc.BehaviorName
		wantDest     consts.Coord[consts.Tile]
		wantFuel     consts.Heat
		wantSub      gc.AIStateSubState
		wantMovement gc.SoloMovement
	}{
		{"休んでいて距離が開く前は動かない", consts.Coord[consts.Tile]{X: 8, Y: 5}, gc.AIStateWaiting, 100, gc.BehaviorWait, consts.Coord[consts.Tile]{}, 100, gc.AIStateWaiting, gc.SoloFollow},
		{"休んでいて距離が開いたら追いかけ始める", consts.Coord[consts.Tile]{X: 9, Y: 5}, gc.AIStateWaiting, 100, gc.BehaviorMove, consts.Coord[consts.Tile]{X: 8, Y: 5}, 100 - step, gc.AIStateDriving, gc.SoloFollow},
		{"追いかけている間は近くても隣まで寄る", consts.Coord[consts.Tile]{X: 8, Y: 5}, gc.AIStateDriving, 100, gc.BehaviorMove, consts.Coord[consts.Tile]{X: 7, Y: 5}, 100 - step, gc.AIStateDriving, gc.SoloFollow},
		{"隣に追いついたら休む", consts.Coord[consts.Tile]{X: 6, Y: 5}, gc.AIStateDriving, 100, gc.BehaviorWait, consts.Coord[consts.Tile]{}, 100, gc.AIStateWaiting, gc.SoloFollow},
		{"燃料が足りなければ待機の方針へ切り替える", consts.Coord[consts.Tile]{X: 8, Y: 5}, gc.AIStateDriving, step - 1, gc.BehaviorWait, consts.Coord[consts.Tile]{}, step - 1, gc.AIStateDriving, gc.SoloStationary},
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
			solo.SubState = tt.subState
			world.Components.FuelTank.Get(cube).Heat = tt.fuel

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
			assert.Equal(t, tt.wantSub, solo.SubState)
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
	solo.SubState = gc.AIStateDriving
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
	staying, err := lifecycle.SpawnCube(world, far.Add(consts.Coord[consts.Tile]{Y: 1}))
	require.NoError(t, err)
	world.Components.SoloAI.Get(staying).Movement = gc.SoloStationary

	kept, err := cullDistantSolo(world, []ecs.Entity{following, staying})
	require.NoError(t, err)
	assert.Equal(t, []ecs.Entity{following}, kept)
}

func TestPlanFollowAction_展開中は方針を保ったまま休む(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 15, Y: 5})
	require.NoError(t, err)
	require.True(t, lifecycle.DeployCube(world, cube))
	solo := world.Components.SoloAI.Get(cube)
	solo.SubState = gc.AIStateDriving

	rp := newSoloPlanner(newTestRNG())
	act := rp.planFollowAction(world, cube, solo, world.Components.GridElement.Get(cube))

	require.NotNil(t, act)
	assert.Equal(t, gc.BehaviorWait, act.BehaviorName)
	assert.Equal(t, gc.SoloFollow, solo.Movement)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState)
	assert.Equal(t, consts.FuelGaugeFullHeat, query.CubeFuelTotal(world, cube))
}
