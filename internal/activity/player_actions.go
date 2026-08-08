package activity

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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
		return fmt.Errorf("player has no GridElement component")
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
							gamelog.New(query.GetGameLog(world)).Markup(query.T(world, "The door is locked.")).Log()
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
			case gc.InteractionTalk, gc.InteractionCubePanel:
				// 会話とコントロールパネルは、歩き込むだけで発動する
				_, err := ExecuteInteraction(entity, interactableEntity, interaction, world)
				return err
			default:
				// 衝突時に自動発動しない種類はここでは扱わない
			}
		}
	}

	// 移動先に押せるキューブがあれば、通行でなく押しになる。キューブは BlockPass なので
	// 通常の CanMoveTo では弾かれる。歩き込みは押し、入るは手動アクションと入力経路を分ける。
	// 押しはキューブだけを動かす。プレイヤーの追随は次入力の通常移動が担い、方向を押し続けると
	// 押しと一歩が交互に起きてキューブが進む
	if cube, ok := pushableAt(world, next); ok {
		// 押し先が塞がっていれば、壁に歩き込むのと同じく何もしない。エラーにすると
		// 入力層で致命化するため、実行可否をここで判定して不可なら no-op にする
		cubeCoord := world.Components.GridElement.Get(cube).Coord
		if !CanMoveTo(world, cubeCoord.Add(direction.GetDelta()), cubeCoord, cube) {
			return nil
		}
		comp, err := NewPushActivity(cube, direction, world)
		if err != nil {
			return err
		}
		_, err = Execute(comp, entity, world)
		return err
	}

	canMove := CanMoveTo(world, next, current, entity)
	if canMove {
		destination := gc.GridElement{Coord: next}
		_, err := Execute(NewMoveActivity(destination), entity, world)
		// 重量超過はプレイヤーの通常状態。エラーにすると入力層で致命化するため、
		// 壁への歩き込みと同じく no-op にする。理由のログは Validate が既に出している
		if errors.Is(err, ErrMoveOverweight) {
			return nil
		}
		return err
	}

	return nil
}

// pushableAt は指定タイルにある押せるキューブを返す。無ければ ok=false。
func pushableAt(world w.World, coord consts.Coord[consts.Tile]) (ecs.Entity, bool) {
	for _, entity := range query.GetEntitiesAt(world, coord.X, coord.Y) {
		if world.Components.Pushable.Has(entity) {
			return entity, true
		}
	}
	return ecs.Entity{}, false
}

// ExecuteWaitAction は待機アクションを実行する
func ExecuteWaitAction(world w.World) error {
	entity, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	_, err = Execute(NewWaitActivity(1), entity, world)
	return err
}

// getInteractableAtSameTile は指定タイルのInteractableとエンティティを取得する。
// 複数ある場合は最初に見つかったものを返す。
// 見つからない場合は interactable が nil になる。interactable != nil のときのみ entity は有効値。
func getInteractableAtSameTile(world w.World, targetGrid *gc.GridElement) (*gc.Interactable, ecs.Entity) {
	var found *gc.Interactable
	var foundEntity ecs.Entity
	interactableQuery := query.ActiveFilter2[gc.GridElement, gc.Interactable](world).Without(ecs.C[gc.Dead]()).Query()
	for interactableQuery.Next() {
		entity := interactableQuery.Entity()
		if found != nil {
			// 先着1件を採用する。途中 return せず反復は最後まで続ける。Ark のワールドロックを外すため
			continue
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

	if d.X == 0 && d.Y == 0 {
		return "here"
	}

	// 8方向を判定
	if d.Y < 0 {
		if d.X < 0 {
			return "upper left"
		} else if d.X > 0 {
			return "upper right"
		}
		return "up"
	} else if d.Y > 0 {
		if d.X < 0 {
			return "lower left"
		} else if d.X > 0 {
			return "lower right"
		}
		return "down"
	}
	if d.X < 0 {
		return "left"
	}
	return "right"
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
					Markup(query.T(world, "%s is here.", gamelog.Tag("item", formattedName))).
					Log()
			case gc.InteractionPortalNext:
				gamelog.New(query.GetGameLog(world)).
					Markup(query.T(world, "There is a warp gate. Press Enter to move.")).
					Log()
			case gc.InteractionPortalPrev:
				gamelog.New(query.GetGameLog(world)).
					Markup(query.T(world, "There is an up staircase. Press Enter to move.")).
					Log()
			case gc.InteractionDungeonEnter:
				gamelog.New(query.GetGameLog(world)).
					Markup(query.T(world, "There is a ruins entrance. Press Enter to enter.")).
					Log()
			case gc.InteractionEnterCube:
				gamelog.New(query.GetGameLog(world)).
					Markup(query.T(world, "%s is here. You can enter it from the Space action menu.", gamelog.Tag("item", query.GetEntityName(entity, world)))).
					Log()
			case gc.InteractionDoor, gc.InteractionDoorLock, gc.InteractionTalk, gc.InteractionItemAll, gc.InteractionStorage, gc.InteractionMelee, gc.InteractionDisassemble, gc.InteractionExitCube, gc.InteractionPullCube, gc.InteractionCubePanel:
				// 足元ログを出さない種類。default を置かず exhaustive に全種別を
				// 明示させ、新しい InteractionKind の対応漏れを lint で検知する
			}
		}
	}
}
