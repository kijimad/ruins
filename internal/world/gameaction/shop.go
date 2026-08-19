package gameaction

import (
	"fmt"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// BuyStock はプレイヤーが商人の在庫アイテムを買う。一覧の1行はスタック代表なので、
// 同一スタックを丸ごと買い、代金は個数×単価にする。実体を商人の収納からバックパックへ移す。
// 通貨が足りなければ何もせずエラーを返す
func BuyStock(world w.World, player ecs.Entity, item ecs.Entity) error {
	// BuyPrice は価値×スタック個数で全量の額を返す。移動もスタック丸ごとで、額と個数が揃う
	price := query.BuyPrice(world, player, item)

	if !query.HasCurrency(world, player, price) {
		return fmt.Errorf("not enough currency: need %d, have %d", price, query.GetCurrency(world, player))
	}
	if !query.ConsumeCurrency(world, player, price) {
		return fmt.Errorf("failed to consume currency")
	}

	// 実体をバックパックへ移す。移送に失敗したら通貨を返して整合を保つ
	if _, err := lifecycle.MoveStackToBackpack(world, item, player); err != nil {
		if refundErr := query.AddCurrency(world, player, price); refundErr != nil {
			return fmt.Errorf("move failed and refund also failed: %w (refund error: %w)", err, refundErr)
		}
		return fmt.Errorf("failed to move items to backpack: %w", err)
	}

	return nil
}

// SellStock はプレイヤーが持ち物を商人へ売る。一覧の1行はスタック代表なので、
// 同一スタックを丸ごと売り、代金は個数×単価で受け取る。実体は商人の収納へ移り店頭に並ぶ
func SellStock(world w.World, player ecs.Entity, merchant ecs.Entity, item ecs.Entity) error {
	// SellPrice は価値×スタック個数で全量の額を返す。額は移動前に確定する。移動後は
	// スタックの位置が変わり個数の数え上げ範囲がずれるため、先に出しておく
	price := query.SellPrice(world, player, item)

	if _, err := lifecycle.MoveStackToStorage(world, item, merchant); err != nil {
		return fmt.Errorf("failed to move items to merchant storage: %w", err)
	}
	// 代金の付与に失敗したら実体を手元へ戻し、品も金も失わないようにする。BuyStock の返金と対称
	if err := query.AddCurrency(world, player, price); err != nil {
		if _, rbErr := lifecycle.MoveStackToBackpack(world, item, player); rbErr != nil {
			return fmt.Errorf("payment failed and item rollback also failed: %w (rollback error: %w)", err, rbErr)
		}
		return fmt.Errorf("failed to add currency: %w", err)
	}

	return nil
}
