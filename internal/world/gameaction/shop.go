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
	price := query.BuyPrice(world, player, item)

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
	price := query.SellPrice(world, player, item)

	if err := lifecycle.MoveToStorage(world, item, merchant); err != nil {
		return fmt.Errorf("failed to move item to merchant storage: %w", err)
	}
	// 代金の付与に失敗したら実体を手元へ戻し、品も金も失わないようにする。BuyStock の返金と対称
	if err := query.AddCurrency(world, player, price); err != nil {
		if rbErr := lifecycle.MoveToBackpack(world, item, player); rbErr != nil {
			return fmt.Errorf("payment failed and item rollback also failed: %w (rollback error: %w)", err, rbErr)
		}
		return fmt.Errorf("failed to add currency: %w", err)
	}

	return nil
}
