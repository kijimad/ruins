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
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
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
