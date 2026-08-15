package states_test

// 自然フレーム golden。ゲームとして自然な状態を、内部に手を入れず state の実 Draw で丸ごと撮る。
// 3D世界レイヤの上に HUD が実際どおり合成される。ピクセルは同一環境で決定的なので、
// assertPNGGolden のトレランスは環境差だけを吸う。prep も分離もしない。

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

// TestGolden_Overworld は新規ゲーム開始直後のオーバーワールド実画面を丸ごと固定する。
// プレイヤーが実際に見るフレーム、すなわち3D世界の上にHUDが合成された状態を、
// DungeonState の実 Draw をフルスタックで通して撮る。
func TestGolden_Overworld(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(_ w.World) []es.State[w.World] {
		s, err := gs.NewOverworldState(
			mapplanner.PlannerTypeOverworldField,
			dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1),
			&overworld.NewGameParams{},
		)()
		require.NoError(t, err)
		return []es.State[w.World]{s}
	})
}

// TestGolden_Dungeon は遺跡へ入った直後のダンジョン実画面を丸ごと固定する。
// プレイヤーは上り階段の上に湧く実スポーンのまま、3D世界の上にHUDを合成して撮る。
func TestGolden_Dungeon(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:          1,
		DefinitionName: dungeon.DungeonDebug.Name(),
		BuilderType:    mapplanner.PlannerTypeSmallRoom,
	}))
}
