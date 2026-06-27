package action

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// BuyItem はプレイヤーがアイテムを購入する
// 通貨が足りない場合や購入に失敗した場合はエラーを返す
func BuyItem(world w.World, playerEntity ecs.Entity, itemName string) error {
	itemDef, err := raw.FindItem(world.Resources.RawMaster, itemName)
	if err != nil {
		return fmt.Errorf("アイテムが見つかりません: %s", itemName)
	}

	baseValue := itemDef.Value
	price := query.CalculateBuyPrice(int(baseValue))

	// 交渉スキルによる買値倍率を適用する
	if playerEntity.HasComponent(world.Components.CharModifiers) {
		mods := world.Components.CharModifiers.Get(playerEntity).(*gc.CharModifiers)
		price = price * mods.BuyPrice / 100
	}

	if !query.HasCurrency(world, playerEntity, price) {
		return fmt.Errorf("地髄が足りません（必要: %d、所持: %d）", price, query.GetCurrency(world, playerEntity))
	}

	if !query.ConsumeCurrency(world, playerEntity, price) {
		return fmt.Errorf("通貨の消費に失敗しました")
	}

	isStackable := itemDef.Stackable != nil && *itemDef.Stackable

	if isStackable {
		err := lifecycle.ChangeStackableCount(world, itemName, 1)
		if err != nil {
			if refundErr := query.AddCurrency(world, playerEntity, price); refundErr != nil {
				return fmt.Errorf("アイテムの生成に失敗し、返金も失敗しました: %w (返金エラー: %v)", err, refundErr)
			}
			return fmt.Errorf("アイテムの生成に失敗しました: %w", err)
		}
	} else {
		_, err := lifecycle.SpawnBackpackItem(world, itemName, 1)
		if err != nil {
			if refundErr := query.AddCurrency(world, playerEntity, price); refundErr != nil {
				return fmt.Errorf("アイテムの生成に失敗し、返金も失敗しました: %w (返金エラー: %v)", err, refundErr)
			}
			return fmt.Errorf("アイテムの生成に失敗しました: %w", err)
		}
	}

	return nil
}

// SellItem はプレイヤーがアイテムを売却する
func SellItem(world w.World, playerEntity ecs.Entity, itemEntity ecs.Entity) error {
	baseValue := query.GetItemValue(world, itemEntity)
	if baseValue == 0 {
		return fmt.Errorf("このアイテムは売却できません")
	}
	price := query.CalculateSellPrice(baseValue)

	// 交渉スキルによる売値倍率を適用する
	if playerEntity.HasComponent(world.Components.CharModifiers) {
		mods := world.Components.CharModifiers.Get(playerEntity).(*gc.CharModifiers)
		price = price * mods.SellPrice / 100
	}

	if err := lifecycle.ChangeItemCount(world, itemEntity, -1); err != nil {
		return fmt.Errorf("アイテムの売却に失敗した: %w", err)
	}

	if err := query.AddCurrency(world, playerEntity, price); err != nil {
		return fmt.Errorf("通貨の追加に失敗しました: %w", err)
	}

	return nil
}

// GetShopInventory は店の品揃えを返す（ハードコーディング）
func GetShopInventory() []string {
	return []string{
		"木刀",
		"ハンドガン",
		"西洋鎧",
		"作業用ヘルメット",
		"革のブーツ",
		"回復薬",
		"陸軍射撃マニュアル",
	}
}
