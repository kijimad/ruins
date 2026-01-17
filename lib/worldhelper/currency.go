package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/lib/components"
	w "github.com/kijimaD/ruins/lib/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// AddCurrency はエンティティに所持金を追加する
// Walletコンポーネントがない場合はエラーを返す
func AddCurrency(world w.World, entity ecs.Entity, amount int) error {
	wallet := world.Components.Wallet.Get(entity)
	if wallet == nil {
		return fmt.Errorf("エンティティにWalletコンポーネントがありません")
	}
	w := wallet.(*gc.Wallet)
	w.Currency += amount
	return nil
}

// GetCurrency はエンティティの所持金を取得する
func GetCurrency(world w.World, entity ecs.Entity) int {
	wallet := world.Components.Wallet.Get(entity)
	if wallet == nil {
		return 0
	}
	return wallet.(*gc.Wallet).Currency
}

// HasCurrency は指定額以上の所持金を持っているか確認
func HasCurrency(world w.World, entity ecs.Entity, amount int) bool {
	return GetCurrency(world, entity) >= amount
}

// ConsumeCurrency はエンティティの所持金を消費する
// 所持金が足りない場合はfalseを返す
// TODO(kijima): 使いにくいのを直す
func ConsumeCurrency(world w.World, entity ecs.Entity, amount int) bool {
	if !HasCurrency(world, entity, amount) {
		return false
	}
	wallet := world.Components.Wallet.Get(entity)
	if wallet == nil {
		return false
	}
	w := wallet.(*gc.Wallet)
	w.Currency -= amount
	return true
}

// FormatCurrency は金額を CZ 形式でフォーマットする
// 3桁ごとにカンマで区切る（例: CZ 100,204）
func FormatCurrency(amount int) string {
	// 数値を文字列に変換
	str := fmt.Sprintf("%d", amount)

	// 負の数の処理
	negative := false
	if amount < 0 {
		negative = true
		str = str[1:] // マイナス記号を除去
	}

	// 3桁ごとにカンマを挿入
	var result string
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}

	if negative {
		result = "-" + result
	}

	return fmt.Sprintf("CZ %s", result)
}
