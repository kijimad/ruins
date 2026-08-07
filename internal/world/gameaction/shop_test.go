package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			got := query.CalculateBuyPrice(tt.baseValue)
			assert.Equal(t, tt.want, got)
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
			got := query.CalculateSellPrice(tt.baseValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuyItem(t *testing.T) {
	t.Parallel()

	t.Run("通常アイテムの購入成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Wallet.Add(player, &gc.Wallet{Currency: 1000})

		err := BuyItem(world, player, "木刀")
		require.NoError(t, err)

		currency := query.GetCurrency(world, player)
		expectedCurrency := 1000 - query.CalculateBuyPrice(80)
		assert.Equal(t, expectedCurrency, currency)
	})

	t.Run("通貨不足で購入失敗", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Wallet.Add(player, &gc.Wallet{Currency: 10})

		err := BuyItem(world, player, "木刀")
		assert.Error(t, err)
	})

	// query.Player のコールバック内で購入するとエンティティ生成が
	// クエリ反復中に走りワールドロック違反でパニックしていた回帰ケース
	t.Run("query.Player経由の購入でパニックしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.FactionAlly.Add(player, &gc.FactionAlly{})
		world.Components.Wallet.Add(player, &gc.Wallet{Currency: 1000})

		var buyErr error
		require.NotPanics(t, func() {
			query.Player(world, func(p ecs.Entity) {
				buyErr = BuyItem(world, p, "木刀")
			})
		})
		require.NoError(t, buyErr)

		currency := query.GetCurrency(world, player)
		assert.Equal(t, 1000-query.CalculateBuyPrice(80), currency, "購入後は通貨が減っているべき")
	})
}

func TestSellItem(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 0})

	item, _ := lifecycle.SpawnBackpackItem(world, "木刀", 1)

	t.Run("アイテムの売却成功", func(t *testing.T) {
		t.Parallel()
		err := SellItem(world, player, item)
		require.NoError(t, err)

		currency := query.GetCurrency(world, player)
		expectedCurrency := query.CalculateSellPrice(80)
		assert.Equal(t, expectedCurrency, currency)
	})
}

func TestGetShopInventory(t *testing.T) {
	t.Parallel()
	inventory := GetShopInventory()

	assert.NotEmpty(t, inventory)
	assert.Contains(t, inventory, "木刀")
}
