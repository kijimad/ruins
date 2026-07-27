package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuyItem_スタッカブルアイテムはスタックに加算される はスタッカブルなアイテムを
// 2回購入すると同一エンティティのCountが積み上がることを確認する。
func TestBuyItem_スタッカブルアイテムはスタックに加算される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 1000})

	require.NoError(t, BuyItem(world, player, "木の棒"))
	require.NoError(t, BuyItem(world, player, "木の棒"))

	stackQuery := ecs.NewFilter2[gc.Name, gc.Stackable](world.ECS).Query()
	found := false
	for stackQuery.Next() {
		name := world.Components.Name.Get(stackQuery.Entity())
		if name.Name != "木の棒" {
			continue
		}
		found = true
		stackable := world.Components.Stackable.Get(stackQuery.Entity())
		assert.Equal(t, 2, stackable.Count, "スタッカブルアイテムは同一エンティティのCountが増える")
	}
	assert.True(t, found, "スタッカブルアイテムのエンティティが存在する")
}

// TestBuyItem_交渉スキルで買値が変わる はCharModifiers.BuyPriceが購入価格に反映されることを確認する。
func TestBuyItem_交渉スキルで買値が変わる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 1000})
	world.Components.CharModifiers.Add(player, &gc.CharModifiers{BuyPrice: 50})

	itemDef, err := raw.FindItem(world.Resources.RawMaster, "木刀")
	require.NoError(t, err)

	require.NoError(t, BuyItem(world, player, "木刀"))

	currency := query.GetCurrency(world, player)
	normalPrice := query.CalculateBuyPrice(int(itemDef.Value))
	discountedPrice := normalPrice / 2
	assert.Equal(t, 1000-discountedPrice, currency, "買値倍率50%で半額になる")
}

// TestSellItem_価値0のアイテムは売却できない はValueコンポーネントを持たないアイテムの売却がエラーになることを確認する。
func TestSellItem_価値0のアイテムは売却できない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 0})
	item := world.ECS.NewEntity()

	err := SellItem(world, player, item)
	require.ErrorContains(t, err, "売却できません")

	currency := query.GetCurrency(world, player)
	assert.Equal(t, 0, currency, "売却失敗時は通貨が変動しない")
}

// TestSellItem_交渉スキルで売値が変わる はCharModifiers.SellPriceが売却価格に反映されることを確認する。
func TestSellItem_交渉スキルで売値が変わる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 0})
	world.Components.CharModifiers.Add(player, &gc.CharModifiers{SellPrice: 200})

	item := world.ECS.NewEntity()
	world.Components.Value.Add(item, &gc.Value{Value: 100})
	world.Components.Name.Add(item, &gc.Name{Name: "テストアイテム"})
	world.Components.Stackable.Add(item, &gc.Stackable{Count: 1})

	require.NoError(t, SellItem(world, player, item))

	currency := query.GetCurrency(world, player)
	normalPrice := query.CalculateSellPrice(100)
	assert.Equal(t, normalPrice*2, currency, "売値倍率200%で倍額になる")
}
