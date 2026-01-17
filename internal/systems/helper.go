package systems

import (
	w "github.com/kijimaD/ruins/internal/world"
)

// ShouldRunner はシステムの実行判定を行うインターフェース
// イベント駆動型システムがフラグをチェックし、UI更新判定などに使用する
type ShouldRunner interface {
	// ShouldRun はシステムを実行する必要があるかチェックし、フラグをクリアする
	// 実行が必要な場合はtrueを返す
	ShouldRun(world w.World) bool
}

// InitializeSystems は全システムを初期化して Updaters と Renderers のマップを返す
func InitializeSystems(world w.World) (map[string]w.Updater, map[string]w.Renderer) {
	updaters := make(map[string]w.Updater)
	renderers := make(map[string]w.Renderer)

	// Updaters（ロジック更新システム） ================
	cameraSystem := &CameraSystem{}
	updaters[cameraSystem.String()] = cameraSystem

	animationSystem := NewAnimationSystem()
	updaters[animationSystem.String()] = animationSystem

	turnSystem := &TurnSystem{}
	updaters[turnSystem.String()] = turnSystem

	deadCleanupSystem := &DeadCleanupSystem{}
	updaters[deadCleanupSystem.String()] = deadCleanupSystem

	autoInteractionSystem := &AutoInteractionSystem{}
	updaters[autoInteractionSystem.String()] = autoInteractionSystem

	equipmentChangedSystem := &EquipmentChangedSystem{}
	updaters[equipmentChangedSystem.String()] = equipmentChangedSystem

	inventoryChangedSystem := &InventoryChangedSystem{}
	updaters[inventoryChangedSystem.String()] = inventoryChangedSystem

	// Renderers（描画システム） ================
	renderSpriteSystem := NewRenderSpriteSystem()
	renderers[renderSpriteSystem.String()] = renderSpriteSystem

	visionSystem := &VisionSystem{}
	renderers[visionSystem.String()] = visionSystem

	// HUDRenderingSystem は Updater と Renderer の両方を実装
	hudRenderingSystem := NewHUDRenderingSystem(world)
	updaters[hudRenderingSystem.String()] = hudRenderingSystem
	renderers[hudRenderingSystem.String()] = hudRenderingSystem

	return updaters, renderers
}
