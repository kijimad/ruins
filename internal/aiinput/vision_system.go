package aiinput

import (
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// VisionSystem はAIの視界判定システム
type VisionSystem interface {
	CanSeeTarget(world w.World, aiEntity, targetEntity ecs.Entity, vision *gc.AIVision) bool
}

// DefaultVisionSystem は標準的な視界判定実装
type DefaultVisionSystem struct{}

// NewVisionSystem は新しいVisionSystemを作成する
func NewVisionSystem() VisionSystem {
	return &DefaultVisionSystem{}
}

// CanSeeTarget はターゲットが視界内にいるかチェック
func (vs *DefaultVisionSystem) CanSeeTarget(world w.World, aiEntity, targetEntity ecs.Entity, vision *gc.AIVision) bool {
	aiGrid := world.Components.GridElement.Get(aiEntity).(*gc.GridElement)
	targetGrid := world.Components.GridElement.Get(targetEntity).(*gc.GridElement)

	// 距離計算（タイル単位）
	dx := float64(int(targetGrid.X) - int(aiGrid.X))
	dy := float64(int(targetGrid.Y) - int(aiGrid.Y))
	distance := math.Sqrt(dx*dx + dy*dy)

	// 視界距離内かチェック（タイル単位で計算）
	viewDistanceInTiles := float64(vision.ViewDistance) / 32.0 // 仮にタイル1つ=32ピクセル

	return distance <= viewDistanceInTiles
}

// CalculateDistance は2つのエンティティ間の距離を計算（タイル単位）
func (vs *DefaultVisionSystem) CalculateDistance(world w.World, entity1, entity2 ecs.Entity) float64 {
	grid1 := world.Components.GridElement.Get(entity1).(*gc.GridElement)
	grid2 := world.Components.GridElement.Get(entity2).(*gc.GridElement)

	dx := float64(int(grid2.X) - int(grid1.X))
	dy := float64(int(grid2.Y) - int(grid1.Y))

	return math.Sqrt(dx*dx + dy*dy)
}

// IsInRange は指定した範囲内にターゲットがいるかチェック
func (vs *DefaultVisionSystem) IsInRange(world w.World, aiEntity, targetEntity ecs.Entity, rangeInTiles float64) bool {
	distance := vs.CalculateDistance(world, aiEntity, targetEntity)
	return distance <= rangeInTiles
}
