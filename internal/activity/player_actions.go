package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
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
		return fmt.Errorf("プレイヤーにGridElementコンポーネントがありません")
	}

	gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
	currentX := int(gridElement.X)
	currentY := int(gridElement.Y)

	deltaX, deltaY := direction.GetDelta()
	newX := currentX + deltaX
	newY := currentY + deltaY

	// 移動先にOnCollision方式のInteractableがある場合は自動実行
	targetGrid := &gc.GridElement{X: consts.Tile(newX), Y: consts.Tile(newY)}
	interactable, interactableEntity := getInteractableAtSameTile(world, targetGrid)

	if interactable != nil {
		for _, interaction := range interactable.Interactions {
			if interaction.Config().ActivationWay != gc.ActivationWayOnCollision {
				continue
			}
			switch interaction.(type) {
			case gc.DoorInteraction:
				// DoorInteractionの場合は、閉じている場合のみ実行（開いている場合は通過）
				if interactableEntity.HasComponent(world.Components.Door) {
					door := world.Components.Door.Get(interactableEntity).(*gc.Door)
					if !door.IsOpen {
						if door.Locked {
							gamelog.New(worldhelper.GetGameLog(world)).Append("扉はロックされている。").Log()
							return nil
						}
						_, err := ExecuteInteraction(entity, interactableEntity, interaction, world)
						return err
					}
				}
			case gc.MeleeInteraction:
				if isHostileEntity(interactableEntity, world) {
					_, err := ExecuteInteraction(entity, interactableEntity, interaction, world)
					return err
				}
			case gc.TalkInteraction:
				_, err := ExecuteInteraction(entity, interactableEntity, interaction, world)
				return err
			}
		}
	}

	canMove := CanMoveTo(world, newX, newY, currentX, currentY, entity)
	if canMove {
		destination := gc.GridElement{X: consts.Tile(newX), Y: consts.Tile(newY)}
		params := ActionParams{
			Actor:       entity,
			Destination: &destination,
		}
		_, err := Execute(&MoveActivity{}, params, world)
		return err
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
	_, err = Execute(&WaitActivity{}, params, world)
	return err
}

// getInteractableAtSameTile は指定タイルのInteractableとエンティティを取得する
// 複数ある場合は最初に見つかったものを返す
func getInteractableAtSameTile(world w.World, targetGrid *gc.GridElement) (*gc.Interactable, ecs.Entity) {
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

		for _, interaction := range interactable.Interactions {
			way := interaction.Config().ActivationWay
			if (way == gc.ActivationWayManual || way == gc.ActivationWayOnCollision) &&
				worldhelper.IsInActivationRange(targetGrid, gridElement, interaction.Config().ActivationRange) {
				results = append(results, entity)
				return // 同じエンティティを重複追加しない
			}
		}
	}))

	return results
}

// isHostileEntity はエンティティがプレイヤーに敵対しているかを判定する
func isHostileEntity(entity ecs.Entity, world w.World) bool {
	d := world.Components.Disposition.Get(entity)
	if d == nil {
		return false
	}
	return d.(*gc.Disposition).Current == gc.DispositionHostile
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

// showTileInteractionMessage は範囲内の全Manual相互作用のメッセージを表示する
func showTileInteractionMessage(world w.World, playerGrid *gc.GridElement) {
	entities := GetAllInteractiveInteractablesInRange(world, playerGrid)
	for _, entity := range entities {
		interactable := world.Components.Interactable.Get(entity).(*gc.Interactable)
		for _, interaction := range interactable.Interactions {
			if interaction.Config().ActivationWay != gc.ActivationWayManual {
				continue
			}
			switch data := interaction.(type) {
			case gc.ItemInteraction:
				formattedName := worldhelper.FormatItemName(world, entity)
				gamelog.New(worldhelper.GetGameLog(world)).
					ItemName(formattedName).
					Append(" がある。").
					Log()
			case gc.PortalInteraction:
				switch data.PortalType {
				case gc.PortalTypeNext:
					gamelog.New(worldhelper.GetGameLog(world)).
						Append("転移ゲートがある。Enterキーで移動。").
						Log()
				case gc.PortalTypeTown:
					gamelog.New(worldhelper.GetGameLog(world)).
						Append("帰還ゲートがある。Enterキーで脱出。").
						Log()
				}
			case gc.DungeonGateInteraction:
				gamelog.New(worldhelper.GetGameLog(world)).
					Append("ダンジョンへの門がある。Enterキーで選択。").
					Log()
			}
		}
	}
}
