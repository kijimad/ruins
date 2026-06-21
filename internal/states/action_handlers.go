package states

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
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
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return nil
	}

	if !playerEntity.HasComponent(world.Components.GridElement) {
		return nil
	}

	gridElement := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)

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

		interactableGrid := world.Components.GridElement.Get(interactableEntity).(*gc.GridElement)
		interactable := world.Components.Interactable.Get(interactableEntity).(*gc.Interactable)
		dirLabel := activity.GetDirectionLabel(gridElement, interactableGrid)
		actionsForEntity := getInteractionActions(world, interactable, interactableEntity, dirLabel)
		interactionActions = append(interactionActions, actionsForEntity...)
	}

	return interactionActions
}

// getInteractionActions はInteractableに対応するアクションを取得する
func getInteractionActions(world w.World, interactable *gc.Interactable, interactableEntity ecs.Entity, dirLabel string) []InteractionAction {
	var result []InteractionAction

	for _, interaction := range interactable.Interactions {
		switch data := interaction.(type) {
		case gc.DoorInteraction:
			if interactableEntity.HasComponent(world.Components.Door) {
				door := world.Components.Door.Get(interactableEntity).(*gc.Door)
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
				name := world.Components.Name.Get(interactableEntity).(*gc.Name)
				result = append(result, InteractionAction{
					Label:       "話しかける(" + name.Name + ")",
					Target:      interactableEntity,
					Interaction: data,
				})
			}
		case gc.ItemInteraction:
			formattedName := worldhelper.FormatItemName(world, interactableEntity)
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
				name := world.Components.Name.Get(interactableEntity).(*gc.Name)
				result = append(result, InteractionAction{
					Label:       "調べる(" + name.Name + ")",
					Target:      interactableEntity,
					Interaction: data,
				})
			}
		case gc.MeleeInteraction:
			if interactableEntity.HasComponent(world.Components.Name) {
				name := world.Components.Name.Get(interactableEntity).(*gc.Name)
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
