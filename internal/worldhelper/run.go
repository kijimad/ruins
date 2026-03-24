package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// SoldItem は売却対象アイテム1件の情報
type SoldItem struct {
	Entity ecs.Entity // 同名グループの代表エンティティ。スペック表示に使う
	Name   string
	Count  int
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

// collectBackpackItems はバックパック内の全アイテムを収集し、同名アイテムを集約して返す。
// エンティティは削除しない。
func collectBackpackItems(world w.World, playerEntity ecs.Entity) AutoSellResult {
	sellPriceMod := 100
	if playerEntity.HasComponent(world.Components.CharModifiers) {
		mods := world.Components.CharModifiers.Get(playerEntity).(*gc.CharModifiers)
		sellPriceMod = mods.SellPrice
	}

	type itemInfo struct {
		entity ecs.Entity
		name   string
		count  int
		price  int
	}
	var items []itemInfo
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := ""
		if entity.HasComponent(world.Components.Name) {
			name = world.Components.Name.Get(entity).(*gc.Name).Name
		}
		count := world.Components.Item.Get(entity).(*gc.Item).Count

		price := 0
		if entity.HasComponent(world.Components.Value) {
			baseValue := world.Components.Value.Get(entity).(*gc.Value).Value
			price = CalculateSellPrice(baseValue) * count * sellPriceMod / 100
		}

		items = append(items, itemInfo{entity: entity, name: name, count: count, price: price})
	}))

	// 同名アイテムを集約する
	merged := map[string]*SoldItem{}
	var order []string
	total := 0
	for _, it := range items {
		if existing, ok := merged[it.name]; ok {
			existing.Count += it.count
			existing.Price += it.price
		} else {
			merged[it.name] = &SoldItem{Entity: it.entity, Name: it.name, Count: it.count, Price: it.price}
			order = append(order, it.name)
		}
		total += it.price
	}

	soldItems := make([]SoldItem, 0, len(order))
	for _, name := range order {
		soldItems = append(soldItems, *merged[name])
	}

	return AutoSellResult{Items: soldItems, Total: total}
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

	ApplyProfession(world, playerEntity, prof)
	return nil
}
