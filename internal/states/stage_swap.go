package states

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/world/query"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// boundEntities は指定ステージに属するエンティティを集める。
// クエリ反復中は構造変更できないので、呼び出し側は集めてから付け外しする。
// 途中 return するとワールドをロックしたまま残すため、クエリは最後まで反復する
func boundEntities(world w.World, key gc.StageKey) []ecs.Entity {
	var members []ecs.Entity
	q := ecs.NewFilter1[gc.StageBound](world.ECS).Query()
	for q.Next() {
		if world.Components.StageBound.Get(q.Entity()).Key == key {
			members = append(members, q.Entity())
		}
	}
	return members
}

// suspendStage は指定ステージのエンティティを退避する。
// Suspended マーカーを付け、ステージ跨ぎのシステムの対象から外す。
// 既に Suspended のエンティティへの二重付与は避ける
func suspendStage(world w.World, key gc.StageKey) {
	for _, e := range boundEntities(world, key) {
		if !world.Components.Suspended.Has(e) {
			world.Components.Suspended.Add(e, &gc.Suspended{})
		}
	}
}

// resumeStage は指定ステージのエンティティを再稼働する。Suspended を外す。
// 退避していないエンティティはそのまま
func resumeStage(world w.World, key gc.StageKey) {
	for _, e := range boundEntities(world, key) {
		if world.Components.Suspended.Has(e) {
			world.Components.Suspended.Remove(e)
		}
	}
}

// stageExists は指定ステージのエンティティが1つでも存在するかを返す。
// 訪問済みかどうかの判定に使う。生成済みなら true
func stageExists(world w.World, key gc.StageKey) bool {
	return len(boundEntities(world, key)) > 0
}

// purgeStage は指定ステージのエンティティを world から完全に除去する。
// 完全離脱でステージを破棄するときに使う
func purgeStage(world w.World, key gc.StageKey) {
	for _, e := range boundEntities(world, key) {
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

// bindToStage は StageBound を持たないフィールドエンティティに key を付ける。
// ステージ生成の直後に呼び、生成物をそのステージの一員として識別できるようにする。
// GridElement を持つが Player・SquadMember・既存 StageBound でないものが対象。
// Player・SquadMember はステージをまたいで生きるので付けない。
// 退避中エンティティは既に StageBound を持つので自然に除外される
func bindToStage(world w.World, key gc.StageKey) {
	var targets []ecs.Entity
	q := ecs.NewFilter1[gc.GridElement](world.ECS).
		Without(ecs.C[gc.StageBound](), ecs.C[gc.Player](), ecs.C[gc.SquadMember]()).Query()
	for q.Next() {
		targets = append(targets, q.Entity())
	}
	for _, e := range targets {
		world.Components.StageBound.Add(e, &gc.StageBound{Key: key})
	}
}

// swapTo は現ステージを退避し target へ切り替える正典操作。往復のすべてがこれに還元される。
// target が訪問済みなら再稼働し、未訪問なら generate で決定的生成する。
// target は現ステージと異なる前提。自己スワップは呼ばない。
//
// generate は生成物へ **必ず** StageBound{target} を付ける責務を負う。付け忘れると未分類の
// フィールドエンティティが生じ、以降の退避で漏れる。生成の実体 spawnFloor は末尾の
// bindToStage で付与している。別の generate を実装するときも同様に付けること。
//
// プレイヤー配置と前線など時間派生の再導出は、遷移ごとに違うので呼び出し側が続けて行う
func swapTo(world w.World, target gc.StageKey, generate func(world w.World, key gc.StageKey) error) error {
	d := query.GetDungeon(world)
	// プレイ中に湧いた未タグのフィールドエンティティを現ステージへ回収する。
	// ドロップ・置いたアイテム・エフェクトが退避されず次ステージへ漏れるのを防ぐ
	bindToStage(world, d.CurrentStage)
	if stageExists(world, target) {
		// 訪問済み。現ステージを退避してから target を再稼働する
		if d.CurrentStage != target {
			suspendStage(world, d.CurrentStage)
		}
		resumeStage(world, target)
	} else {
		// 未訪問。先に生成し、失敗したら現ステージを退避せず CurrentStage も動かさない。
		// 生成中は現ステージがまだ稼働しているが、生成物は新しい StageBound を持つので混ざらない
		if err := generate(world, target); err != nil {
			return err
		}
		if d.CurrentStage != target {
			suspendStage(world, d.CurrentStage)
		}
	}
	d.CurrentStage = target
	resetExploredTiles(world)
	// 座標索引は現ステージのみで作り直す。swap で無効化し次アクセスで再構築させる
	query.InvalidateSpatialIndex(world)
	return nil
}
