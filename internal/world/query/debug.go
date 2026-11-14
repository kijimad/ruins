package query

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// IsDebugInvincible は entity がデバッグの無敵で守られるプレイヤーかを返す
func IsDebugInvincible(world w.World, entity ecs.Entity) bool {
	cfg := world.Resources.Config
	return cfg != nil && cfg.DebugInvincible && world.Components.Player.Has(entity)
}
