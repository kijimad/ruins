package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuctionPrice_重い安物は発送料で赤字になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// fire_extinguisher は価値40で重量5kg。発送料が落札額を食って手取りが負になる
	item, err := lifecycle.SpawnFieldItem(world, "fire_extinguisher", 5, 5, 1)
	require.NoError(t, err)

	bid := query.AuctionOpeningBid(world, item)
	ship := query.AuctionShippingCost(world, item)
	fee := query.AuctionFee(bid)
	assert.Positive(t, ship, "重量に応じた発送料がかかる")
	assert.Negative(t, bid-ship-fee, "基準価値が低く重い品は開始入札の時点で発送料に負ける")
}

func TestAuctionPrice_軽い高額品は黒字になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// angel_sword は価値500で重量2kg。軽く高価なので手取りが出る
	item, err := lifecycle.SpawnFieldItem(world, "angel_sword", 5, 5, 1)
	require.NoError(t, err)

	bid := query.AuctionOpeningBid(world, item)
	ship := query.AuctionShippingCost(world, item)
	fee := query.AuctionFee(bid)
	assert.Positive(t, bid-ship-fee, "軽く価値が高い品は開始入札の時点で手取りが出る")
}

func TestAuctionRaise_入札の上げ幅は必ず1以上になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 入札が来たのに現在値が動かない事態を防ぐため、上げ幅は最低1を保証する
	item, err := lifecycle.SpawnFieldItem(world, "fire_extinguisher", 5, 5, 1)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, query.AuctionRaise(world, item), 1, "上げ幅は必ず1以上")
}

func TestStartAuctionListing_連番を採番して出品する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	first, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	second, err := lifecycle.SpawnBackpackItem(world, "shovel", 1)
	require.NoError(t, err)

	n1 := query.StartAuctionListing(world, first, 0)
	n2 := query.StartAuctionListing(world, second, 0)

	assert.Equal(t, 1, n1, "最初の出品は #1")
	assert.Equal(t, 2, n2, "次の出品は #2")
	require.True(t, world.Components.AuctionListing.Has(first), "タグを貼ると出品中になる")
	assert.Positive(t, world.Components.AuctionListing.Get(first).CurrentBid, "開始入札で現在値が付く")
}

func TestAuctionSystem_出品中の品はターン経過で落札されるが入金はしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	item, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	before := query.GetCurrency(world, player)
	query.StartAuctionListing(world, item, int(query.GetGameTime(world).TotalTurns))

	sys := &AuctionSystem{}
	sold := false
	for i := 0; i < 200 && !sold; i++ {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
		sold = world.Components.AuctionSold.Has(item)
	}

	require.True(t, sold, "入札が止まったターンに落札が確定する")
	assert.True(t, world.ECS.Alive(item), "落札しても品は消えず落札済みで残る")
	assert.False(t, world.Components.AuctionListing.Has(item), "落札で出品中は外れる")
	assert.Equal(t, before, query.GetCurrency(world, player), "落札だけでは入金しない")
	assert.Empty(t, query.GetAuctionHistory(world).Records, "出荷するまで履歴には残らない")
}

func TestAuctionSystem_期限内に出荷しないと評判が下がる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	item, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	query.StartAuctionListing(world, item, int(query.GetGameTime(world).TotalTurns))

	sys := &AuctionSystem{}
	for i := 0; i < 200 && !world.Components.AuctionSold.Has(item); i++ {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
	}
	require.True(t, world.Components.AuctionSold.Has(item), "落札済みになる")

	before := query.GetAuctionHistory(world).Reputation
	due := world.Components.AuctionSold.Get(item).DueTurn

	// 積荷へ渡さないまま出荷期限を過ぎる
	for int(query.GetGameTime(world).TotalTurns) <= due+1 {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
	}

	assert.Equal(t, before-auctionReputationPenalty, query.GetAuctionHistory(world).Reputation, "期限を破ると評判が下がる")
	assert.True(t, world.Components.AuctionSold.Get(item).Penalized, "ペナルティ済みの印が付く")
	assert.True(t, world.ECS.Alive(item), "期限を破っても品は消えない")

	// さらにターンが過ぎても二重には罰しない
	penalized := query.GetAuctionHistory(world).Reputation
	for range 5 {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
	}
	assert.Equal(t, penalized, query.GetAuctionHistory(world).Reputation, "同じ品で二重に評判を下げない")
}

