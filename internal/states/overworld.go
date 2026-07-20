package states

import (
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	w "github.com/kijimaD/ruins/internal/world"
)

// NewOverworldState はオーバーワールド探索ステートのファクトリを返す。
//
// オーバーワールドは「帯を持つダンジョン」(DungeonOverworld, Seamless=true)で、専用の State 型は
// 持たず DungeonState として動く。帯固有のロジックは overworld.Session に閉じ込めてあり、
// DungeonState は OnStart でセッションを構成して開始を委譲し、Update でシフトを委譲するだけ。
//
// params が非 nil なら新規開始として初期帯を生成する。nil ならセーブからの復元とみなし、
// 帯パラメータは Session の Start が Dungeon.SeamlessBand から読み取って再構築する。
func NewOverworldState(planner mapplanner.PlannerType, params *overworld.NewGameParams) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &DungeonState{
			// 定義名を Seamless なオーバーワールド定義にすることで、OnStart が帯モードへ分岐する
			DefinitionName: dungeon.DungeonOverworld.Name,
			planner:        planner,
			newGame:        params,
		}, nil
	}
}
