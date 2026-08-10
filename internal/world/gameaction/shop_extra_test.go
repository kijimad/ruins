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

// TestBuyStock_スタッカブルはバックパックのスタックに統合される は商人在庫のスタッカブルを買うと
// バックパック内の同名アイテムと同一エンティティに統合されることを確認する。
func TestBuyStock_スタッカブルはバックパックのスタックに統合される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 1000})

	// 先にプレイヤーが1個持っている
	_, err := lifecycle.SpawnBackpackItem(world, "wooden_stick", 1)
	require.NoError(t, err)

	merchant := world.ECS.NewEntity()
	item, err := lifecycle.SpawnStorageItem(world, "wooden_stick", 1, merchant)
	require.NoError(t, err)

	require.NoError(t, BuyStock(world, player, item))

	stackQuery := ecs.NewFilter2[gc.Name, gc.Stackable](world.ECS).Query()
	found := false
	for stackQuery.Next() {
		name := world.Components.Name.Get(stackQuery.Entity())
		if name.Name != "Wooden Stick" {
			continue
		}
		found = true
		stackable := world.Components.Stackable.Get(stackQuery.Entity())
		assert.Equal(t, 2, stackable.Count, "スタッカブルアイテムは同一エンティティのCountが増える")
	}
	assert.True(t, found, "スタッカブルアイテムのエンティティが存在する")
}

// TestBuyStock_交渉スキルで買値が変わる はCharModifiers.BuyPriceが購入価格に反映されることを確認する。
func TestBuyStock_交渉スキルで買値が変わる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 1000})
	world.Components.CharModifiers.Add(player, &gc.CharModifiers{BuyPrice: 50})

	merchant := world.ECS.NewEntity()
	item, err := lifecycle.SpawnStorageItem(world, "wooden_sword", 1, merchant)
	require.NoError(t, err)

	require.NoError(t, BuyStock(world, player, item))

	currency := query.GetCurrency(world, player)
	normalPrice := query.CalculateBuyPrice(woodenSwordValue)
	discountedPrice := normalPrice / 2
	assert.Equal(t, 1000-discountedPrice, currency, "買値倍率50%で半額になる")
}

// TestSellStock_価値0のアイテムは対価0で売れる は無価値な品でも売却は成功し、対価が0になることを確認する。
func TestSellStock_価値0のアイテムは対価0で売れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 0})
	merchant := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	world.Components.Value.Add(item, &gc.Value{Value: 0})
	world.Components.Name.Add(item, &gc.Name{Name: "Scrap"})
	world.Components.RawID.Add(item, &gc.RawID{ID: "scrap"})
	world.Components.Stackable.Add(item, &gc.Stackable{Count: 1})

	require.NoError(t, SellStock(world, player, merchant, item), "価値0でも売却は成功する")

	currency := query.GetCurrency(world, player)
	assert.Equal(t, 0, currency, "無価値な品の対価は0で通貨は増えない")

	// 売った品は商人の在庫へ並ぶ
	require.True(t, world.Components.LocationInStorage.Has(item), "実体は商人の収納へ移る")
	assert.Equal(t, merchant, world.Components.LocationInStorage.Get(item).Owner)
}

// TestSellStock_交渉スキルで売値が変わる はCharModifiers.SellPriceが売却価格に反映されることを確認する。
func TestSellStock_交渉スキルで売値が変わる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 0})
	world.Components.CharModifiers.Add(player, &gc.CharModifiers{SellPrice: 200})

	merchant := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	world.Components.Value.Add(item, &gc.Value{Value: 100})
	world.Components.Name.Add(item, &gc.Name{Name: "Test Item"})
	// 実アイテムは必ず RawID を持つ。収納内スタック統合はこれで同名を引く
	world.Components.RawID.Add(item, &gc.RawID{ID: "test_item"})
	world.Components.Stackable.Add(item, &gc.Stackable{Count: 1})

	require.NoError(t, SellStock(world, player, merchant, item))

	currency := query.GetCurrency(world, player)
	normalPrice := query.CalculateSellPrice(100)
	assert.Equal(t, normalPrice*2, currency, "売値倍率200%で倍額になる")

	// 売った品は商人の在庫へ並ぶ
	require.True(t, world.Components.LocationInStorage.Has(item))
	assert.Equal(t, merchant, world.Components.LocationInStorage.Get(item).Owner)
}
