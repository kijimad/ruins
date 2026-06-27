package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetEntityName はエンティティの名前を取得する
func GetEntityName(entity ecs.Entity, world w.World) string {
	name := world.Components.Name.Get(entity)
	if name != nil {
		return name.(*gc.Name).Name
	}
	return "Unknown"
}

// AppendNameWithColor はエンティティの種類に応じて色付きで名前を追加する
func AppendNameWithColor(logger *gamelog.Logger, entity ecs.Entity, name string, world w.World) {
	if entity.HasComponent(world.Components.Player) {
		logger.PlayerName(name)
	} else if entity.HasComponent(world.Components.AIMoveFSM) {
		logger.NPCName(name)
	} else {
		logger.Append(name)
	}
}
