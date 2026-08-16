package query

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// AddCurrency はエンティティに所持金を追加する
func AddCurrency(world w.World, entity ecs.Entity, amount consts.Currency) error {
	wallet := world.Components.Wallet.Get(entity)
	if wallet == nil {
		return fmt.Errorf("entity has no Wallet component")
	}
	wallet.Currency += amount
	return nil
}

// GetCurrency はエンティティの所持金を取得する
func GetCurrency(world w.World, entity ecs.Entity) consts.Currency {
	wallet := world.Components.Wallet.Get(entity)
	if wallet == nil {
		return 0
	}
	return wallet.Currency
}

// HasCurrency は指定額以上の所持金を持っているか確認
func HasCurrency(world w.World, entity ecs.Entity, amount consts.Currency) bool {
	return GetCurrency(world, entity) >= amount
}

// ConsumeCurrency はエンティティの所持金を消費する
// 所持金が足りない場合はfalseを返す
// TODO(kijima): 使いにくいのを直す
func ConsumeCurrency(world w.World, entity ecs.Entity, amount consts.Currency) bool {
	if !HasCurrency(world, entity, amount) {
		return false
	}
	wallet := world.Components.Wallet.Get(entity)
	if wallet == nil {
		return false
	}
	wallet.Currency -= amount
	return true
}
