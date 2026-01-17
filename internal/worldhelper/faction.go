package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// FactionType は派閥を表すenum型
type FactionType int

const (
	// FactionAlly は味方
	FactionAlly FactionType = iota
	// FactionEnemy は敵
	FactionEnemy
	// FactionNeutral は中立
	FactionNeutral
)

// ChangeToAlly は味方に変更
func ChangeToAlly(world w.World, entity ecs.Entity) {
	setFactionType(world, entity, FactionAlly)
}

// ChangeToEnemy は敵に変更
func ChangeToEnemy(world w.World, entity ecs.Entity) {
	setFactionType(world, entity, FactionEnemy)
}

// ChangeToNeutral は中立に変更
func ChangeToNeutral(world w.World, entity ecs.Entity) {
	setFactionType(world, entity, FactionNeutral)
}

// setFactionType は派閥を設定（排他制御を保証）
// 既存の派閥コンポーネントをすべて削除してから、新しい派閥を設定する
// 内部用関数なので直接呼び出さず、MakeAlly, MakeEnemy, MakeNeutral を使用すること
func setFactionType(world w.World, entity ecs.Entity, factionType FactionType) {
	// すべての派閥コンポーネントを削除（排他制御）
	entity.RemoveComponent(world.Components.FactionAlly)
	entity.RemoveComponent(world.Components.FactionEnemy)
	entity.RemoveComponent(world.Components.FactionNeutral)

	// 指定された派閥コンポーネントを追加
	switch factionType {
	case FactionAlly:
		entity.AddComponent(world.Components.FactionAlly, &gc.FactionAllyData{})
	case FactionEnemy:
		entity.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemyData{})
	case FactionNeutral:
		entity.AddComponent(world.Components.FactionNeutral, &gc.FactionNeutralData{})
	}
}
