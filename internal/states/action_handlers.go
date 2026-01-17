package states

import (
	"github.com/kijimaD/ruins/internal/actions"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/movement"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ExecuteMoveAction は移動アクションを実行する
func ExecuteMoveAction(world w.World, direction gc.Direction) {
	entity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return
	}

	if !entity.HasComponent(world.Components.GridElement) {
		return
	}

	gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
	currentX := int(gridElement.X)
	currentY := int(gridElement.Y)

	deltaX, deltaY := direction.GetDelta()
	newX := currentX + deltaX
	newY := currentY + deltaY

	// 移動先にOnCollision方式のInteractableがある場合は自動実行
	targetGrid := &gc.GridElement{X: gc.Tile(newX), Y: gc.Tile(newY)}
	interactable, interactableEntity := getInteractableAtSameTile(world, targetGrid)
	if interactable != nil && interactable.Data.Config().ActivationWay == gc.ActivationWayOnCollision {
		// DoorInteractionの場合は、閉じている場合のみ実行（開いている場合は通過）
		if _, isDoorInteraction := interactable.Data.(gc.DoorInteraction); isDoorInteraction {
			if interactableEntity.HasComponent(world.Components.Door) {
				door := world.Components.Door.Get(interactableEntity).(*gc.Door)
				if !door.IsOpen {
					// 閉じているドアは開く相互作用を実行
					params := actions.ActionParams{Actor: entity}
					executeActivity(world, &actions.InteractionActivateActivity{InteractableEntity: interactableEntity}, params)
					return
				}
				// 開いているドアは通過可能なので、相互作用を実行せずに下の移動処理に進む
			}
		} else {
			// ドア以外のOnCollision相互作用（会話など）は常に実行
			params := actions.ActionParams{Actor: entity}
			executeActivity(world, &actions.InteractionActivateActivity{InteractableEntity: interactableEntity}, params)
			return
		}
	}

	canMove := movement.CanMoveTo(world, newX, newY, entity)
	if canMove {
		destination := gc.Position{X: gc.Pixel(newX), Y: gc.Pixel(newY)}
		params := actions.ActionParams{
			Actor:       entity,
			Destination: &destination,
		}
		executeActivity(world, &actions.MoveActivity{}, params)
	}
}

// executeActivity はアクティビティ実行関数
func executeActivity(world w.World, actorImpl actions.ActivityInterface, params actions.ActionParams) {
	manager := actions.NewActivityManager(logger.New(logger.CategoryAction))

	result, err := manager.Execute(actorImpl, params, world)
	if err != nil {
		_ = result // エラーの場合は結果を使用しない
		return
	}

	// 会話の場合は会話メッセージを表示するStateEventを設定
	if _, isTalkActivity := actorImpl.(*actions.TalkActivity); isTalkActivity && result != nil && result.Success && params.Target != nil {
		targetEntity := *params.Target
		if targetEntity.HasComponent(world.Components.Dialog) {
			dialog := world.Components.Dialog.Get(targetEntity).(*gc.Dialog)
			world.Resources.Dungeon.SetStateEvent(resources.ShowDialogEvent{
				MessageKey:    dialog.MessageKey,
				SpeakerEntity: targetEntity,
			})
		}
	}

	// 移動の場合は追加でタイルイベントをチェック
	if _, isMoveActivity := actorImpl.(*actions.MoveActivity); isMoveActivity && result != nil && result.Success && params.Destination != nil {
		checkTileEvents(world, params.Actor, int(params.Destination.X), int(params.Destination.Y))
	}
}

// ExecuteWaitAction は待機アクションを実行する
func ExecuteWaitAction(world w.World) {
	entity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return
	}

	params := actions.ActionParams{
		Actor:    entity,
		Duration: 1,
		Reason:   "プレイヤー待機",
	}
	executeActivity(world, &actions.WaitActivity{}, params)
}

