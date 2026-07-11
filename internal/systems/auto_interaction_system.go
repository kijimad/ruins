package systems

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
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
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	// プレイヤーの位置を取得
	if !world.Components.GridElement.Has(playerEntity) {
		return nil
	}
	playerGrid := world.Components.GridElement.Get(playerEntity)

	// プレイヤーの範囲内にある相互作用を検索
	var interactablesToProcess []ecs.Entity
	interactableQuery := ecs.NewFilter2[gc.Interactable, gc.GridElement](world.ECS).
		Without(ecs.C[gc.Dead]()).Query()
	for interactableQuery.Next() {
		entity := interactableQuery.Entity()
		interactable := world.Components.Interactable.Get(entity)
		interactableGrid := world.Components.GridElement.Get(entity)

		// いずれかのインタラクションが範囲内にあれば候補に追加する
		for _, interaction := range interactable.Interactions {
			if query.IsInActivationRange(playerGrid, interactableGrid, interaction.Config().ActivationRange) {
				logger.New(logger.CategoryAction).Debug("Found interactable in range",
					"entity", entity,
					"playerPos", playerGrid,
					"interactablePos", interactableGrid,
					"range", interaction.Config().ActivationRange)
				interactablesToProcess = append(interactablesToProcess, entity)
				break
			}
		}
	}

	// 検索した自動実行相互作用を処理する
	for _, interactableEntity := range interactablesToProcess {
		interactable := world.Components.Interactable.Get(interactableEntity)

		for _, interaction := range interactable.Interactions {
			config := interaction.Config()

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
			_, err := activity.ExecuteInteraction(playerEntity, interactableEntity, interaction, world)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
