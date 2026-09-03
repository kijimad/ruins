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

// TestBuyStock_スタッカブルはバックパックのスタックに統合される は商人在庫のスタッカブルを買うと
// バックパック内の同名アイテムと同一エンティティに統合されることを確認する。
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

// TestBuyStock_交渉スキルで買値が変わる は買値倍率が購入価格に反映されることを確認する。
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

	// 店頭の表示価格と取引で引かれる額が同じ1関数から出る
	expected := query.BuyPrice(world, player, item)
	assert.Less(t, expected, query.CalculateBuyPrice(woodenSwordValue), "交渉スキルで基準価格より安くなる")

	require.NoError(t, BuyStock(world, player, item))

	assert.Equal(t, 1000-expected, query.GetCurrency(world, player), "表示価格と同額が引かれる")
}

// TestSellStock_価値0のアイテムは対価0で売れる は無価値な品でも売却は成功し、対価が0になることを確認する。
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

	// 売った品は商人の在庫へ並ぶ
	require.True(t, world.Components.LocationInStorage.Has(item), "実体は商人の収納へ移る")
	assert.Equal(t, merchant, world.Components.LocationInStorage.Get(item).Owner)
}

// TestSellStock_交渉スキルで売値が変わる は売値倍率が売却価格に反映されることを確認する。
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

	// 店頭の表示価格と取引で得る額が同じ1関数から出る
	expected := query.SellPrice(world, player, item)
	assert.Greater(t, expected, query.CalculateSellPrice(100), "交渉スキルで基準価格より高く売れる")

	require.NoError(t, SellStock(world, player, merchant, item))

	assert.Equal(t, expected, query.GetCurrency(world, player), "表示価格と同額を得る")

	// 売った品は商人の在庫へ並ぶ
	require.True(t, world.Components.LocationInStorage.Has(item))
	assert.Equal(t, merchant, world.Components.LocationInStorage.Get(item).Owner)
}
