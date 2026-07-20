package states

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/world/query"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// stageMembers は指定ステージに属するエンティティを集める。
// クエリ反復中は構造変更できないので、呼び出し側は集めてから付け外しする。
// 途中 return するとワールドをロックしたまま残すため、クエリは最後まで反復する
func stageMembers(world w.World, key gc.StageKey) []ecs.Entity {
	var members []ecs.Entity
	q := ecs.NewFilter1[gc.StageMember](world.ECS).Query()
	for q.Next() {
		if world.Components.StageMember.Get(q.Entity()).Key == key {
			members = append(members, q.Entity())
		}
	}
	return members
}

// suspendStage は指定ステージのエンティティを退避する。
// Suspended マーカーを付け、ステージ跨ぎのシステムの対象から外す。
// 既に Suspended のエンティティへの二重付与は避ける
func suspendStage(world w.World, key gc.StageKey) {
	for _, e := range stageMembers(world, key) {
		if !world.Components.Suspended.Has(e) {
			world.Components.Suspended.Add(e, &gc.Suspended{})
		}
	}
}

// resumeStage は指定ステージのエンティティを再稼働する。Suspended を外す。
// 退避していないエンティティはそのまま
func resumeStage(world w.World, key gc.StageKey) {
	for _, e := range stageMembers(world, key) {
		if world.Components.Suspended.Has(e) {
			world.Components.Suspended.Remove(e)
		}
	}
}

// stageExists は指定ステージのエンティティが1つでも存在するかを返す。
// 訪問済みかどうかの判定に使う。生成済みなら true
func stageExists(world w.World, key gc.StageKey) bool {
	return len(stageMembers(world, key)) > 0
}

// purgeStage は指定ステージのエンティティを world から完全に除去する。
// 完全離脱でステージを破棄するときに使う
func purgeStage(world w.World, key gc.StageKey) {
	for _, e := range stageMembers(world, key) {
		if world.ECS.Alive(e) {
			world.ECS.RemoveEntity(e)
		}
	}
}

// resetExploredTiles は探索履歴を空にする。ステージへ入り直すたびに履歴をリセットする方針。
// ExploredTiles は単一 map だが、入場時 clear で現ステージ分の座標だけが載り衝突しない
func resetExploredTiles(world w.World) {
	d := query.GetDungeon(world)
	d.ExploredTiles = make(map[gc.GridElement]bool)
}

// swapTo は現ステージを退避し target へ切り替える正典操作。往復のすべてがこれに還元される。
// target が訪問済みなら再稼働し、未訪問なら generate で決定的生成する。
// generate は生成物へ StageMember{target} を付ける責務を負う。
// プレイヤー配置と前線など時間派生の再導出は、遷移ごとに違うので呼び出し側が続けて行う
func swapTo(world w.World, target gc.StageKey, generate func(world w.World, key gc.StageKey)) {
	d := query.GetDungeon(world)
	if d.CurrentStage != target {
		suspendStage(world, d.CurrentStage)
	}
	if stageExists(world, target) {
		resumeStage(world, target)
	} else {
		generate(world, target)
	}
	d.CurrentStage = target
	resetExploredTiles(world)
}
