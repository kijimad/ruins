package aiinput

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/geometry"
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
	distance := geometry.Distance(float64(aiGrid.X), float64(aiGrid.Y), float64(targetGrid.X), float64(targetGrid.Y))

	// 視界距離内かチェック（タイル単位で計算）
	viewDistanceInTiles := float64(vision.ViewDistance) / 32.0 // 仮にタイル1つ=32ピクセル

	// ターゲットの隠密スキルによる被発見距離倍率を適用する
	if targetEntity.HasComponent(world.Components.CharModifiers) {
		mods := world.Components.CharModifiers.Get(targetEntity).(*gc.CharModifiers)
		viewDistanceInTiles = viewDistanceInTiles * float64(mods.EnemyVision) / 100
	}

	return distance <= viewDistanceInTiles
}