// ExecuteEnterAction は直上タイルの相互作用を実行する
func ExecuteEnterAction(world w.World) {
	entity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return
	}

	if !entity.HasComponent(world.Components.GridElement) {
		return
	}

	gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

	interactable, interactableEntity := getInteractableAtSameTile(world, gridElement)
	if interactable != nil && interactable.Data.Config().ActivationRange == gc.ActivationRangeSameTile {
		params := actions.ActionParams{Actor: entity}
		executeActivity(world, &actions.InteractionActivateActivity{InteractableEntity: interactableEntity}, params)
	}
}

// checkTileEvents はタイル上のイベントをチェックする
func checkTileEvents(world w.World, entity ecs.Entity, tileX, tileY int) {
	// プレイヤーの場合のみタイルイベントをチェック
	if entity.HasComponent(world.Components.Player) {
		gridElement := &gc.GridElement{X: gc.Tile(tileX), Y: gc.Tile(tileY)}

		// 手動相互作用のメッセージ表示
		showTileInteractionMessage(world, gridElement)
	}
}

// getInteractableAtSameTile はプレイヤーの直上タイルのInteractableとエンティティを取得する
// 複数ある場合は最初に見つかったものを返す
func getInteractableAtSameTile(world w.World, playerGrid *gc.GridElement) (*gc.Interactable, ecs.Entity) {
	var interactable *gc.Interactable
	var interactableEntity ecs.Entity
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.Interactable,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if interactable != nil {
			return // 既に見つかっている
		}
		ge := world.Components.GridElement.Get(entity).(*gc.GridElement)
		// 直上タイルのみ
		if ge.X == playerGrid.X && ge.Y == playerGrid.Y {
			interactable = world.Components.Interactable.Get(entity).(*gc.Interactable)
			interactableEntity = entity
		}
	}))
	return interactable, interactableEntity
}

// getInteractableInRange はプレイヤーの範囲内のInteractableとエンティティを取得する
// 複数ある場合は最初に見つかったものを返す
func getInteractableInRange(world w.World, playerGrid *gc.GridElement) (*gc.Interactable, ecs.Entity) {
	var interactable *gc.Interactable
	var interactableEntity ecs.Entity
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.Interactable,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if interactable != nil {
			return // 既に見つかっている
		}
		i := world.Components.Interactable.Get(entity).(*gc.Interactable)
		ge := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// ActivationRangeに応じた範囲チェック
		if worldhelper.IsInActivationRange(playerGrid, ge, i.Data.Config().ActivationRange) {
			interactable = i
			interactableEntity = entity
		}
	}))
	return interactable, interactableEntity
}

// getAllInteractiveInteractablesInRange はプレイヤーの範囲内の全てのインタラクティブなInteractableエンティティを取得する
// Manual と OnCollision 方式のInteractableが対象
func getAllInteractiveInteractablesInRange(world w.World, playerGrid *gc.GridElement) []ecs.Entity {
	var results []ecs.Entity

	world.Manager.Join(
		world.Components.GridElement,
		world.Components.Interactable,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		interactable := world.Components.Interactable.Get(entity).(*gc.Interactable)
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		way := interactable.Data.Config().ActivationWay
		// ManualまたはOnCollision方式で、範囲内にあるものを取得
		if (way == gc.ActivationWayManual || way == gc.ActivationWayOnCollision) &&
			worldhelper.IsInActivationRange(playerGrid, gridElement, interactable.Data.Config().ActivationRange) {
			results = append(results, entity)
		}
	}))

	return results
}

// getDirectionLabel はプレイヤーからターゲットへの方向ラベルを取得する
func getDirectionLabel(playerGrid, targetGrid *gc.GridElement) string {
	dx := int(targetGrid.X) - int(playerGrid.X)
	dy := int(targetGrid.Y) - int(playerGrid.Y)

	// 同じタイル
	if dx == 0 && dy == 0 {
		return "直上"
	}

	// 8方向を判定
	if dy < 0 {
		if dx < 0 {
			return "左上"
		} else if dx > 0 {
			return "右上"
		}
		return "上"
	} else if dy > 0 {
		if dx < 0 {
			return "左下"
		} else if dx > 0 {
			return "右下"
		}
		return "下"
	}
	if dx < 0 {
		return "左"
	}
	return "右"
}

