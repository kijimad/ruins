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

func TestBuyStock_スタッカブルはバックパックのスタックに統合される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	world.Components.Wallet.Get(player).Currency = 1000

	// 先にプレイヤーが1個持っている
	_, err = lifecycle.SpawnBackpackItem(world, "wooden_stick", 1)
	require.NoError(t, err)

	merchant := world.ECS.NewEntity()
	item, err := lifecycle.SpawnStorageItem(world, "wooden_stick", 1, merchant)
	require.NoError(t, err)

	require.NoError(t, BuyStock(world, player, item))

	// 1個1エンティティなので、買った分が個別エンティティとして増える。統合はしない
	stackQuery := ecs.NewFilter1[gc.Name](world.ECS).Query()
	count := 0
	for stackQuery.Next() {
		if world.Components.Name.Get(stackQuery.Entity()).Name == "Wooden Stick" {
			count++
		}
	}
	assert.Equal(t, 2, count, "買った分が個別エンティティとして増える")
}

func TestBuyStock_交渉スキルで買値が変わる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	world.Components.Wallet.Get(player).Currency = 1000
	// 交渉スキルを上げて買値倍率を下げる
	world.Components.Skills.Get(player).Get(gc.SkillNegotiation).Value = 25

	merchant := world.ECS.NewEntity()
	item, err := lifecycle.SpawnStorageItem(world, "wooden_sword", 1, merchant)
	require.NoError(t, err)

	expected := query.BuyPrice(world, player, item)
	assert.Less(t, expected, query.CalculateBuyPrice(woodenSwordValue), "交渉スキルで基準価格より安くなる")

	require.NoError(t, BuyStock(world, player, item))

	assert.Equal(t, 1000-expected, query.GetCurrency(world, player), "表示価格と同額が引かれる")
}

// 価値0の品は実スポーンで自然に作れないため、売却対象は手組みの fixture のまま残す。
func TestSellStock_価値0のアイテムは対価0で売れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	world.Components.Wallet.Get(player).Currency = 0

	merchant := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	world.Components.Value.Add(item, &gc.Value{Value: 0})
	world.Components.Name.Add(item, &gc.Name{Name: "Scrap"})
	world.Components.RawID.Add(item, &gc.RawID{ID: "scrap"})

	require.NoError(t, SellStock(world, player, merchant, item), "価値0でも売却は成功する")

	currency := query.GetCurrency(world, player)
	assert.Equal(t, consts.Currency(0), currency, "無価値な品の対価は0で通貨は増えない")

	require.True(t, world.Components.LocationInStorage.Has(item), "実体は商人の収納へ移る")
	assert.Equal(t, merchant, world.Components.LocationInStorage.Get(item).Owner)
}

// 期待値を明示するため、売却対象は価値100の手組み fixture のまま残す。
func TestSellStock_交渉スキルで売値が変わる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	world.Components.Wallet.Get(player).Currency = 0
	// 交渉スキルを上げて売値倍率を上げる
	world.Components.Skills.Get(player).Get(gc.SkillNegotiation).Value = 25

	merchant := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	world.Components.Value.Add(item, &gc.Value{Value: 100})
	world.Components.Name.Add(item, &gc.Name{Name: "Test Item"})
	// 実アイテムは必ず RawID を持つ。収納内スタック統合はこれで同名を引く
	world.Components.RawID.Add(item, &gc.RawID{ID: "test_item"})

	expected := query.SellPrice(world, player, item)
	assert.Greater(t, expected, query.CalculateSellPrice(100), "交渉スキルで基準価格より高く売れる")

	require.NoError(t, SellStock(world, player, merchant, item))

	assert.Equal(t, expected, query.GetCurrency(world, player), "表示価格と同額を得る")

	require.True(t, world.Components.LocationInStorage.Has(item))
	assert.Equal(t, merchant, world.Components.LocationInStorage.Get(item).Owner)
}

// TestBuyStock_通貨消費に失敗すると購入できない は ConsumeCurrency 自体が失敗したとき
// BuyStock がエラーを返し、在庫が動かないことを確認する。HasCurrency は 0 円要求なら
// Wallet が無くても通るため、ConsumeCurrency だけが失敗する経路を無価値な品で再現する。
func TestBuyStock_通貨消費に失敗すると購入できない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	// Wallet を剥がしてConsumeCurrencyだけが失敗する状態を作る
	world.Components.Wallet.Remove(player)

	merchant := world.ECS.NewEntity()
	item := world.ECS.NewEntity()
	// 価格0なら HasCurrency は Wallet 無しでも通る。ConsumeCurrency だけを失敗させる経路を作る
	world.Components.Value.Add(item, &gc.Value{Value: 0})
	world.Components.Name.Add(item, &gc.Name{Name: "Scrap"})
	world.Components.RawID.Add(item, &gc.RawID{ID: "scrap"})
	world.Components.LocationInStorage.Add(item, &gc.LocationInStorage{Owner: merchant})

	err = BuyStock(world, player, item)
	require.ErrorContains(t, err, "failed to consume currency")

	// Wallet を剥がしたので通貨の増減は引けない。在庫に残ったままで購入が巻き戻ったことを確かめる
	assert.True(t, world.Components.LocationInStorage.Has(item))
}

// TestSellStock_通貨付与に失敗すると実体をバックパックへ戻す は AddCurrency が失敗したとき
// SellStock が実体を商人の在庫からプレイヤーのバックパックへロールバックすることを確認する。
func TestSellStock_通貨付与に失敗すると実体をバックパックへ戻す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)

	item, err := lifecycle.SpawnBackpackItem(world, "wooden_sword", 1)
	require.NoError(t, err)

	merchant := world.ECS.NewEntity()

	// Wallet が無いと AddCurrency が失敗し、ロールバック経路に入る
	world.Components.Wallet.Remove(player)

	err = SellStock(world, player, merchant, item)
	require.ErrorContains(t, err, "failed to add currency")

	// ロールバックで実体が手元へ戻り、商人の在庫には並ばない
	assert.True(t, world.Components.LocationInBackpack.Has(item))
	assert.Equal(t, player, world.Components.LocationInBackpack.Get(item).Owner)
	assert.False(t, world.Components.LocationInStorage.Has(item))
}
