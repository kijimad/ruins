package states

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
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
		interactableGrid := world.Components.GridElement.Get(interactableEntity)
		// 実体単位でなく相互作用単位で発動範囲を絞る。隣接に来た実体でも、直上でしか発動しない
		// 相互作用、例えばキューブメニュー InteractionOpenCubeMenu は隣接メニューへ出さない。範囲判定は
		// GetAllInteractiveInteractablesInRange と同じ IsInActivationRange を使う
		var inRange []gc.InteractionKind
		for _, interaction := range interactable.Interactions {
			config := interaction.Config()
			if (config.ActivationWay == gc.ActivationWayManual || config.ActivationWay == gc.ActivationWayOnCollision) &&
				query.IsInActivationRange(gridElement, interactableGrid, config.ActivationRange) {
				inRange = append(inRange, interaction)
			}
		}
		_, individual := splitByMenuUnit(inRange)
		if len(individual) > 0 {
			dirLabel := query.T(world, activity.GetDirectionLabel(gridElement, interactableGrid))
			actions = append(actions, getInteractionActions(world, &gc.Interactable{Interactions: individual}, interactableEntity, dirLabel)...)
		}
	}

	actions = appendIgniteActions(world, playerEntity, gridElement, actions)
	return appendFeedFuelActions(world, gridElement, actions)
}

// appendFeedFuelActions は隣接する、燃えている火へ給油するアクションを足す。
// 対象は Burning を持つ火だけ。着火した火は必ず Burning を持つので、これが火であることの判定を兼ねる。
// 燃料の有無に関わらず出し、残ターン数を確認しつつ、燃料を持っていればくべられるようにする。
func appendFeedFuelActions(world w.World, playerGrid *gc.GridElement, actions []InteractionAction) []InteractionAction {
	fireQuery := query.ActiveFilter2[gc.Burning, gc.GridElement](world).Query()
	for fireQuery.Next() {
		fire := fireQuery.Entity()
		grid := world.Components.GridElement.Get(fire)
		if !geometry.IsAdjacent(playerGrid.Coord, grid.Coord) {
			continue
		}
		dirLabel := query.T(world, activity.GetDirectionLabel(playerGrid, grid))
		actions = append(actions, InteractionAction{
			Label:       query.T(world, "Feed fire (%s)", dirLabel),
			Target:      fire,
			Interaction: gc.InteractionFeedFuel,
		})
	}
	return actions
}

// appendIgniteActions は火種を持つとき、隣接タイルの燃焼物へ着火するアクションをタイルごとに1つ足す。
// 足元は自分が燃えるため対象にしない。同じタイルに複数の燃料があっても着火は1タイル1回にまとめる。
// 着火の可否は火種の所持だけで判定する。火種を消費するかは別の関心で、ここでは扱わない。
func appendIgniteActions(world w.World, player ecs.Entity, playerGrid *gc.GridElement, actions []InteractionAction) []InteractionAction {
	if !query.HoldsFireStarter(world, player) {
		return actions
	}
	// タイルごとに代表の燃料を1つ選ぶ。map の走査順に依存しないよう出現順を別に記録する
	repByTile := map[consts.Coord[consts.Tile]]ecs.Entity{}
	var order []consts.Coord[consts.Tile]
	// 地面の燃焼物を隣接タイルで絞るので GridElement も条件に入れる。フィルタで保証されるので Get は安全
	fuelQuery := query.ActiveFilter3[gc.Material, gc.LocationOnField, gc.GridElement](world).Query()
	for fuelQuery.Next() {
		entity := fuelQuery.Entity()
		if !query.IsCombustible(world, entity) {
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

// startPickup は足元と周囲8タイルから拾う。アイテム1つだけならそのまま拾い、それ以外は拾うメニューを開く
func startPickup(world w.World) (es.Transition[w.World], error) {
	actions := nearbyPickupActions(world)
	switch {
	case len(actions) == 0:
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "There is nothing to pick up nearby.")).
			Log()
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case len(actions) == 1 && actions[0].Interaction == gc.InteractionItem:
		player, err := query.GetPlayerEntity(world)
		if err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		_, err = activity.ExecuteInteraction(player, actions[0].Target, actions[0].Interaction, world)
		return es.Transition[w.World]{Type: es.TransNone}, err
	}
	return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
		func() (es.State[w.World], error) {
			return NewChoiceMenu(pickupChoices), nil
		},
	}}, nil
}

