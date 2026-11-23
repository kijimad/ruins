package states_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/require"
)

// TestGolden_Facing_SideOn は、カメラの前後方向に沿って並ぶ facing の置物の見え方を固定する。
// 北を見るカメラの左端に東を向いた押し入れを映し、裏の絵が表に重ならず、1台に絵が1枚だけ出る
func TestGolden_Facing_SideOn(t *testing.T) {
	t.Parallel()
	world := vrt.InitReplayWorld(t)
	sm, err := es.Init(&gs.DungeonState{Depth: 1, DefinitionName: dungeon.DebugID, BuilderType: mapplanner.LayoutPlannerType("small_room")}, world)
	require.NoError(t, err)
	for range 5 {
		require.NoError(t, sm.Update(world))
	}

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	pc := world.Components.GridElement.Get(player).Coord
	for i := range 2 {
		e, err := lifecycle.SpawnProp(world, "closet", pc.X, pc.Y-2+consts.Tile(i))
		require.NoError(t, err)
		world.Components.Pose.Add(e, &gc.Pose{Facing: consts.East, Anchor: gc.AnchorBack})
	}
	world.Components.GridElement.Get(player).X += 2
	query.GetPlayerCamera(world).Dist = 6

	require.NoError(t, sm.Update(world))
	vrt.DrawFrame(consts.GameWidth, consts.GameHeight, func(screen *ebiten.Image) {
		require.NoError(t, sm.Draw(world, screen))
	}).AssertGolden(t, "TestGolden_Facing_SideOn")
}

// TestGolden_Inset_OpenDoor は開いた扉の見え方を固定する。市街地の扉を開くと、板を開口部と直角に立てて脇の壁へ寄せる
func TestGolden_Inset_OpenDoor(t *testing.T) {
	t.Parallel()
	world := vrt.InitReplayWorld(t)
	// RunSeed 6 は開始チャンクのすぐ西が市街地になる
	s, err := gs.NewOverworldState(mapplanner.PlannerTypeOverworldField, dungeon.DungeonOverworld, &overworld.NewGameParams{RunSeed: 6})()
	require.NoError(t, err)
	sm, err := es.Init(s, world)
	require.NoError(t, err)
	query.GetGameTime(world).TotalTurns = 625
	require.NoError(t, sm.Update(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	from := world.Components.GridElement.Get(player).Coord
	var door ecs.Entity
	best := -1
	q := query.ActiveFilter2[gc.GridElement, gc.Door](world).Query()
	for q.Next() {
		c := world.Components.GridElement.Get(q.Entity()).Coord
		d := int((c.X-from.X)*(c.X-from.X) + (c.Y-from.Y)*(c.Y-from.Y))
		if best < 0 || d < best {
			door, best = q.Entity(), d
		}
	}
	require.GreaterOrEqual(t, best, 0, "前提: 帯に扉がある")
	require.NoError(t, lifecycle.OpenDoor(world, door))
	at := world.Components.GridElement.Get(door).Coord
	si := query.GetSpatialIndex(world)
	for _, d := range []consts.Coord[consts.Tile]{{Y: 2}, {Y: -2}, {X: 2}, {X: -2}} {
		if c := at.Add(d); !si.IsBlockPass(c) {
			require.NoError(t, lifecycle.MovePlayerToPosition(world, c))
			break
		}
	}
	query.GetPlayerCamera(world).Dist = 6

	require.NoError(t, sm.Update(world))
	vrt.DrawFrame(consts.GameWidth, consts.GameHeight, func(screen *ebiten.Image) {
		require.NoError(t, sm.Draw(world, screen))
	}).AssertGolden(t, "TestGolden_Inset_OpenDoor")
}
