package gameaction

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// BuyItem はプレイヤーがアイテムを購入する
// 通貨が足りない場合や購入に失敗した場合はエラーを返す
func BuyItem(world w.World, playerEntity ecs.Entity, itemName string) error {
	itemDef, err := raw.FindItem(world.Resources.RawMaster, itemName)
	if err != nil {
		return fmt.Errorf("item not found: %s", itemName)
	}

	baseValue := itemDef.Value
	price := query.CalculateBuyPrice(int(baseValue))

	// 交渉スキルによる買値倍率を適用する
	if world.Components.CharModifiers.Has(playerEntity) {
		mods := world.Components.CharModifiers.Get(playerEntity)
		price = mods.BuyPrice.ApplyInt(price)
	}

	if !query.HasCurrency(world, playerEntity, price) {
		return fmt.Errorf("not enough currency: need %d, have %d", price, query.GetCurrency(world, playerEntity))
	}

	if !query.ConsumeCurrency(world, playerEntity, price) {
		return fmt.Errorf("failed to consume currency")
	}

	isStackable := itemDef.Stackable != nil && *itemDef.Stackable

	if isStackable {
		err := lifecycle.ChangeStackableCount(world, itemName, 1)
		if err != nil {
			if refundErr := query.AddCurrency(world, playerEntity, price); refundErr != nil {
				return fmt.Errorf("item generation failed and refund also failed: %w (refund error: %w)", err, refundErr)
			}
			return fmt.Errorf("failed to generate item: %w", err)
		}
	} else {
		_, err := lifecycle.SpawnBackpackItem(world, itemName, 1)
		if err != nil {
			if refundErr := query.AddCurrency(world, playerEntity, price); refundErr != nil {
				return fmt.Errorf("item generation failed and refund also failed: %w (refund error: %w)", err, refundErr)
			}
			return fmt.Errorf("failed to generate item: %w", err)
		}
	}

	return nil
}

// SellItem はプレイヤーがアイテムを売却する
func SellItem(world w.World, playerEntity ecs.Entity, itemEntity ecs.Entity) error {
	baseValue := query.GetItemValue(world, itemEntity)
	if baseValue == 0 {
		return fmt.Errorf("this item cannot be sold")
	}
	price := query.CalculateSellPrice(baseValue)

	// 交渉スキルによる売値倍率を適用する
	if world.Components.CharModifiers.Has(playerEntity) {
		mods := world.Components.CharModifiers.Get(playerEntity)
		price = mods.SellPrice.ApplyInt(price)
	}

	if err := lifecycle.ChangeItemCount(world, itemEntity, -1); err != nil {
		return fmt.Errorf("failed to sell item: %w", err)
	}

	if err := query.AddCurrency(world, playerEntity, price); err != nil {
		return fmt.Errorf("failed to add currency: %w", err)
	}

	return nil
}

// GetShopInventory は店の品揃えを返す（ハードコーディング）
func GetShopInventory() []string {
	return []string{
		"wooden_sword",
		"handgun",
		"western_armor",
		"work_helmet",
		"leather_boots",
		"healing_potion",
		"army_shooting_manual",
	}
}
