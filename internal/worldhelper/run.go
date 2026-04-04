package worldhelper

import (
	"fmt"
	"sort"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// SoldItem は売却対象アイテム1件の情報
type SoldItem struct {
	Entity ecs.Entity
	Name   string
	Price  int
}

// AutoSellResult は自動売却の結果
type AutoSellResult struct {
	Items []SoldItem
	Total int
}

// PreviewEndRun はラン終了時の精算プレビューを生成する。
// 全装備をバックパックに移動し、売却候補と合計金額を返す。
// エンティティは削除されず、スペック表示に使える状態で残る。
func PreviewEndRun(world w.World, playerEntity ecs.Entity) (AutoSellResult, error) {
	unequipAll(world, playerEntity)
	result := collectBackpackItems(world, playerEntity)
	return result, nil
}

// ExecuteEndRun は精算を実行する。
// 通貨を加算し、バックパック内アイテムを全て削除し、職業を再適用する。
func ExecuteEndRun(world w.World, playerEntity ecs.Entity, total int) error {
	// 通貨を加算する
	if total > 0 {
		wallet := world.Components.Wallet.Get(playerEntity).(*gc.Wallet)
		wallet.Currency += total
	}

	// バックパック内アイテムを全て削除する
	var toDelete []ecs.Entity
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		toDelete = append(toDelete, entity)
	}))
	for _, e := range toDelete {
		world.Manager.DeleteEntity(e)
	}

	// 職業を再適用する
	if err := reapplyProfession(world, playerEntity); err != nil {
		return fmt.Errorf("職業の再適用に失敗: %w", err)
	}
	return nil
}

// unequipAll はプレイヤーの装備中アイテムを全てバックパックに移動する
func unequipAll(world w.World, playerEntity ecs.Entity) {
	var equipped []ecs.Entity
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationEquipped,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		loc := world.Components.ItemLocationEquipped.Get(entity).(*gc.LocationEquipped)
		if loc.Owner == playerEntity {
			equipped = append(equipped, entity)
		}
	}))

	for _, item := range equipped {
		MoveToBackpack(world, item, playerEntity)
	}
}

// collectBackpackItems はバックパック内の全アイテムを収集して返す。
// エンティティは削除しない。
func collectBackpackItems(world w.World, playerEntity ecs.Entity) AutoSellResult {
	sellPriceMod := 100
	if playerEntity.HasComponent(world.Components.CharModifiers) {
		mods := world.Components.CharModifiers.Get(playerEntity).(*gc.CharModifiers)
		sellPriceMod = mods.SellPrice
	}

	var items []SoldItem
	total := 0
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := ""
		if entity.HasComponent(world.Components.Name) {
			name = world.Components.Name.Get(entity).(*gc.Name).Name
		}

		price := 0
		if entity.HasComponent(world.Components.Value) {
			count := world.Components.Item.Get(entity).(*gc.Item).Count
			baseValue := world.Components.Value.Get(entity).(*gc.Value).Value
			price = CalculateSellPrice(baseValue) * count * sellPriceMod / 100
		}

		items = append(items, SoldItem{Entity: entity, Name: name, Price: price})
		total += price
	}))

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return AutoSellResult{Items: items, Total: total}
}

// reapplyProfession はプレイヤーの職業を再適用する
func reapplyProfession(world w.World, playerEntity ecs.Entity) error {
	if !playerEntity.HasComponent(world.Components.Profession) {
		return fmt.Errorf("プレイヤーにProfessionコンポーネントがない")
	}
	profComp := world.Components.Profession.Get(playerEntity).(*gc.Profession)

	prof, err := world.Resources.RawMaster.GetProfession(profComp.ID)
	if err != nil {
		return fmt.Errorf("職業データの取得に失敗: %w", err)
	}

	return ApplyProfession(world, playerEntity, prof)
}
