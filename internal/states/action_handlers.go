package states

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// InteractionAction はインタラクション可能なアクション情報
type InteractionAction struct {
	Label       string             // 表示ラベル（例："開く(上)"）
	Target      ecs.Entity         // ターゲットエンティティ
	Interaction gc.InteractionData // 実行するインタラクション
}

// GetInteractionActions はプレイヤー周辺の実行可能なアクションを取得する
func GetInteractionActions(world w.World) []InteractionAction {
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return nil
	}

	if !playerEntity.HasComponent(world.Components.GridElement) {
		return nil
	}

	gridElement := world.Components.GridElement.MustGet(playerEntity)

	var interactionActions []InteractionAction

	// インタラクティブな相互作用を全て取得してアクションを生成
	interactableEntities := activity.GetAllInteractiveInteractablesInRange(world, gridElement)
	for _, interactableEntity := range interactableEntities {
		if !interactableEntity.HasComponent(world.Components.GridElement) {
			continue
		}
		if !interactableEntity.HasComponent(world.Components.Interactable) {
			continue
		}

		interactableGrid := world.Components.GridElement.MustGet(interactableEntity)
		interactable := world.Components.Interactable.MustGet(interactableEntity)
		dirLabel := activity.GetDirectionLabel(gridElement, interactableGrid)
		actionsForEntity := getInteractionActions(world, interactable, interactableEntity, dirLabel)
		interactionActions = append(interactionActions, actionsForEntity...)
	}

	return interactionActions
}

// GetSameTileManualActions はプレイヤー直上のManual発動アクションを全て取得する
func GetSameTileManualActions(world w.World) []InteractionAction {
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return nil
	}
	if !playerEntity.HasComponent(world.Components.GridElement) {
		return nil
	}
	playerGrid := world.Components.GridElement.MustGet(playerEntity)

	var actions []InteractionAction
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.Interactable,
		world.Components.Dead.Not(),
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		ge := world.Components.GridElement.MustGet(entity)
		if ge.X != playerGrid.X || ge.Y != playerGrid.Y {
			return
		}
		interactable := world.Components.Interactable.MustGet(entity)
		// Manual+SameTileのインタラクションのみフィルタする
		var filtered []gc.InteractionData
		for _, interaction := range interactable.Interactions {
			config := interaction.Config()
			if config.ActivationRange == gc.ActivationRangeSameTile && config.ActivationWay == gc.ActivationWayManual {
				filtered = append(filtered, interaction)
			}
		}
		if len(filtered) > 0 {
			filteredInteractable := &gc.Interactable{Interactions: filtered}
			entityActions := getInteractionActions(world, filteredInteractable, entity, "直上")
			actions = append(actions, entityActions...)
		}
	}))

	// アイテム拾得アクションが2個以上ある場合、「すべて拾う」を先頭に追加する
	itemCount := 0
	for _, action := range actions {
		if _, ok := action.Interaction.(gc.ItemInteraction); ok {
			itemCount++
		}
	}
	if itemCount >= 2 {
		pickupAll := InteractionAction{
			Label:       "すべて拾う",
			Interaction: gc.ItemAllInteraction{},
		}
		actions = append([]InteractionAction{pickupAll}, actions...)
	}

	return actions
}

// getInteractionActions はInteractableに対応するアクションを取得する
func getInteractionActions(world w.World, interactable *gc.Interactable, interactableEntity ecs.Entity, dirLabel string) []InteractionAction {
	var result []InteractionAction

	for _, interaction := range interactable.Interactions {
		switch data := interaction.(type) {
		case gc.DoorInteraction:
			if interactableEntity.HasComponent(world.Components.Door) {
				door := world.Components.Door.MustGet(interactableEntity)
				var label string
				if door.IsOpen {
					label = "閉じる(" + dirLabel + ")"
				} else {
					label = "開く(" + dirLabel + ")"
				}
				result = append(result, InteractionAction{
					Label:       label,
					Target:      interactableEntity,
					Interaction: data,
				})
			}
		case gc.TalkInteraction:
			if interactableEntity.HasComponent(world.Components.Name) {
				name := world.Components.Name.MustGet(interactableEntity)
				result = append(result, InteractionAction{
					Label:       "話しかける(" + name.Name + ")",
					Target:      interactableEntity,
					Interaction: data,
				})
			}
		case gc.ItemInteraction:
			formattedName := query.FormatItemName(world, interactableEntity)
			result = append(result, InteractionAction{
				Label:       "拾う(" + formattedName + ")",
				Target:      interactableEntity,
				Interaction: data,
			})
		case gc.PortalInteraction:
			var label string
			switch data.PortalType {
			case gc.PortalTypeNext:
				label = "転移する(次階)"
			case gc.PortalTypeTown:
				label = "転移する(帰還)"
			}
			result = append(result, InteractionAction{
				Label:       label,
				Target:      interactableEntity,
				Interaction: data,
			})
		case gc.DungeonGateInteraction:
			result = append(result, InteractionAction{
				Label:       "ダンジョンを選ぶ",
				Target:      interactableEntity,
				Interaction: data,
			})
		case gc.StorageInteraction:
			if interactableEntity.HasComponent(world.Components.Name) {
				name := world.Components.Name.MustGet(interactableEntity)
				result = append(result, InteractionAction{
					Label:       "調べる(" + name.Name + ")",
					Target:      interactableEntity,
					Interaction: data,
				})
			}
		case gc.MeleeInteraction:
			if interactableEntity.HasComponent(world.Components.Name) {
				name := world.Components.Name.MustGet(interactableEntity)
				result = append(result, InteractionAction{
					Label:       "攻撃する(" + name.Name + ")",
					Target:      interactableEntity,
					Interaction: data,
				})
			}
		}
	}

	return result
}