// showTileInteractionMessage は手動相互作用のメッセージを表示する
func showTileInteractionMessage(world w.World, playerGrid *gc.GridElement) {
	interactable, interactableEntity := getInteractableInRange(world, playerGrid)
	if interactable == nil {
		return
	}

	if interactable.Data.Config().ActivationWay != gc.ActivationWayManual {
		return
	}

	switch interactable.Data.(type) {
	case gc.WarpNextInteraction:
		gamelog.New(gamelog.FieldLog).
			Append("転移ゲートがある。Enterキーで移動。").
			Log()
	case gc.WarpEscapeInteraction:
		gamelog.New(gamelog.FieldLog).
			Append("脱出ゲートがある。Enterキーで移動。").
			Log()
	case gc.ItemInteraction:
		formattedName := worldhelper.FormatItemName(world, interactableEntity)
		gamelog.New(gamelog.FieldLog).
			Append(formattedName).
			Append(" がある。").
			Log()
	}
}

// InteractionAction はインタラクション可能なアクション情報
type InteractionAction struct {
	Label    string                    // 表示ラベル（例："開く(上)"）
	Activity actions.ActivityInterface // 実行するアクティビティ
	Target   ecs.Entity                // ターゲットエンティティ
}

// getInteractionActions はInteractableに対応するアクションを取得する
func getInteractionActions(world w.World, interactable *gc.Interactable, interactableEntity ecs.Entity, dirLabel string) []InteractionAction {
	var result []InteractionAction

	switch interactable.Data.(type) {
	case gc.WarpNextInteraction:
		result = append(result, InteractionAction{
			Label:    "転移(" + dirLabel + ")",
			Activity: &actions.InteractionActivateActivity{InteractableEntity: interactableEntity},
			Target:   interactableEntity,
		})
	case gc.WarpEscapeInteraction:
		result = append(result, InteractionAction{
			Label:    "脱出(" + dirLabel + ")",
			Activity: &actions.InteractionActivateActivity{InteractableEntity: interactableEntity},
			Target:   interactableEntity,
		})
	case gc.DoorInteraction:
		// ドアの状態に応じたアクションを生成
		if interactableEntity.HasComponent(world.Components.Door) {
			door := world.Components.Door.Get(interactableEntity).(*gc.Door)
			if door.IsOpen {
				result = append(result, InteractionAction{
					Label:    "閉じる(" + dirLabel + ")",
					Activity: &actions.CloseDoorActivity{},
					Target:   interactableEntity,
				})
			} else {
				result = append(result, InteractionAction{
					Label:    "開く(" + dirLabel + ")",
					Activity: &actions.OpenDoorActivity{},
					Target:   interactableEntity,
				})
			}
		}
	case gc.TalkInteraction:
		// 会話アクションを生成
		if interactableEntity.HasComponent(world.Components.Name) {
			name := world.Components.Name.Get(interactableEntity).(*gc.Name)
			result = append(result, InteractionAction{
				Label:    "話しかける(" + name.Name + ")",
				Activity: &actions.TalkActivity{},
				Target:   interactableEntity,
			})
		}
	case gc.ItemInteraction:
		// アイテム拾得アクションを生成
		formattedName := worldhelper.FormatItemName(world, interactableEntity)
		result = append(result, InteractionAction{
			Label:    "拾う(" + formattedName + ")",
			Activity: &actions.PickupActivity{},
			Target:   interactableEntity,
		})
	}

	return result
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
	interactableEntities := getAllInteractiveInteractablesInRange(world, gridElement)
	for _, interactableEntity := range interactableEntities {
		if !interactableEntity.HasComponent(world.Components.GridElement) {
			continue
		}
		if !interactableEntity.HasComponent(world.Components.Interactable) {
			continue
		}

		interactableGrid := world.Components.GridElement.Get(interactableEntity).(*gc.GridElement)
		interactable := world.Components.Interactable.Get(interactableEntity).(*gc.Interactable)
		dirLabel := getDirectionLabel(gridElement, interactableGrid)
		actions := getInteractionActions(world, interactable, interactableEntity, dirLabel)
		interactionActions = append(interactionActions, actions...)
	}

	return interactionActions
}
