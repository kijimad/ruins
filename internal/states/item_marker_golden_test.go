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

// TestGolden_ItemMarker はプレイヤー隣接の「中身のある収納」と「重なった複数アイテム」の升に
// ☰ マーカーが出ることを固定する。単品で見えているアイテムには出さない。
func TestGolden_ItemMarker(t *testing.T) {
	t.Parallel()
	world := vrt.InitReplayWorld(t)
	sm, err := es.Init(&gs.DungeonState{Depth: 1, DefinitionName: dungeon.DungeonDebug.Name(), BuilderType: mapplanner.PlannerTypeSmallRoom}, world)
	require.NoError(t, err)
	// プレイヤー生成と視界確定のため数フレーム回す
	for range 5 {
		require.NoError(t, sm.Update(world))
	}

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	pc := world.Components.GridElement.Get(player).Coord

	// プレイヤー右手の開けた床へ、距離2以内で視線方向に3ケースをずらして並べ、対比を一目で読めるようにする。
	// 重なり2個(マーカー有り)、単品(マーカー無し)、中身入り収納(マーカー有り)。
	// SpawnFieldItem の末尾引数は同じ升に生成するエンティティ個数で、2 なら重なり2個になる
	_, err = lifecycle.SpawnFieldItem(world, "healing_potion", pc.X+2, pc.Y-1, 2)
	require.NoError(t, err)
	_, err = lifecycle.SpawnFieldItem(world, "healing_potion", pc.X+2, pc.Y, 1)
	require.NoError(t, err)
	closet, err := lifecycle.SpawnProp(world, "closet", pc.X+2, pc.Y+1)
	require.NoError(t, err)
	_, err = lifecycle.SpawnStorageItem(world, "healing_potion", 3, closet)
	require.NoError(t, err)

	// 視界の再計算のため1フレーム進めてから撮る
	require.NoError(t, sm.Update(world))
	screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
	require.NoError(t, sm.Draw(world, screen))
	vrt.AssertFrameGolden(t, "TestGolden_ItemMarker", screen)
}
