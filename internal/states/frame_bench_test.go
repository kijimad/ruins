package states_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/require"
)

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
	builders := []struct {
		name string
		bt   mapplanner.PlannerType
	}{
		{"SmallRoom", mapplanner.PlannerTypeSmallRoom},
		{"BigRoom", mapplanner.PlannerTypeBigRoom},
		{"Cave", mapplanner.PlannerTypeCave},
	}

	for _, bld := range builders {
		world := vrt.InitVRTWorld(b)
		sm, err := es.Init(&gs.DungeonState{Depth: 1, BuilderType: bld.bt}, world)
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
