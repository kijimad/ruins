package states_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
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

// TestGolden_Remains は床に寝かせて描く跡の見え方を固定する。物の跡と敵の跡を並べ、ワープの穴に重なる
// 跡は穴の下に描かれる
func TestGolden_Remains(t *testing.T) {
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

	for i, id := range []string{"wood_splinters", "metal_fragments", "stone_rubble"} {
		_, err := lifecycle.SpawnRemains(world, id, consts.Coord[consts.Tile]{X: pc.X + 2, Y: pc.Y - 1 + consts.Tile(i)})
		require.NoError(t, err)
	}
	for i, id := range []string{"blood_stain", "oil_stain", "slime_puddle", "scorch_mark", "ectoplasm"} {
		_, err := lifecycle.SpawnRemains(world, id, consts.Coord[consts.Tile]{X: pc.X, Y: pc.Y - 3 + consts.Tile(i)})
		require.NoError(t, err)
	}
	warp := consts.Coord[consts.Tile]{X: pc.X + 1, Y: pc.Y + 1}
	_, err = lifecycle.SpawnProp(world, "warp_next", warp.X, warp.Y)
	require.NoError(t, err)
	_, err = lifecycle.SpawnRemains(world, "wood_splinters", warp)
	require.NoError(t, err)

	require.NoError(t, sm.Update(world))
	vrt.DrawFrame(consts.GameWidth, consts.GameHeight, func(screen *ebiten.Image) {
		require.NoError(t, sm.Draw(world, screen))
	}).AssertGolden(t, "TestGolden_Remains")
}
