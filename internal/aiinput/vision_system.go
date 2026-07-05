package aiinput

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
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
	aiGrid := world.Components.GridElement.Get(aiEntity).(*gc.GridElement)
	targetGrid := world.Components.GridElement.Get(targetEntity).(*gc.GridElement)

	dx := int(aiGrid.X) - int(targetGrid.X)
	dy := int(aiGrid.Y) - int(targetGrid.Y)
	distSq := dx*dx + dy*dy

	viewDist := float64(viewDistance)

	if targetEntity.HasComponent(world.Components.CharModifiers) {
		mods := world.Components.CharModifiers.Get(targetEntity).(*gc.CharModifiers)
		viewDist = viewDist * float64(mods.EnemyVision) / 100
	}

	return float64(distSq) <= viewDist*viewDist
}
