package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ExecuteMoveAction は移動アクションを実行する
func ExecuteMoveAction(world w.World, direction gc.Direction) error {
	entity, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	if !world.Components.GridElement.Has(entity) {
		return fmt.Errorf("プレイヤーにGridElementコンポーネントがありません")
	}

	gridElement := world.Components.GridElement.Get(entity)
	current := gridElement.Coord

	next := current.Add(direction.GetDelta())

	// 移動先にOnCollision方式のInteractableがある場合は自動実行
	targetGrid := &gc.GridElement{Coord: next}
	interactable, interactableEntity := getInteractableAtSameTile(world, targetGrid)

	if interactable != nil {
		for _, interaction := range interactable.Interactions {
			if interaction.Config().ActivationWay != gc.ActivationWayOnCollision {
				continue
			}
			switch interaction {
			case gc.InteractionDoor:
				// 扉は閉じている場合のみ実行（開いている場合は通過）
				if world.Components.Door.Has(interactableEntity) {
					door := world.Components.Door.Get(interactableEntity)
					if !door.IsOpen {
						if door.Locked {
							gamelog.New(query.GetGameLog(world)).Append("扉はロックされている。").Log()
							return nil
						}
						_, err := ExecuteInteraction(entity, interactableEntity, interaction, world)
						return err
					}
				}
			case gc.InteractionMelee:
				if query.FactionRelation(world, entity, interactableEntity) == query.RelationHostile {
					_, err := ExecuteInteraction(entity, interactableEntity, interaction, world)
					return err
				}
			case gc.InteractionTalk:
				_, err := ExecuteInteraction(entity, interactableEntity, interaction, world)
				return err
			default:
				// 衝突時に自動発動しない種類はここでは扱わない
			}
		}
	}

	canMove := CanMoveTo(world, next, current, entity)
	if canMove {
		destination := gc.GridElement{Coord: next}
		_, err := Execute(&MoveActivity{Destination: destination}, entity, world)
		return err
	}

	return nil
}

// ExecuteWaitAction は待機アクションを実行する
func ExecuteWaitAction(world w.World) error {
	entity, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	_, err = Execute(&WaitActivity{Duration: 1, Reason: "プレイヤー待機"}, entity, world)
	return err
}

// getInteractableAtSameTile は指定タイルのInteractableとエンティティを取得する。
// 複数ある場合は最初に見つかったものを返す。
// 見つからない場合は interactable が nil になる。interactable != nil のときのみ entity は有効値。
func getInteractableAtSameTile(world w.World, targetGrid *gc.GridElement) (*gc.Interactable, ecs.Entity) {
	var found *gc.Interactable
	var foundEntity ecs.Entity
	interactableQuery := query.ActiveFilter2[gc.GridElement, gc.Interactable](world, ecs.C[gc.Dead]()).Query()
	for interactableQuery.Next() {
		entity := interactableQuery.Entity()
		if found != nil {
			continue // 既に見つかっている
		}
		ge := world.Components.GridElement.Get(entity)
		// 直上タイルのみ
		if ge.X == targetGrid.X && ge.Y == targetGrid.Y {
			found = world.Components.Interactable.Get(entity)
			foundEntity = entity
		}
	}
	return found, foundEntity
}

// GetAllInteractiveInteractablesInRange は範囲内の全てのインタラクティブなInteractableエンティティを取得する
// Manual と OnCollision 方式のInteractableが対象
func GetAllInteractiveInteractablesInRange(world w.World, targetGrid *gc.GridElement) []ecs.Entity {
	var results []ecs.Entity

	rangeQuery := query.ActiveFilter2[gc.GridElement, gc.Interactable](world).Query()
	for rangeQuery.Next() {
		entity := rangeQuery.Entity()
		interactable := world.Components.Interactable.Get(entity)
		gridElement := world.Components.GridElement.Get(entity)

		for _, interaction := range interactable.Interactions {
			way := interaction.Config().ActivationWay
			if (way == gc.ActivationWayManual || way == gc.ActivationWayOnCollision) &&
				query.IsInActivationRange(targetGrid, gridElement, interaction.Config().ActivationRange) {
				results = append(results, entity)
				break // 同じエンティティを重複追加しない
			}
		}
	}

	return results
}

// GetDirectionLabel はプレイヤーからターゲットへの方向ラベルを取得する
func GetDirectionLabel(playerGrid, targetGrid *gc.GridElement) string {
	d := targetGrid.Sub(playerGrid.Coord)

	// 同じタイル
	if d.X == 0 && d.Y == 0 {
		return "直上"
	}

	// 8方向を判定
	if d.Y < 0 {
		if d.X < 0 {
			return "左上"
		} else if d.X > 0 {
			return "右上"
		}
		return "上"
	} else if d.Y > 0 {
		if d.X < 0 {
			return "左下"
		} else if d.X > 0 {
			return "右下"
		}
		return "下"
	}
	if d.X < 0 {
		return "左"
	}
	return "右"
}

// showTileInteractionMessage は範囲内の全Manual相互作用のメッセージを表示する
func showTileInteractionMessage(world w.World, playerGrid *gc.GridElement) {
	entities := GetAllInteractiveInteractablesInRange(world, playerGrid)
	for _, entity := range entities {
		interactable := world.Components.Interactable.Get(entity)
		for _, interaction := range interactable.Interactions {
			if interaction.Config().ActivationWay != gc.ActivationWayManual {
				continue
			}
			switch interaction {
			case gc.InteractionItem:
				formattedName := query.FormatItemName(world, entity)
				gamelog.New(query.GetGameLog(world)).
					ItemName(formattedName).
					Append(" がある。").
					Log()
			case gc.InteractionPortalNext:
				gamelog.New(query.GetGameLog(world)).
					Append("転移ゲートがある。Enterキーで移動。").
					Log()
			case gc.InteractionPortalTown:
				gamelog.New(query.GetGameLog(world)).
					Append("帰還ゲートがある。Enterキーで脱出。").
					Log()
			case gc.InteractionDungeonGate:
				gamelog.New(query.GetGameLog(world)).
					Append("ダンジョンへの門がある。Enterキーで選択。").
					Log()
			default:
				// ログ表示対象外の種類は何もしない
			}
		}
	}
}
