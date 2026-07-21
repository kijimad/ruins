package stage

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// BoundEntities は指定ステージに束縛されたエンティティを集める。
// クエリ反復中は構造変更できないので、呼び出し側は集めてから付け外しする。
// 途中 return するとワールドをロックしたまま残すため、クエリは最後まで反復する
func BoundEntities(world w.World, key gc.StageKey) []ecs.Entity {
	var bound []ecs.Entity
	q := ecs.NewFilter1[gc.StageBound](world.ECS).Query()
	for q.Next() {
		if world.Components.StageBound.Get(q.Entity()).Key == key {
			bound = append(bound, q.Entity())
		}
	}
	return bound
}

// suspend は指定ステージのエンティティを退避する。
// Suspended マーカーを付け、ステージ跨ぎのシステムの対象から外す。
// 既に Suspended のエンティティへの二重付与は避ける
func suspend(world w.World, key gc.StageKey) {
	for _, e := range BoundEntities(world, key) {
		if !world.Components.Suspended.Has(e) {
			world.Components.Suspended.Add(e, &gc.Suspended{})
		}
	}
}

// resume は指定ステージのエンティティを再稼働する。Suspended を外す。
// 退避していないエンティティはそのまま
func resume(world w.World, key gc.StageKey) {
	for _, e := range BoundEntities(world, key) {
		if world.Components.Suspended.Has(e) {
			world.Components.Suspended.Remove(e)
		}
	}
}

// exists は指定ステージに束縛されたエンティティが1つでも存在するかを返す。
// 訪問済みかどうかの判定に使う。生成済みなら true
func exists(world w.World, key gc.StageKey) bool {
	return len(BoundEntities(world, key)) > 0
}

// Purge は指定ステージのエンティティを world から完全に除去する。
// 特定ステージだけの破棄に使う。遺跡からオーバーワールドへ戻る際など、
// 一部ステージだけを捨て、残りを共存させたまま離脱するときに呼ぶ
func Purge(world w.World, key gc.StageKey) {
	for _, e := range BoundEntities(world, key) {
		if world.ECS.Alive(e) {
			world.ECS.RemoveEntity(e)
		}
	}
}

// ResetExploredTiles は現ステージの探索履歴を空にする。ステージへ入り直すたびにリセットする方針。
// 探索履歴は StageField が持つため、現ステージの StageField の map を貼り替える
func ResetExploredTiles(world w.World) {
	field := query.GetCurrentStageField(world)
	if field == nil {
		return
	}
	field.ExploredTiles = make(map[gc.GridElement]bool)
}

// Bind は StageBound を持たないフィールドエンティティに key を付ける。
// ステージ生成の直後に呼び、生成物をそのステージへ束縛して識別できるようにする。
// GridElement を持つが Player・SquadMember・既存 StageBound でないものが対象。
// Player・SquadMember はステージをまたいで生きるので束縛しない。
// 退避中エンティティは既に StageBound を持つので自然に除外される
func Bind(world w.World, key gc.StageKey) {
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

// SwapTo は現ステージを退避し target へ切り替える正典操作。往復のすべてがこれに還元される。
// target が訪問済みなら再稼働し、未訪問なら generate で決定的生成する。
// target は現ステージと異なる前提。自己スワップは呼ばない。
//
// generate は生成物へ **必ず** StageBound{target} を付ける責務を負う。付け忘れると未分類の
// フィールドエンティティが生じ、以降の退避で漏れる。生成側は末尾で Bind を呼び付与する。
// 別の generate を実装するときも同様に付けること。
//
// プレイヤー配置と前線など時間派生の再導出は、遷移ごとに違うので呼び出し側が続けて行う
func SwapTo(world w.World, target gc.StageKey, generate func(world w.World, key gc.StageKey) error) error {
	d := query.GetDungeon(world)
	// プレイ中に湧いた未束縛のフィールドエンティティを現ステージへ回収する。
	// ドロップ・置いたアイテム・エフェクトが退避されず次ステージへ漏れるのを防ぐ
	Bind(world, d.CurrentStage)
	if exists(world, target) {
		// 訪問済み。現ステージを退避してから target を再稼働する
		if d.CurrentStage != target {
			suspend(world, d.CurrentStage)
		}
		resume(world, target)
	} else {
		// 未訪問。先に生成し、失敗したら現ステージを退避せず CurrentStage も動かさない。
		// 生成中は現ステージがまだ稼働しているが、生成物は新しい StageBound を持つので混ざらない
		if err := generate(world, target); err != nil {
			return err
		}
		if d.CurrentStage != target {
			suspend(world, d.CurrentStage)
		}
	}
	d.CurrentStage = target
	ResetExploredTiles(world)
	// 座標索引は現ステージのみで作り直す。swap で無効化し次アクセスで再構築させる
	query.InvalidateSpatialIndex(world)
	return nil
}
