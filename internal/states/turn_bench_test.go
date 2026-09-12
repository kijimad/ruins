package states_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/require"
)

// BenchmarkTurn は実ダンジョンで1ゲームターン全体の実時間を測る。vrt.InitReplayWorld と es.Init で
// 本番のシステム登録込みのワールドを組み、TurnSystem を Phase=AI にして Update を2回回すと、
// AI フェーズと End フェーズが走って1ターンが確定する。
func BenchmarkTurn(b *testing.B) {
	builders := []struct {
		name string
		bt   mapplanner.PlannerType
	}{
		{"SmallRoom", mapplanner.PlannerTypeSmallRoom},
		{"BigRoom", mapplanner.PlannerTypeBigRoom},
		{"Cave", mapplanner.PlannerTypeCave},
	}

	for _, bld := range builders {
		world := vrt.InitReplayWorld(b)
		sm, err := es.Init(&gs.DungeonState{Depth: 1, DefinitionName: dungeon.DungeonDebug.Name(), BuilderType: bld.bt}, world)
		require.NoError(b, err)
		// 数フレーム回して世界を落ち着かせてから測る
		for range 5 {
			require.NoError(b, sm.Update(world))
		}

		turnState := query.GetTurnState(world)
		sys := &systems.TurnSystem{}

		b.Run(bld.name, func(b *testing.B) {
			for range b.N {
				turnState.Phase = gc.TurnPhaseAI
				require.NoError(b, sys.Update(world)) // AI フェーズ -> End へ
				require.NoError(b, sys.Update(world)) // End フェーズ -> Player へ。ターンが1つ確定する
			}
		})
	}
}
