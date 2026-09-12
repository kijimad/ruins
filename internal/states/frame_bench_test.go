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
	"github.com/stretchr/testify/require"
)

// benchDungeons はフレーム・ターンのベンチで使う実ダンジョンのビルダー。
var benchDungeons = []struct {
	name string
	bt   mapplanner.PlannerType
}{
	{"SmallRoom", mapplanner.PlannerTypeSmallRoom},
	{"BigRoom", mapplanner.PlannerTypeBigRoom},
	{"Cave", mapplanner.PlannerTypeCave},
}

// BenchmarkDungeonFrame は実ダンジョンの1フレームを Update / Draw に分けて実時間を計測する。
//
// これで「Update 側の最適化（AIフェーズ等）が実フレームの何%をカバーするか」が分かる。
// 60fps のフレーム予算は約16.67ms/frame。
//
// 注意：steady-state（入力待ち）では TurnSystem は Player フェーズで待機し AI フェーズは走らない。
// つまり Update 計測は「通常プレイの Update（AIスパイクを含まない）」を表し、Draw との比が
// 通常プレイで描画が占める割合を示す。AIスパイク自体は BenchmarkProcessAll で別途計測している。
// custom metric "gridEnts" は GridElement を持つエンティティ数（≒タイル数）で、Draw のスケール要因。
func BenchmarkDungeonFrame(b *testing.B) {
	for _, bld := range benchDungeons {
		world := vrt.InitReplayWorld(b)
		sm, err := es.Init(&gs.DungeonState{Depth: 1, DefinitionName: dungeon.DungeonDebug.Name(), BuilderType: bld.bt}, world)
		require.NoError(b, err)
		for range 5 {
			require.NoError(b, sm.Update(world))
		}
		screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)

		countQuery := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
		gridEnts := float64(countQuery.Count())
		countQuery.Close()

		b.Run(bld.name+"/Update", func(b *testing.B) {
			for range b.N {
				require.NoError(b, sm.Update(world))
			}
			b.ReportMetric(gridEnts, "gridEnts")
		})
		b.Run(bld.name+"/Draw", func(b *testing.B) {
			for range b.N {
				require.NoError(b, sm.Draw(world, screen))
			}
			b.ReportMetric(gridEnts, "gridEnts")
		})
	}
}

// BenchmarkTurn は実ダンジョンで1ゲームターン全体の実時間を測る。vrt.InitReplayWorld と es.Init で
// 本番のシステム登録込みのワールドを組み、TurnSystem を Phase=AI にして Update を2回回すと、
// AI フェーズと End フェーズが走って1ターンが確定する。
func BenchmarkTurn(b *testing.B) {
	for _, bld := range benchDungeons {
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
			// 反復ごとにワールドが1ターン進み状態は累積する。実プレイの流れを測るため毎回リセットしない
			for range b.N {
				turnState.Phase = gc.TurnPhaseAI
				require.NoError(b, sys.Update(world)) // AI フェーズ -> End へ
				require.NoError(b, sys.Update(world)) // End フェーズ -> Player へ。ターンが1つ確定する
			}
		})
	}
}