func TestCollectStagedItems_集荷は明細を発生させ所持金は動かさない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	won, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	junk, err := lifecycle.SpawnBackpackItem(world, "shovel", 1)
	require.NoError(t, err)
	query.StartAuctionListing(world, won, int(query.GetGameTime(world).TotalTurns))

	// won だけ落札まで進める。junk はタグを貼らないので未出品のまま
	sys := &AuctionSystem{}
	for i := 0; i < 200 && !world.Components.AuctionSold.Has(won); i++ {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
	}
	require.True(t, world.Components.AuctionSold.Has(won), "won は落札済みになる")

	// 両方を積荷に回す。集荷は落札済みだけ受取金を立て、それ以外は精算されず消える
	world.Components.AuctionStaged.Add(won, &gc.AuctionStaged{})
	world.Components.AuctionStaged.Add(junk, &gc.AuctionStaged{})

	bid := world.Components.AuctionSold.Get(won).Bid
	expectedNet := bid - query.AuctionShippingCost(world, won) - query.AuctionFee(bid)
	before := query.GetCurrency(world, player)

	collected, receipts := query.CollectStagedItems(world)

	assert.Equal(t, 2, collected, "積んだ品はすべて集荷される")
	assert.Equal(t, 1, receipts, "受取金の明細は落札済みの分だけ")
	assert.Equal(t, before, query.GetCurrency(world, player), "集荷では所持金は動かない")
	assert.False(t, world.ECS.Alive(won), "落札済みは集荷で手放す")
	assert.False(t, world.ECS.Alive(junk), "未落札も集荷され金にならず消える")

	entries := query.GetAuctionHistory(world).Entries
	require.Len(t, entries, 2, "受取金1件と集荷料金の請求1件が金銭タブへ届く")
	assert.Equal(t, gc.AuctionEntryReceipt, entries[0].Kind, "先頭は受取金")
	assert.Equal(t, expectedNet, entries[0].Amount, "受取金は落札額から配送料と手数料を引いた手取り")
	assert.Equal(t, gc.AuctionEntryInvoice, entries[1].Kind, "末尾は集荷料金の請求")
	assert.Equal(t, query.AuctionPickupFee, entries[1].Amount, "請求額は集荷手数料")
}

func TestSettleAuctionEntry_受取金は加算し請求は減算する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	h := query.GetAuctionHistory(world)
	h.Entries = []gc.AuctionEntry{
		{Kind: gc.AuctionEntryReceipt, Number: 1, Name: "Angel Sword", Amount: 300, Bid: 400, Ship: 50, Fee: 50},
		{Kind: gc.AuctionEntryInvoice, Name: "Pickup fee", Amount: query.AuctionPickupFee},
	}
	before := query.GetCurrency(world, player)

	// 受取金を精算すると所持金が増え、履歴へ移る
	got, ok := query.SettleAuctionEntry(world, player, 0, 7)
	require.True(t, ok)
	assert.Equal(t, gc.AuctionEntryReceipt, got.Kind)
	assert.Equal(t, before+300, query.GetCurrency(world, player), "受取金は所持金へ加える")
	require.Len(t, query.GetAuctionHistory(world).Entries, 1, "精算した明細は消える")
	records := query.GetAuctionHistory(world).Records
	require.Len(t, records, 1, "精算した受取金は履歴へ移る")
	assert.Equal(t, 7, records[0].Turn)

	// 請求を精算すると所持金が減る
	_, ok = query.SettleAuctionEntry(world, player, 0, 8)
	require.True(t, ok)
	assert.Equal(t, before+300-query.AuctionPickupFee, query.GetCurrency(world, player), "請求は所持金から引く")
	assert.Empty(t, query.GetAuctionHistory(world).Entries, "明細をすべて精算した")
}

func TestAuctionSystem_積荷はタイマーで集荷され明細が届く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	item, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	query.StartAuctionListing(world, item, int(query.GetGameTime(world).TotalTurns))

	sys := &AuctionSystem{}
	for i := 0; i < 200 && !world.Components.AuctionSold.Has(item); i++ {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
	}
	require.True(t, world.Components.AuctionSold.Has(item), "落札済みになる")

	// 落札済みを積荷へ回す。以後はプレイヤーの操作なしに集荷タイマーで集荷される
	world.Components.AuctionStaged.Add(item, &gc.AuctionStaged{})
	before := query.GetCurrency(world, player)

	for i := 0; i < auctionShipDelay+2 && world.ECS.Alive(item); i++ {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
	}

	assert.False(t, world.ECS.Alive(item), "積荷はタイマー満了で集荷され手放す")
	assert.Equal(t, before, query.GetCurrency(world, player), "集荷だけでは所持金は動かない。精算で入る")
	assert.NotEmpty(t, query.GetAuctionHistory(world).Entries, "受取金と請求の明細が金銭タブへ届く")
}

func TestCollectStagedItems_積荷が無ければ何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	collected, receipts := query.CollectStagedItems(world)

	assert.Zero(t, collected, "積荷が無ければ0件")
	assert.Zero(t, receipts, "受取金の明細も0件")
	assert.Empty(t, query.GetAuctionHistory(world).Entries, "明細は発生しない")
}

func TestMarkShippingStations_専用propを出荷場所にする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	station, err := lifecycle.SpawnProp(world, "shipping_station", 5, 5)
	require.NoError(t, err)

	markShippingStations(world)

	require.True(t, world.Components.AuctionStation.Has(station), "shipping_station prop は出荷場所になる")
	it := world.Components.Interactable.Get(station)
	assert.Equal(t, []gc.InteractionKind{gc.InteractionAuction}, it.Interactions, "出荷場所メニューを開く相互作用が付く")
}

func TestMarkShippingStations_他のpropには触れない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 5, 5)
	require.NoError(t, err)

	markShippingStations(world)

	assert.False(t, world.Components.AuctionStation.Has(crate), "shipping_station 以外の prop には触れない")
}

func TestAuctionSystem_出品が無ければ何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	sys := &AuctionSystem{}
	assert.NoError(t, sys.Update(world), "出品中の品が無ければ即座に終わる")
}
