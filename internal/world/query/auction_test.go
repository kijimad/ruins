package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuctionFee_落札額に比例した手数料を返す(t *testing.T) {
	t.Parallel()

	assert.Equal(t, consts.Currency(0), AuctionFee(0), "落札額0なら手数料も0")
	assert.Equal(t, consts.Currency(12), AuctionFee(100), "12%の手数料")
	assert.Equal(t, consts.Currency(120), AuctionFee(1000))
}

func TestAuctionShippingCost_重量に比例した送料を返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 2kgの品。1kgあたり25なので送料は50
	item := world.ECS.NewEntity()
	world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.Milligram(2 * consts.MilligramPerKg)})

	assert.Equal(t, consts.Currency(50), AuctionShippingCost(world, item))
}

func TestAuctionShippingCost_重量が無ければ送料0(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	item := world.ECS.NewEntity()
	assert.Equal(t, consts.Currency(0), AuctionShippingCost(world, item))
}

func TestAuctionOpeningBid_基準価値の分散範囲内に収まる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	item := world.ECS.NewEntity()
	world.Components.Value.Add(item, &gc.Value{Value: 1000})

	// 基準価値1000 * 0.4 = 400。分散は0.8〜1.2倍なので320〜480に収まる
	for range 20 {
		bid := AuctionOpeningBid(world, item)
		assert.GreaterOrEqual(t, bid, consts.Currency(320))
		assert.LessOrEqual(t, bid, consts.Currency(480))
	}
}

func TestAuctionRaise(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	t.Run("基準価値が大きい場合は分散範囲内", func(t *testing.T) {
		t.Parallel()
		item := world.ECS.NewEntity()
		world.Components.Value.Add(item, &gc.Value{Value: 1000})

		// 基準価値1000 * 0.15 = 150。分散は0.8〜1.2倍なので120〜180に収まる
		for range 20 {
			raise := AuctionRaise(world, item)
			assert.GreaterOrEqual(t, raise, consts.Currency(120))
			assert.LessOrEqual(t, raise, consts.Currency(180))
		}
	})

	t.Run("基準価値が0でも最低1を保証する", func(t *testing.T) {
		t.Parallel()
		item := world.ECS.NewEntity()

		raise := AuctionRaise(world, item)
		assert.Equal(t, consts.Currency(1), raise, "価値0でも上げ幅は最低1")
	})
}

func TestGetAuctionHistory_InitWorldで初期状態を取得できる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	history := GetAuctionHistory(world)
	require.NotNil(t, history)
	assert.Equal(t, 0, history.NextNumber)
	assert.Empty(t, history.Entries)
	assert.Empty(t, history.Records)
}

func TestStartAuctionListing_採番して開始入札のタグを貼る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	item1 := world.ECS.NewEntity()
	world.Components.Value.Add(item1, &gc.Value{Value: 100})
	item2 := world.ECS.NewEntity()
	world.Components.Value.Add(item2, &gc.Value{Value: 100})

	number1 := StartAuctionListing(world, item1, 10)
	number2 := StartAuctionListing(world, item2, 20)

	assert.Equal(t, 1, number1, "最初の出品は1番")
	assert.Equal(t, 2, number2, "2件目は連番で2番")

	require.True(t, world.Components.AuctionListing.Has(item1))
	listing1 := world.Components.AuctionListing.Get(item1)
	assert.Equal(t, 1, listing1.Number)
	assert.Equal(t, 10, listing1.LastTurn)
	assert.Positive(t, listing1.CurrentBid, "開始入札は正の額")

	assert.Equal(t, 2, GetAuctionHistory(world).NextNumber, "履歴の採番カウンタも進む")
}

func TestCollectStagedItems_積荷が空なら何も集荷しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	station := world.ECS.NewEntity()

	collected, receipts := CollectStagedItems(world, station)

	assert.Equal(t, 0, collected)
	assert.Equal(t, 0, receipts)
	assert.Empty(t, GetAuctionHistory(world).Entries, "明細も増えない")
}

