package gameaction

import (
	"fmt"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// BuyStock はプレイヤーが商人の在庫アイテムを買う。実体を商人の収納からプレイヤーのバックパックへ移す。
// 通貨が足りなければ何もせずエラーを返す
func BuyStock(world w.World, player ecs.Entity, item ecs.Entity) error {
	price := buyPrice(world, player, item)

	if !query.HasCurrency(world, player, price) {
		return fmt.Errorf("not enough currency: need %d, have %d", price, query.GetCurrency(world, player))
	}
	if !query.ConsumeCurrency(world, player, price) {
		return fmt.Errorf("failed to consume currency")
	}

	// 実体をバックパックへ移す。移送に失敗したら通貨を返して整合を保つ
	if err := lifecycle.MoveToBackpack(world, item, player); err != nil {
		if refundErr := query.AddCurrency(world, player, price); refundErr != nil {
			return fmt.Errorf("move failed and refund also failed: %w (refund error: %w)", err, refundErr)
		}
		return fmt.Errorf("failed to move item to backpack: %w", err)
	}

	return nil
}

// SellStock はプレイヤーが持ち物を商人へ売る。実体を商人の収納へ移し、代金を受け取る。
// 売った品は商人の在庫として店頭に並ぶ
func SellStock(world w.World, player ecs.Entity, merchant ecs.Entity, item ecs.Entity) error {
	price := sellPrice(world, player, item)
	if price == 0 {
		return fmt.Errorf("this item cannot be sold")
	}

	if err := lifecycle.MoveToStorage(world, item, merchant); err != nil {
		return fmt.Errorf("failed to move item to merchant storage: %w", err)
	}
	if err := query.AddCurrency(world, player, price); err != nil {
		return fmt.Errorf("failed to add currency: %w", err)
	}

	return nil
}

// HireRecruit はプレイヤーが商人の在庫の隊員候補を雇う。候補を隊員として活性化し、代金を支払う。
// 通貨が足りなければ何もせずエラーを返す
func HireRecruit(world w.World, player ecs.Entity, recruit ecs.Entity) error {
	price := buyPrice(world, player, recruit)

	if !query.HasCurrency(world, player, price) {
		return fmt.Errorf("not enough currency: need %d, have %d", price, query.GetCurrency(world, player))
	}
	if !query.ConsumeCurrency(world, player, price) {
		return fmt.Errorf("failed to consume currency")
	}

	// 隊員を活性化する。失敗したら通貨を返して整合を保つ
	if _, err := lifecycle.ActivateRecruit(world, player, recruit); err != nil {
		if refundErr := query.AddCurrency(world, player, price); refundErr != nil {
			return fmt.Errorf("hire failed and refund also failed: %w (refund error: %w)", err, refundErr)
		}
		return fmt.Errorf("failed to hire recruit: %w", err)
	}

	return nil
}

// buyPrice は交渉スキルの買値倍率込みの購入価格を返す
func buyPrice(world w.World, player ecs.Entity, entity ecs.Entity) int {
	price := query.CalculateBuyPrice(query.StockBaseValue(world, entity))
	if world.Components.CharModifiers.Has(player) {
		price = world.Components.CharModifiers.Get(player).BuyPrice.ApplyInt(price)
	}
	return price
}

// sellPrice は交渉スキルの売値倍率込みの売却価格を返す
func sellPrice(world w.World, player ecs.Entity, entity ecs.Entity) int {
	price := query.CalculateSellPrice(query.StockBaseValue(world, entity))
	if world.Components.CharModifiers.Has(player) {
		price = world.Components.CharModifiers.Get(player).SellPrice.ApplyInt(price)
	}
	return price
}
