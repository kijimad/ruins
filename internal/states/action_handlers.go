package states

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// InteractionAction はインタラクション可能なアクション情報
type InteractionAction struct {
	Label       string             // 表示ラベル
	Target      ecs.Entity         // ターゲットエンティティ
	Interaction gc.InteractionKind // 実行するインタラクション
}

// GetInteractionActions はプレイヤー周辺の実行可能なアクションを取得する
func GetInteractionActions(world w.World) []InteractionAction {
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return nil
	}

	if !world.Components.GridElement.Has(playerEntity) {
		return nil
	}

	gridElement := world.Components.GridElement.Get(playerEntity)

	var interactionActions []InteractionAction

	// インタラクティブな相互作用を全て取得してアクションを生成
	interactableEntities := activity.GetAllInteractiveInteractablesInRange(world, gridElement)
	for _, interactableEntity := range interactableEntities {
		if !world.Components.GridElement.Has(interactableEntity) {
			continue
		}
		if !world.Components.Interactable.Has(interactableEntity) {
			continue
		}

		interactableGrid := world.Components.GridElement.Get(interactableEntity)
		interactable := world.Components.Interactable.Get(interactableEntity)
		dirLabel := query.T(world, activity.GetDirectionLabel(gridElement, interactableGrid))
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
	if !world.Components.GridElement.Has(playerEntity) {
		return nil
	}
	playerGrid := world.Components.GridElement.Get(playerEntity)

	var actions []InteractionAction
	sameTileQuery := query.ActiveFilter2[gc.GridElement, gc.Interactable](world).Query()
	for sameTileQuery.Next() {
		entity := sameTileQuery.Entity()
		if world.Components.Dead.Has(entity) {
			continue
		}
		ge := world.Components.GridElement.Get(entity)
		if ge.X != playerGrid.X || ge.Y != playerGrid.Y {
			continue
		}
		interactable := world.Components.Interactable.Get(entity)
		// Manual+SameTileのインタラクションのみフィルタする
		var filtered []gc.InteractionKind
		for _, interaction := range interactable.Interactions {
			config := interaction.Config()
			if config.ActivationRange == gc.ActivationRangeSameTile && config.ActivationWay == gc.ActivationWayManual {
				filtered = append(filtered, interaction)
			}
		}
		if len(filtered) > 0 {
			filteredInteractable := &gc.Interactable{Interactions: filtered}
			entityActions := getInteractionActions(world, filteredInteractable, entity, query.T(world, "directly above"))
			actions = append(actions, entityActions...)
		}
	}

	// アイテム拾得アクションが2個以上ある場合、「すべて拾う」を先頭に追加する
	itemCount := 0
	for _, action := range actions {
		if action.Interaction == gc.InteractionItem {
			itemCount++
		}
	}
	if itemCount >= 2 {
		pickupAll := InteractionAction{
			Label:       query.T(world, "Pick up all"),
			Interaction: gc.InteractionItemAll,
		}
		actions = append([]InteractionAction{pickupAll}, actions...)
	}

	return actions
}

// getInteractionActions はInteractableに対応するアクションを取得する
func getInteractionActions(world w.World, interactable *gc.Interactable, interactableEntity ecs.Entity, dirLabel string) []InteractionAction {
	var result []InteractionAction

	for _, interaction := range interactable.Interactions {
		switch interaction {
		case gc.InteractionDoor:
			if world.Components.Door.Has(interactableEntity) {
				door := world.Components.Door.Get(interactableEntity)
				var label string
				if door.IsOpen {
					label = query.T(world, "Close (%s)", dirLabel)
				} else {
					label = query.T(world, "Open (%s)", dirLabel)
				}
				result = append(result, InteractionAction{
					Label:       label,
					Target:      interactableEntity,
					Interaction: interaction,
				})
			}
		case gc.InteractionTalk:
			if world.Components.Name.Has(interactableEntity) {
				result = append(result, InteractionAction{
					Label:       query.T(world, "Talk (%s)", query.GetEntityName(interactableEntity, world)),
					Target:      interactableEntity,
					Interaction: interaction,
				})
			}
		case gc.InteractionItem:
			formattedName := query.FormatItemName(world, interactableEntity)
			result = append(result, InteractionAction{
				Label:       query.T(world, "Pick up (%s)", formattedName),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionPortalNext:
			result = append(result, InteractionAction{
				Label:       query.T(world, "Warp (next floor)"),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionPortalPrev:
			result = append(result, InteractionAction{
				Label:       query.T(world, "Warp (previous floor)"),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionDungeonEnter:
			result = append(result, InteractionAction{
				Label:       query.T(world, "Enter ruins"),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionStorage:
			if world.Components.Name.Has(interactableEntity) {
				result = append(result, InteractionAction{
					Label:       query.T(world, "Inspect (%s)", query.GetEntityName(interactableEntity, world)),
					Target:      interactableEntity,
					Interaction: interaction,
				})
			}
		case gc.InteractionMelee:
			if world.Components.Name.Has(interactableEntity) {
				result = append(result, InteractionAction{
					Label:       query.T(world, "Attack (%s)", query.GetEntityName(interactableEntity, world)),
					Target:      interactableEntity,
					Interaction: interaction,
				})
			}
		case gc.InteractionDisassemble:
			if world.Components.Name.Has(interactableEntity) {
				result = append(result, InteractionAction{
					Label:       query.T(world, "Disassemble (%s)", query.GetEntityName(interactableEntity, world)),
					Target:      interactableEntity,
					Interaction: interaction,
				})
			}
		case gc.InteractionEnterCube:
			result = append(result, InteractionAction{
				Label:       query.T(world, "Enter (%s)", dirLabel),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionExitCube:
			result = append(result, InteractionAction{
				Label:       query.T(world, "Exit"),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionPullCube:
			result = append(result, InteractionAction{
				Label:       query.T(world, "Pull (%s)", dirLabel),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionCubePanel:
			result = append(result, InteractionAction{
				Label:       query.T(world, "Inspect (control panel)"),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionAuction:
			result = append(result, InteractionAction{
				Label:       query.T(world, "Open auction house"),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionItemAll:
			// アクションメニューに出さない種類。default を置かず exhaustive に全種別を
			// 明示させ、新しい InteractionKind の対応漏れを lint で検知する
		}
	}

	return result
}