func TestCollectStagedItems_落札済みと未落札を集荷して明細をためる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	station := world.ECS.NewEntity()

	// 落札済みの品。受取金の明細を1件生む
	sold := world.ECS.NewEntity()
	world.Components.Name.Add(sold, &gc.Name{Name: "落札品"})
	world.Components.LocationInStorage.Add(sold, &gc.LocationInStorage{Owner: station})
	world.Components.AuctionSold.Add(sold, &gc.AuctionSold{Number: 1, Bid: 1000})

	// 未落札のまま積荷にある品。明細を生まず、ただ手放される
	unsold := world.ECS.NewEntity()
	world.Components.LocationInStorage.Add(unsold, &gc.LocationInStorage{Owner: station})

	collected, receipts := CollectStagedItems(world, station)

	assert.Equal(t, 2, collected, "積荷2件をまとめて集荷")
	assert.Equal(t, 1, receipts, "受取金の明細は落札済みの1件だけ")
	assert.False(t, world.ECS.Alive(sold), "集荷した品は消える")
	assert.False(t, world.ECS.Alive(unsold), "未落札の品も消える")

	history := GetAuctionHistory(world)
	require.Len(t, history.Entries, 2, "受取金1件+集荷料金の請求1件")
	assert.Equal(t, gc.AuctionEntryReceipt, history.Entries[0].Kind)
	assert.Equal(t, consts.Currency(1000), history.Entries[0].Bid)
	assert.Equal(t, gc.AuctionEntryInvoice, history.Entries[1].Kind)
	assert.Equal(t, AuctionPickupFee, history.Entries[1].Amount, "集荷料金は集荷1回につき定額")
}

func TestSettleAuctionEntry(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 0})

	t.Run("負のインデックス", func(t *testing.T) {
		t.Parallel()
		_, ok := SettleAuctionEntry(world, player, -1, 0)
		assert.False(t, ok)
	})

	t.Run("要素数以上のインデックス", func(t *testing.T) {
		t.Parallel()
		_, ok := SettleAuctionEntry(world, player, 0, 0)
		assert.False(t, ok, "明細が空なので0番目も範囲外")
	})
}

func TestSettleAuctionEntry_受取金は所持金へ加え実績へ移す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 100})

	history := GetAuctionHistory(world)
	history.Entries = append(history.Entries, gc.AuctionEntry{
		Kind: gc.AuctionEntryReceipt, Number: 1, Name: "売れた品", Amount: 500, Bid: 600, Ship: 50, Fee: 50,
	})

	entry, ok := SettleAuctionEntry(world, player, 0, 42)

	require.True(t, ok)
	assert.Equal(t, consts.Currency(500), entry.Amount)
	assert.Equal(t, consts.Currency(600), GetCurrency(world, player), "受取金が所持金へ加わる")
	assert.Empty(t, history.Entries, "精算した明細は取り除かれる")
	require.Len(t, history.Records, 1, "出荷実績へ1件移る")
	assert.Equal(t, 42, history.Records[0].Turn)
	assert.Equal(t, consts.Currency(500), history.Records[0].Net)
}

func TestSettleAuctionEntry_請求は所持金から引く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 300})

	history := GetAuctionHistory(world)
	history.Entries = append(history.Entries, gc.AuctionEntry{
		Kind: gc.AuctionEntryInvoice, Name: "集荷料金", Amount: AuctionPickupFee,
	})

	_, ok := SettleAuctionEntry(world, player, 0, 1)

	require.True(t, ok)
	assert.Equal(t, consts.Currency(200), GetCurrency(world, player), "請求額が所持金から引かれる")
	assert.Empty(t, history.Entries)
	assert.Empty(t, history.Records, "請求は出荷実績に残らない")
}

func TestSettleAuctionEntry_Walletが無ければ失敗し明細は残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	playerWithoutWallet := world.ECS.NewEntity()

	history := GetAuctionHistory(world)
	history.Entries = append(history.Entries, gc.AuctionEntry{
		Kind: gc.AuctionEntryReceipt, Amount: 100,
	})

	_, ok := SettleAuctionEntry(world, playerWithoutWallet, 0, 0)

	assert.False(t, ok)
	assert.Len(t, history.Entries, 1, "精算に失敗した明細は取り除かれない")
}
