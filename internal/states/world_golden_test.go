package states_test

// 自然フレーム golden。ゲームとして自然な状態を、内部に手を入れず state の実 Draw で丸ごと撮る。
// 3D世界の上に HUD が実プレイどおり合成される。prep も分離もしない。

import (
	"testing"

	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/stretchr/testify/require"
)

// TestGolden_Overworld は新規ゲーム開始直後のオーバーワールド実画面を固定する。
func TestGolden_Overworld(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, t.Name(), func(_ w.World) []es.State[w.World] {
		s, err := gs.NewOverworldState(
			mapplanner.PlannerTypeOverworldField,
			dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1),
			&overworld.NewGameParams{},
		)()
		require.NoError(t, err)
		return []es.State[w.World]{s}
	})
}

// TestGolden_Dungeon は遺跡へ入った直後のダンジョン実画面を固定する。
// プレイヤーは上り階段の上に湧く実スポーンのまま撮る。
func TestGolden_Dungeon(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, t.Name(), vrt.States(&gs.DungeonState{
		Depth:          1,
		DefinitionName: dungeon.DungeonDebug.Name(),
		BuilderType:    mapplanner.PlannerTypeSmallRoom,
	}))
}
