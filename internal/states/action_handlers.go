package states

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
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

	var actions []InteractionAction
	var itemEntities []ecs.Entity

	// インタラクティブな相互作用を全て取得してアクションを生成
	interactableEntities := activity.GetAllInteractiveInteractablesInRange(world, gridElement)
	for _, interactableEntity := range interactableEntities {
		if !world.Components.GridElement.Has(interactableEntity) {
			continue
		}
		if !world.Components.Interactable.Has(interactableEntity) {
			continue
		}

		interactable := world.Components.Interactable.Get(interactableEntity)
		// スタック単位の種別を持つ実体は束ね経路へ集める。拾得は同タイルに限られるので
		// 位置を無視して束ねても別タイルの同種が混ざることはない
		bundled, individual := splitByMenuUnit(interactable.Interactions)
		if len(bundled) > 0 {
			itemEntities = append(itemEntities, interactableEntity)
		}
		if len(individual) > 0 {
			interactableGrid := world.Components.GridElement.Get(interactableEntity)
			dirLabel := query.T(world, activity.GetDirectionLabel(gridElement, interactableGrid))
			actions = append(actions, getInteractionActions(world, &gc.Interactable{Interactions: individual}, interactableEntity, dirLabel)...)
		}
	}

	actions = appendItemPickupActions(world, actions, itemEntities)
	return appendIgniteActions(world, playerEntity, gridElement, actions)
}

// appendIgniteActions は火種を持つとき、隣接タイルの燃焼物へ着火するアクションをタイルごとに1つ足す。
// 足元は自分が燃えるため対象にしない。同じタイルに複数の燃料があっても着火は1タイル1回にまとめる。
// 火種の道具は再利用できるので消費しない。
func appendIgniteActions(world w.World, player ecs.Entity, playerGrid *gc.GridElement, actions []InteractionAction) []InteractionAction {
	if !query.HoldsFireStarter(world, player) {
		return actions
	}
	// タイルごとに代表の燃料を1つ選ぶ。map の走査順に依存しないよう出現順を別に記録する
	repByTile := map[consts.Coord[consts.Tile]]ecs.Entity{}
	var order []consts.Coord[consts.Tile]
	fuelQuery := query.ActiveFilter2[gc.Fuel, gc.LocationOnField](world).Query()
	for fuelQuery.Next() {
		entity := fuelQuery.Entity()
		if !world.Components.GridElement.Has(entity) {
			continue
		}
		coord := world.Components.GridElement.Get(entity).Coord
		if !geometry.IsAdjacent(playerGrid.Coord, coord) {
			continue
		}
		if _, seen := repByTile[coord]; !seen {
			repByTile[coord] = entity
			order = append(order, coord)
		}
	}
	for _, coord := range order {
		rep := repByTile[coord]
		grid := world.Components.GridElement.Get(rep)
		dirLabel := query.T(world, activity.GetDirectionLabel(playerGrid, grid))
		actions = append(actions, InteractionAction{
			Label:       query.T(world, "Light fire (%s)", dirLabel),
			Target:      rep,
			Interaction: gc.InteractionIgnite,
		})
	}
	return actions
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
	var itemEntities []ecs.Entity
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
		if len(filtered) == 0 {
			continue
		}
		bundled, individual := splitByMenuUnit(filtered)
		if len(bundled) > 0 {
			itemEntities = append(itemEntities, entity)
		}
		if len(individual) > 0 {
			actions = append(actions, getInteractionActions(world, &gc.Interactable{Interactions: individual}, entity, query.T(world, "directly above"))...)
		}
	}

	return appendItemPickupActions(world, actions, itemEntities)
}

// splitByMenuUnit は種別を、行の単位の宣言 Config().MenuUnit に従ってスタック単位と
// エンティティ単位へ分ける。一覧を組む関数はこの振り分けに従う。
// 両方を持つ実体は両経路に載り、束ね行と個別行が並ぶ。
// switch に default を置かず、単位の追加漏れを exhaustive linter に検知させる。
// 未知の種別の単位はゼロ値で raw/save 由来でありうるので、行を組まず黙って落とす
func splitByMenuUnit(kinds []gc.InteractionKind) (stack, entity []gc.InteractionKind) {
	for _, kind := range kinds {
		switch kind.Config().MenuUnit {
		case gc.MenuUnitStack:
			stack = append(stack, kind)
		case gc.MenuUnitEntity:
			entity = append(entity, kind)
		}
	}
	return stack, entity
}

// appendItemPickupActions は itemEntities をスタックごとに1行へ束ねて拾得アクションを足す。
// 代表を target にし、拾得はスタック丸ごとを対象にする。スタックが2種類以上なら「すべて拾う」を先頭に付ける。
// 床の同種が1個1エンティティで個数ぶん並ぶのを防ぐため、アイテムを一覧へ出す関数は必ずこれを通す。
func appendItemPickupActions(world w.World, actions []InteractionAction, itemEntities []ecs.Entity) []InteractionAction {
	stacks := query.GroupStacks(world, itemEntities)
	for _, stack := range stacks {
		label := query.FormatNameCount(query.GetEntityName(stack.Rep, world), stack.Count)
		actions = append(actions, InteractionAction{
			Label:       query.T(world, "Pick up (%s)", label),
			Target:      stack.Rep,
			Interaction: gc.InteractionItem,
		})
	}
	if len(stacks) >= 2 {
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
			// スタック単位の種別は splitByMenuUnit で束ね経路へ振られ、ここには来ない。
			// 拾得行は appendItemPickupActions がスタック単位で組む
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
				Label:       query.T(world, "Open shipping station"),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionItemAll:
			// アクションメニューに出さない種類。default を置かず exhaustive に全種別を
			// 明示させ、新しい InteractionKind の対応漏れを lint で検知する
		case gc.InteractionIgnite:
			// 着火はエンティティの Interactable でなく appendIgniteActions が隣接タイル走査で
			// タイル単位に組む。ここには来ないが exhaustive のため case を明示する
		}
	}

	return result
}
