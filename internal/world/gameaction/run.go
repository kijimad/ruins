package gameaction

import (
	"fmt"
	"sort"

	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
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
	if err := lifecycle.UnequipAll(world, playerEntity); err != nil {
		return AutoSellResult{}, fmt.Errorf("装備解除に失敗: %w", err)
	}
	result := collectBackpackItems(world, playerEntity)
	return result, nil
}

// ExecuteEndRun は精算を実行する。
// 通貨を加算し、バックパック内アイテムを全て削除し、職業を再適用する。
func ExecuteEndRun(world w.World, playerEntity ecs.Entity, total int) error {
	if total > 0 {
		wallet := world.Components.Wallet.MustGet(playerEntity)
		wallet.Currency += total
	}

	var toDelete []ecs.Entity
	world.Manager.Join(
		world.Components.LocationInBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		toDelete = append(toDelete, entity)
	}))
	for _, e := range toDelete {
		world.Manager.DeleteEntity(e)
	}

	if err := reapplyProfession(world, playerEntity); err != nil {
		return fmt.Errorf("職業の再適用に失敗: %w", err)
	}
	return nil
}

// collectBackpackItems はバックパック内の全アイテムを収集して返す。
// エンティティは削除しない。
func collectBackpackItems(world w.World, playerEntity ecs.Entity) AutoSellResult {
	sellPriceMod := 100
	if playerEntity.HasComponent(world.Components.CharModifiers) {
		mods := world.Components.CharModifiers.MustGet(playerEntity)
		sellPriceMod = mods.SellPrice
	}

	var items []SoldItem
	total := 0
	world.Manager.Join(
		world.Components.LocationInBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := ""
		if entity.HasComponent(world.Components.Name) {
			name = world.Components.Name.MustGet(entity).Name
		}

		price := 0
		if entity.HasComponent(world.Components.Value) {
			count := query.GetEntityCount(world, entity)
			baseValue := world.Components.Value.MustGet(entity).Value
			price = query.CalculateSellPrice(baseValue) * count * sellPriceMod / 100
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
	profComp := world.Components.Profession.MustGet(playerEntity)

	prof, err := raw.GetProfession(world.Resources.RawMaster, profComp.ID)
	if err != nil {
		return fmt.Errorf("職業データの取得に失敗: %w", err)
	}

	return ApplyProfession(world, playerEntity, prof)
}
