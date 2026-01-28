package systems

import (
	"github.com/kijimaD/ruins/internal/actions"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// AutoInteractionSystem はプレイヤーが自動実行の相互作用に接触した際に自動実行する
type AutoInteractionSystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys AutoInteractionSystem) String() string {
	return "AutoInteractionSystem"
}

// Update はプレイヤーが自動実行の相互作用に接触した際に自動実行する
// w.Updater interfaceを実装
func (sys *AutoInteractionSystem) Update(world w.World) error {
	// プレイヤーエンティティを取得
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	// プレイヤーの位置を取得
	if !playerEntity.HasComponent(world.Components.GridElement) {
		return nil
	}
	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)

	// プレイヤーの範囲内にある相互作用を検索
	var interactablesToProcess []ecs.Entity
	world.Manager.Join(
		world.Components.Interactable,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		interactable := world.Components.Interactable.Get(entity).(*gc.Interactable)
		interactableGrid := world.Components.GridElement.Get(entity).(*gc.GridElement)

		if !worldhelper.IsInActivationRange(playerGrid, interactableGrid, interactable.Data.Config().ActivationRange) {
			return
		}

		logger.New(logger.CategoryAction).Debug("Found interactable in range",
			"entity", entity,
			"playerPos", playerGrid,
			"interactablePos", interactableGrid,
			"range", interactable.Data.Config().ActivationRange)

		interactablesToProcess = append(interactablesToProcess, entity)
	}))

	// 検索した自動実行相互作用を処理する
	for _, interactableEntity := range interactablesToProcess {
		interactable := world.Components.Interactable.Get(interactableEntity).(*gc.Interactable)
		config := interactable.Data.Config()

		// 相互作用の設定が有効かチェック
		if err := config.ActivationRange.Valid(); err != nil {
			logger.New(logger.CategoryAction).Warn("無効なActivationRangeを持つ相互作用をスキップ",
				"entity", interactableEntity,
				"range", config.ActivationRange,
				"error", err)
			continue
		}
		if err := config.ActivationWay.Valid(); err != nil {
			logger.New(logger.CategoryAction).Warn("無効なActivationWayを持つ相互作用をスキップ",
				"entity", interactableEntity,
				"way", config.ActivationWay,
				"error", err)
			continue
		}

		if config.ActivationWay != gc.ActivationWayAuto {
			continue
		}

		// 自動実行の相互作用を実行する
		activity := &actions.InteractionActivateActivity{
			InteractableEntity: interactableEntity,
		}
		params := actions.ActionParams{
			Actor: playerEntity,
		}
		manager := actions.NewActivityManager(logger.New(logger.CategoryAction))
		_, err := manager.Execute(activity, params, world)
		if err != nil {
			return err
		}
	}

	return nil
}
