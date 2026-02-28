package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/movement"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ExecuteMoveAction は移動アクションを実行する
func ExecuteMoveAction(world w.World, direction gc.Direction) error {
	entity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	if !entity.HasComponent(world.Components.GridElement) {
		return nil
	}

	gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
	currentX := int(gridElement.X)
	currentY := int(gridElement.Y)

	deltaX, deltaY := direction.GetDelta()
	newX := currentX + deltaX
	newY := currentY + deltaY

	// 移動先にOnCollision方式のInteractableがある場合は自動実行
	targetGrid := &gc.GridElement{X: consts.Tile(newX), Y: consts.Tile(newY)}
	interactable, interactableEntity := GetInteractableAtSameTile(world, targetGrid)

	if interactable != nil && interactable.Data.Config().ActivationWay == gc.ActivationWayOnCollision {
		// DoorInteractionの場合は、閉じている場合のみ実行（開いている場合は通過）
		if _, isDoorInteraction := interactable.Data.(gc.DoorInteraction); isDoorInteraction {
			if interactableEntity.HasComponent(world.Components.Door) {
				door := world.Components.Door.Get(interactableEntity).(*gc.Door)
				if !door.IsOpen {
					// 閉じているドアは開く相互作用を実行
					return executeInteractionForPlayer(world, entity, interactableEntity)
				}
				// 開いているドアは通過可能なので、相互作用を実行せずに下の移動処理に進む
			}
		} else {
			// ドア以外のOnCollision相互作用（攻撃など）を実行
			return executeInteractionForPlayer(world, entity, interactableEntity)
		}
	}

	canMove := movement.CanMoveTo(world, newX, newY, entity)
	if canMove {
		destination := gc.Position{X: consts.Pixel(newX), Y: consts.Pixel(newY)}
		params := ActionParams{
			Actor:       entity,
			Destination: &destination,
		}
		return executeActivityWithPostProcess(world, &MoveActivity{}, params)
	}

	return nil
}

// ExecuteWaitAction は待機アクションを実行する
func ExecuteWaitAction(world w.World) error {
	entity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	params := ActionParams{
		Actor:    entity,
		Duration: 1,
		Reason:   "プレイヤー待機",
	}
	return executeActivityWithPostProcess(world, &WaitActivity{}, params)
}

// ExecuteEnterAction は直上タイルの相互作用を実行する
func ExecuteEnterAction(world w.World) error {
	entity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	if !entity.HasComponent(world.Components.GridElement) {
		return nil
	}

	gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

	interactable, interactableEntity := GetInteractableAtSameTile(world, gridElement)
	if interactable != nil {
		config := interactable.Data.Config()
		// 手動発動（Enterキー）かつ同タイルのみ実行
		if config.ActivationRange == gc.ActivationRangeSameTile && config.ActivationWay == gc.ActivationWayManual {
			return executeInteractionForPlayer(world, entity, interactableEntity)
		}
	}

	return nil
}

// executeInteractionForPlayer は相互作用を実行するヘルパー関数
func executeInteractionForPlayer(world w.World, actor ecs.Entity, interactable ecs.Entity) error {
	manager := world.Resources.ActivityManager.(*ActivityManager)
	_, err := ExecuteInteraction(manager, actor, interactable, world)
	return err
}

// executeActivityWithPostProcess はアクティビティ実行と後処理を行う関数
func executeActivityWithPostProcess(world w.World, actorImpl ActivityInterface, params ActionParams) error {
	manager := world.Resources.ActivityManager.(*ActivityManager)
	result, err := manager.Execute(actorImpl, params, world)
	if err != nil {
		return err
	}

	// 会話の場合は会話メッセージを表示する状態変更を要求
	if _, isTalkActivity := actorImpl.(*TalkActivity); isTalkActivity && result != nil && result.Success && params.Target != nil {
		targetEntity := *params.Target
		if targetEntity.HasComponent(world.Components.Dialog) {
			dialog := world.Components.Dialog.Get(targetEntity).(*gc.Dialog)
			if err := world.Resources.Dungeon.RequestStateChange(resources.ShowDialogEvent{
				MessageKey:    dialog.MessageKey,
				SpeakerEntity: targetEntity,
			}); err != nil {
				return fmt.Errorf("会話状態変更要求エラー: %w", err)
			}
		}
	}

	// 移動の場合は追加でタイルイベントをチェック
	if _, isMoveActivity := actorImpl.(*MoveActivity); isMoveActivity && result != nil && result.Success && params.Destination != nil {
		checkTileEvents(world, params.Actor, int(params.Destination.X), int(params.Destination.Y))
	}

	return nil
}

