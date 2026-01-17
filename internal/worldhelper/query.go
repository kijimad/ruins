package worldhelper

import (
	"fmt"

	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// QueryOwnedStackable は所持中のスタック可能アイテムを取得する
func QueryOwnedStackable(world w.World, f func(entity ecs.Entity)) {
	world.Manager.Join(
		world.Components.Stackable,
		world.Components.ItemLocationInBackpack,
	).Visit(ecs.Visit(f))
}

// QueryPlayer はプレイヤー
func QueryPlayer(world w.World, f func(entity ecs.Entity)) {
	world.Manager.Join(
		world.Components.Player,
		world.Components.FactionAlly,
	).Visit(ecs.Visit(f))
}

// GetPlayerEntity はプレイヤーエンティティを返す
// プレイヤーが0個または2個以上の場合はエラーを返す
func GetPlayerEntity(world w.World) (ecs.Entity, error) {
	var entities []ecs.Entity
	world.Manager.Join(world.Components.Player).Visit(ecs.Visit(func(entity ecs.Entity) {
		entities = append(entities, entity)
	}))

	if len(entities) == 0 {
		return 0, fmt.Errorf("プレイヤーエンティティが存在しません")
	}
	if len(entities) > 1 {
		return 0, fmt.Errorf("プレイヤーエンティティが複数存在します: %d個", len(entities))
	}

	return entities[0], nil
}
