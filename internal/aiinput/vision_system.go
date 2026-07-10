package aiinput

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// VisionSystem はAIの視界判定システム
type VisionSystem interface {
	CanSeeTarget(world w.World, aiEntity, targetEntity ecs.Entity, viewDistance consts.Tile) bool
}

// DefaultVisionSystem は標準的な視界判定実装
type DefaultVisionSystem struct{}

// NewVisionSystem は新しいVisionSystemを作成する
func NewVisionSystem() VisionSystem {
	return &DefaultVisionSystem{}
}

// CanSeeTarget はターゲットが視界内にいるかチェック
func (vs *DefaultVisionSystem) CanSeeTarget(world w.World, aiEntity, targetEntity ecs.Entity, viewDistance consts.Tile) bool {
	aiGrid := world.Components.GridElement.Get(aiEntity)
	targetGrid := world.Components.GridElement.Get(targetEntity)

	dx := int(aiGrid.X) - int(targetGrid.X)
	dy := int(aiGrid.Y) - int(targetGrid.Y)
	distSq := dx*dx + dy*dy

	viewDist := float64(viewDistance)

	if world.Components.CharModifiers.Has(targetEntity) {
		mods := world.Components.CharModifiers.Get(targetEntity)
		viewDist = viewDist * float64(mods.EnemyVision) / 100
	}

	return float64(distSq) <= viewDist*viewDist
}
