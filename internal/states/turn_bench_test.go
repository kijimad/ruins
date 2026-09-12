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

// BenchmarkTurn は本番と同じワールドで1ゲームターン全体の実時間を計測する。#809 のトレースで言う
// turn:AI と turn:End を合わせた「1ターンの中身」で、benchstat で回帰を追える per-turn の基準になる。
//
// ワールドは BenchmarkDungeonFrame と同じく vrt.InitReplayWorld と es.Init で実ダンジョンを組む。
// これで maingame.InitWorld 経由の本番 InitializeSystems が走り、ターン末システムが world.Updaters
// へ本番のまま登録される。手張りせずに実システムを測れる。
//
// TurnSystem を Phase=AI にして Update を2回回す。1回目で AI フェーズ、2回目で End フェーズが走り、
// 1ターンが確定する。プレイヤー入力を差し込まずに実プレイと同じ AI 処理とターン末処理を測れる。
// 視界の再計算は AI フェーズが要求するだけで別システムが実行するので、ここには含まれない。
// AI のスケール網羅は aiinput の BenchmarkProcessAll が担うので、こちらは実ダンジョンの忠実な
// 1ターンコストに徹する。確保 B/op は AI が乱数駆動なので決定論的でない。CI 判定は sec/op の geomean。
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
