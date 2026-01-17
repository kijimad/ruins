package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
)

func TestCalculateBuyPrice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		baseValue int
		want      int
	}{
		{"価値100のアイテム", 100, 200},
		{"価値50のアイテム", 50, 100},
		{"価値0のアイテム", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CalculateBuyPrice(tt.baseValue)
			if got != tt.want {
				t.Errorf("CalculateBuyPrice(%d) = %d, want %d", tt.baseValue, got, tt.want)
			}
		})
	}
}

func TestCalculateSellPrice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		baseValue int
		want      int
	}{
		{"価値100のアイテム", 100, 50},
		{"価値50のアイテム", 50, 25},
		{"価値0のアイテム", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CalculateSellPrice(tt.baseValue)
			if got != tt.want {
				t.Errorf("CalculateSellPrice(%d) = %d, want %d", tt.baseValue, got, tt.want)
			}
		})
	}
}

func TestBuyItem(t *testing.T) {
	t.Parallel()

	t.Run("通常アイテムの購入成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Wallet, &gc.Wallet{Currency: 1000})

		err := BuyItem(world, player, "木刀")
		if err != nil {
			t.Errorf("購入に失敗しました: %v", err)
		}

		// 所持金が減っていることを確認
		currency := GetCurrency(world, player)
		expectedCurrency := 1000 - CalculateBuyPrice(80) // 木刀の価値は80
		if currency != expectedCurrency {
			t.Errorf("通貨 = %d, want %d", currency, expectedCurrency)
		}
	})

	t.Run("通貨不足で購入失敗", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Wallet, &gc.Wallet{Currency: 10})

		err := BuyItem(world, player, "木刀")
		if err == nil {
			t.Error("通貨不足なのに購入できてしまった")
		}
	})
}

func TestSellItem(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Wallet, &gc.Wallet{Currency: 0})

	// アイテムを生成
	item, _ := SpawnItem(world, "木刀", 1, gc.ItemLocationInBackpack)

	t.Run("アイテムの売却成功", func(t *testing.T) {
		t.Parallel()
		err := SellItem(world, player, item)
		if err != nil {
			t.Errorf("売却に失敗しました: %v", err)
		}

		// 所持金が増えていることを確認
		currency := GetCurrency(world, player)
		expectedCurrency := CalculateSellPrice(80) // 木刀の価値は80
		if currency != expectedCurrency {
			t.Errorf("通貨 = %d, want %d", currency, expectedCurrency)
		}
	})
}

func TestGetShopInventory(t *testing.T) {
	t.Parallel()
	inventory := GetShopInventory()

	if len(inventory) == 0 {
		t.Error("品揃えが空です")
	}

	// 最低限のアイテムが含まれているかチェック
	hasItem := false
	for _, item := range inventory {
		if item == "木刀" {
			hasItem = true
			break
		}
	}
	if !hasItem {
		t.Error("品揃えに木刀が含まれていません")
	}
}
