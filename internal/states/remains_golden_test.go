package states_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
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
	for i, id := range []string{"red_blood_stain", "green_blood_stain", "blue_blood_stain", "oil_stain", "slime_puddle", "scorch_mark", "ectoplasm"} {
		_, err := lifecycle.SpawnRemains(world, id, consts.Coord[consts.Tile]{X: pc.X + consts.Tile(i%3), Y: pc.Y - 3 + consts.Tile(i/3)})
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

// TestGolden_Remains_Fadeout は倒した敵が消えていく途中の見え方を固定する。ターン末に跡が置かれ、
// 薄れていく敵の絵の下に跡が透けて見える
func TestGolden_Remains_Fadeout(t *testing.T) {
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
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: pc.X + 1, Y: pc.Y - 2}, "bat")
	require.NoError(t, err)
	world.Components.Dead.Add(enemy, &gc.Dead{})

	// 撮影用の world はアニメーションを止めて効果をすぐ消すので、この場面だけ動かしてフェードアウトを残す
	world.Resources.Config.DisableAnimation = false
	// 待機でターンを進め、ターン末に跡が置かれて敵が消えたフレームで止める。フェードアウトは始まったばかりになる
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return inputmapper.ActionWait, true }
	at := consts.Coord[consts.Tile]{X: pc.X + 1, Y: pc.Y - 2}
	for range 60 {
		if _, ok := query.RemainsAt(world, at); ok {
			break
		}
		require.NoError(t, sm.Update(world))
	}
	world.Resources.InputSource = nil
	_, ok := query.RemainsAt(world, at)
	require.True(t, ok, "前提: 倒した敵のマスに跡がある")
	require.False(t, world.ECS.Alive(enemy), "前提: 倒した敵は消えている")
	vrt.DrawFrame(consts.GameWidth, consts.GameHeight, func(screen *ebiten.Image) {
		require.NoError(t, sm.Draw(world, screen))
	}).AssertGolden(t, "TestGolden_Remains_Fadeout")
}