// checkTileEvents はタイル上のイベントをチェックする
func checkTileEvents(world w.World, entity ecs.Entity, tileX, tileY int) {
	// プレイヤーの場合のみタイルイベントをチェック
	if entity.HasComponent(world.Components.Player) {
		gridElement := &gc.GridElement{X: consts.Tile(tileX), Y: consts.Tile(tileY)}

		// 手動相互作用のメッセージ表示
		showTileInteractionMessage(world, gridElement)
	}
}

// GetInteractableAtSameTile は指定タイルのInteractableとエンティティを取得する
// 複数ある場合は最初に見つかったものを返す
func GetInteractableAtSameTile(world w.World, targetGrid *gc.GridElement) (*gc.Interactable, ecs.Entity) {
	var interactable *gc.Interactable
	var interactableEntity ecs.Entity
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.Interactable,
		world.Components.Dead.Not(),
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if interactable != nil {
			return // 既に見つかっている
		}
		ge := world.Components.GridElement.Get(entity).(*gc.GridElement)
		// 直上タイルのみ
		if ge.X == targetGrid.X && ge.Y == targetGrid.Y {
			interactable = world.Components.Interactable.Get(entity).(*gc.Interactable)
			interactableEntity = entity
		}
	}))
	return interactable, interactableEntity
}

// GetInteractableInRange は範囲内のInteractableとエンティティを取得する
// 複数ある場合は最初に見つかったものを返す
func GetInteractableInRange(world w.World, targetGrid *gc.GridElement) (*gc.Interactable, ecs.Entity) {
	var interactable *gc.Interactable
	var interactableEntity ecs.Entity
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.Interactable,
		world.Components.Dead.Not(),
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if interactable != nil {
			return // 既に見つかっている
		}
		i := world.Components.Interactable.Get(entity).(*gc.Interactable)
		ge := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// ActivationRangeに応じた範囲チェック
		if worldhelper.IsInActivationRange(targetGrid, ge, i.Data.Config().ActivationRange) {
			interactable = i
			interactableEntity = entity
		}
	}))
	return interactable, interactableEntity
}

// GetAllInteractiveInteractablesInRange は範囲内の全てのインタラクティブなInteractableエンティティを取得する
// Manual と OnCollision 方式のInteractableが対象
func GetAllInteractiveInteractablesInRange(world w.World, targetGrid *gc.GridElement) []ecs.Entity {
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
			worldhelper.IsInActivationRange(targetGrid, gridElement, interactable.Data.Config().ActivationRange) {
			results = append(results, entity)
		}
	}))

	return results
}

// GetDirectionLabel はプレイヤーからターゲットへの方向ラベルを取得する
func GetDirectionLabel(playerGrid, targetGrid *gc.GridElement) string {
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
	interactable, interactableEntity := GetInteractableInRange(world, playerGrid)
	if interactable == nil {
		return
	}

	if interactable.Data.Config().ActivationWay != gc.ActivationWayManual {
		return
	}

	switch data := interactable.Data.(type) {
	case gc.ItemInteraction:
		formattedName := worldhelper.FormatItemName(world, interactableEntity)
		gamelog.New(gamelog.FieldLog).
			ItemName(formattedName).
			Append(" がある。").
			Log()
	case gc.PortalInteraction:
		switch data.PortalType {
		case gc.PortalTypeNext:
			gamelog.New(gamelog.FieldLog).
				Append("転移ゲートがある。Enterキーで移動。").
				Log()
		case gc.PortalTypeTown:
			gamelog.New(gamelog.FieldLog).
				Append("帰還ゲートがある。Enterキーで脱出。").
				Log()
		}
	case gc.DungeonGateInteraction:
		gamelog.New(gamelog.FieldLog).
			Append("ダンジョンへの門がある。Enterキーで選択。").
			Log()
	}
}
