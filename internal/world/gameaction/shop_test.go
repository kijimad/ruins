package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
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

		player := world.ECS.NewEntity()
		world.Components.Wallet.Add(player, &gc.Wallet{Currency: 1000})

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

		player := world.ECS.NewEntity()
		world.Components.Wallet.Add(player, &gc.Wallet{Currency: 10})

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

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.FactionAlly.Add(player, &gc.FactionAlly{})
		world.Components.Wallet.Add(player, &gc.Wallet{Currency: 1000})

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

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 0})

	merchant := world.ECS.NewEntity()

	item, err := lifecycle.SpawnBackpackItem(world, "wooden_sword", 1)
	require.NoError(t, err)

	require.NoError(t, SellStock(world, player, merchant, item))

	// 代金を受け取り、実体が商人の在庫へ並ぶ
	assert.Equal(t, query.CalculateSellPrice(woodenSwordValue), query.GetCurrency(world, player))
	require.True(t, world.Components.LocationInStorage.Has(item))
	assert.Equal(t, merchant, world.Components.LocationInStorage.Get(item).Owner)
}

func TestHireRecruit(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 100000})
	world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})

	merchant := world.ECS.NewEntity()
	abilities := gc.Abilities{
		Vitality:  gc.Ability{Base: 5},
		Strength:  gc.Ability{Base: 5},
		Sensation: gc.Ability{Base: 5},
		Dexterity: gc.Ability{Base: 5},
		Agility:   gc.Ability{Base: 5},
		Defense:   gc.Ability{Base: 5},
	}
	recruit, err := lifecycle.SpawnStorageRecruit(world, merchant, "Test", abilities, "general")
	require.NoError(t, err)
	// 生成時に確定した基準価値。買値の期待値に使う
	recruitValue := world.Components.Value.Get(recruit).Value

	before := query.SquadMembers(world)
	require.NoError(t, HireRecruit(world, player, recruit))

	// 候補実体は消え、隊員が1人増えている
	assert.False(t, world.ECS.Alive(recruit))
	after := query.SquadMembers(world)
	assert.Len(t, after, len(before)+1)

	// 代金は生成時に確定した基準価値の買値
	expected := 100000 - query.CalculateBuyPrice(recruitValue)
	assert.Equal(t, expected, query.GetCurrency(world, player))
}