// pickupChoices は拾うメニューの選択肢を返す。収納画面はメニューの上に開き、閉じるとメニューへ戻る
func pickupChoices(world w.World) (string, []Choice) {
	actions := nearbyPickupActions(world)
	choices := make([]Choice, 0, len(actions))
	for _, a := range actions {
		if a.Interaction == gc.InteractionStorage {
			choices = append(choices, Choice{Label: a.Label, Run: pushChoice(func() (es.State[w.World], error) {
				return NewStorageMenuState(a.Target)
			})})
			continue
		}
		choices = append(choices, Choice{Label: a.Label, Run: func(world w.World) (es.Transition[w.World], error) {
			player, err := query.GetPlayerEntity(world)
			if err != nil {
				return es.Transition[w.World]{}, err
			}
			if _, err := activity.ExecuteInteraction(player, a.Target, a.Interaction, world); err != nil {
				return es.Transition[w.World]{}, err
			}
			if len(nearbyPickupActions(world)) == 0 {
				return es.Transition[w.World]{Type: es.TransPop}, nil
			}
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}})
	}
	return "", choices
}

// nearbyPickupActions は足元と周囲8タイルの収納と、タイルごとにスタックで束ねた床のアイテムを行にする
func nearbyPickupActions(world w.World) []InteractionAction {
	player, err := query.GetPlayerEntity(world)
	if err != nil || !world.Components.GridElement.Has(player) {
		return nil
	}
	playerGrid := world.Components.GridElement.Get(player)

	var actions []InteractionAction
	for dy := consts.Tile(-1); dy <= 1; dy++ {
		for dx := consts.Tile(-1); dx <= 1; dx++ {
			c := consts.Coord[consts.Tile]{X: playerGrid.X + dx, Y: playerGrid.Y + dy}
			dirLabel := query.T(world, activity.GetDirectionLabel(playerGrid, &gc.GridElement{Coord: c}))
			if storage, ok := interactableAt(world, c, gc.InteractionStorage); ok {
				actions = append(actions, InteractionAction{
					Label:       query.T(world, "Open %s (%s)", query.GetEntityName(storage, world), dirLabel),
					Target:      storage,
					Interaction: gc.InteractionStorage,
				})
			}
			for _, stack := range query.GroupStacks(world, query.PickablesAt(world, c)) {
				actions = append(actions, InteractionAction{
					Label:       query.T(world, "Pick up %s (%s)", query.FormatNameCount(query.GetEntityName(stack.Rep, world), stack.Count), dirLabel),
					Target:      stack.Rep,
					Interaction: gc.InteractionItem,
				})
			}
		}
	}
	return actions
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
		case gc.InteractionDisassemble:
			if world.Components.Name.Has(interactableEntity) {
				result = append(result, InteractionAction{
					Label:       query.T(world, "Disassemble (%s)", query.GetEntityName(interactableEntity, world)),
					Target:      interactableEntity,
					Interaction: interaction,
				})
			}
		case gc.InteractionOpenCubeMenu:
			// SameTile 発動なので方向は常に直上。他の直上相互作用と同じく方向ラベルを付けない
			result = append(result, InteractionAction{
				Label:       query.T(world, "Open cube menu"),
				Target:      interactableEntity,
				Interaction: interaction,
			})
		case gc.InteractionItemAll, gc.InteractionMelee, gc.InteractionStorage:
			// アクションメニューに出さない種類。default を置かず exhaustive に全種別を
			// 明示させ、新しい InteractionKind の対応漏れを lint で検知する
		case gc.InteractionIgnite:
			// 着火はエンティティの Interactable でなく appendIgniteActions が隣接タイル走査で
			// タイル単位に組む。ここには来ないが exhaustive のため case を明示する
		case gc.InteractionFeedFuel:
			// 給油も Interactable でなく appendFeedFuelActions が隣接の火を走査して組む。
			// ここには来ないが exhaustive のため case を明示する
		}
	}

	return result
}
