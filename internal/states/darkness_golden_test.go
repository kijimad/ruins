package states_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGolden_PitchDark は光源の無い真っ暗な場所の見え方を固定する。足元の1マスと自分の姿は無彩色で描き、
// 周りは記憶の色味で描く
func TestGolden_PitchDark(t *testing.T) {
	t.Parallel()
	world := vrt.InitReplayWorld(t)
	sm, err := es.Init(&gs.DungeonState{Depth: 1, DefinitionName: dungeon.DebugID, BuilderType: mapplanner.LayoutPlannerType("small_room")}, world)
	require.NoError(t, err)
	for range 5 {
		require.NoError(t, sm.Update(world))
	}
	var lit []ecs.Entity
	q := ecs.NewFilter1[gc.LightSource](world.ECS).Query()
	for q.Next() {
		lit = append(lit, q.Entity())
	}
	for _, e := range lit {
		world.Components.LightSource.Remove(e)
	}
	query.GetVisionState(world).RequestUpdate()
	query.GetPlayerCamera(world).Dist = 6
	for range 2 {
		require.NoError(t, sm.Update(world))
	}
	vrt.DrawFrame(consts.GameWidth, consts.GameHeight, func(screen *ebiten.Image) {
		require.NoError(t, sm.Draw(world, screen))
	}).AssertGolden(t, "TestGolden_PitchDark")
}

// TestVisionSystem_光源が無ければ足元の1マスだけが見える は真っ暗でも足元が見えることを固定する
func TestVisionSystem_光源が無ければ足元の1マスだけが見える(t *testing.T) {
	t.Parallel()
	world := vrt.InitReplayWorld(t)
	sm, err := es.Init(&gs.DungeonState{Depth: 1, DefinitionName: dungeon.DebugID, BuilderType: mapplanner.LayoutPlannerType("small_room")}, world)
	require.NoError(t, err)
	require.NoError(t, sm.Update(world))
	var lit []ecs.Entity
	q := ecs.NewFilter1[gc.LightSource](world.ECS).Query()
	for q.Next() {
		lit = append(lit, q.Entity())
	}
	for _, e := range lit {
		world.Components.LightSource.Remove(e)
	}
	query.GetVisionState(world).RequestUpdate()

	require.NoError(t, systems.NewVisionSystem().Update(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	assert.Equal(t, map[gc.GridElement]bool{*world.Components.GridElement.Get(player): true}, query.GetVisionState(world).VisibleTiles)
}
