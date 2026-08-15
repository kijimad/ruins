package gameaction

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
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

// woodenSwordValue は wooden_sword の基準価値。価格計算の期待値に使う
const woodenSwordValue = 80

func TestBuyStock(t *testing.T) {
	t.Parallel()

	t.Run("在庫アイテムの購入成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// SpawnPlayer は Wallet を備えるので、所持金は Get して上書きする
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
		require.NoError(t, err)
		world.Components.Wallet.Get(player).Currency = 1000

		merchant := world.ECS.NewEntity()
		item, err := lifecycle.SpawnStorageItem(world, "wooden_sword", 1, merchant)
		require.NoError(t, err)

		require.NoError(t, BuyStock(world, player, item))

		// 実体がプレイヤーのバックパックへ移り、商人の在庫から消えている
		require.True(t, world.Components.LocationInBackpack.Has(item))
		assert.Equal(t, player, world.Components.LocationInBackpack.Get(item).Owner)
		assert.False(t, world.Components.LocationInStorage.Has(item))

		currency := query.GetCurrency(world, player)
		assert.Equal(t, 1000-query.CalculateBuyPrice(woodenSwordValue), currency)
	})

	t.Run("通貨不足で購入失敗", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
		require.NoError(t, err)
		world.Components.Wallet.Get(player).Currency = 10

		merchant := world.ECS.NewEntity()
		item, err := lifecycle.SpawnStorageItem(world, "wooden_sword", 1, merchant)
		require.NoError(t, err)

		require.Error(t, BuyStock(world, player, item))
		// 在庫に残ったまま
		assert.True(t, world.Components.LocationInStorage.Has(item))
	})

	// query.Player のコールバック内で購入すると実体移送がクエリ反復中に走り
	// ワールドロック違反でパニックしていた回帰ケース
	t.Run("query.Player経由の購入でパニックしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// query.Player は Player と FactionAlly を要求する。SpawnPlayer が両方備える
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
		require.NoError(t, err)
		world.Components.Wallet.Get(player).Currency = 1000

		merchant := world.ECS.NewEntity()
		item, err := lifecycle.SpawnStorageItem(world, "wooden_sword", 1, merchant)
		require.NoError(t, err)

		var buyErr error
		require.NotPanics(t, func() {
			query.Player(world, func(p ecs.Entity) {
				buyErr = BuyStock(world, p, item)
			})
		})
		require.NoError(t, buyErr)
		assert.Equal(t, 1000-query.CalculateBuyPrice(woodenSwordValue), query.GetCurrency(world, player))
	})
}

func TestSellStock(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	world.Components.Wallet.Get(player).Currency = 0

	merchant := world.ECS.NewEntity()

	item, err := lifecycle.SpawnBackpackItem(world, "wooden_sword", 1)
	require.NoError(t, err)

	require.NoError(t, SellStock(world, player, merchant, item))

	// 代金を受け取り、実体が商人の在庫へ並ぶ
	assert.Equal(t, query.CalculateSellPrice(woodenSwordValue), query.GetCurrency(world, player))
	require.True(t, world.Components.LocationInStorage.Has(item))
	assert.Equal(t, merchant, world.Components.LocationInStorage.Get(item).Owner)
}
