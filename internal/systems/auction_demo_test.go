package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuctionPrice_重い安物は発送料で赤字になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// fire_extinguisher は価値40で重量5kg。発送料が落札額を食って手取りが負になる
	item, err := lifecycle.SpawnFieldItem(world, "fire_extinguisher", 5, 5, 1)
	require.NoError(t, err)

	bid := query.AuctionBid(world, item, true)
	ship := query.AuctionShippingCost(world, item)
	fee := query.AuctionFee(bid)
	assert.Positive(t, ship, "重量に応じた発送料がかかる")
	assert.Negative(t, bid-ship-fee, "基準価値が低く重い品は発送料で赤字になる")
}

func TestAuctionPrice_軽い高額品は黒字になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// angel_sword は価値500で重量2kg。軽く高価なので手取りが出る
	item, err := lifecycle.SpawnFieldItem(world, "angel_sword", 5, 5, 1)
	require.NoError(t, err)

	bid := query.AuctionBid(world, item, true)
	ship := query.AuctionShippingCost(world, item)
	fee := query.AuctionFee(bid)
	assert.Positive(t, bid-ship-fee, "軽く価値が高い品は手取りが出る")
}

func TestAuctionSaleChance_高価な品ほど落札されにくい(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	cheap, err := lifecycle.SpawnFieldItem(world, "fire_extinguisher", 5, 5, 1)
	require.NoError(t, err)
	expensive, err := lifecycle.SpawnFieldItem(world, "angel_sword", 6, 5, 1)
	require.NoError(t, err)

	cheapChance := query.AuctionSaleChance(world, cheap)
	expensiveChance := query.AuctionSaleChance(world, expensive)
	assert.Greater(t, cheapChance, expensiveChance, "安い品ほど落札されやすい")
	assert.Greater(t, cheapChance, 0.0)
	assert.LessOrEqual(t, cheapChance, 1.0)
	assert.Greater(t, expensiveChance, 0.0)
}

func TestAuctionDemoSystem_残り歩数0で落札か在庫戻りに解決する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	item, err := lifecycle.SpawnFieldItem(world, "angel_sword", 5, 6, 1)
	require.NoError(t, err)
	world.Components.AuctionListing.Add(item, &gc.AuctionListing{ID: 1, StepsLeft: 0, Slow: true, Announced: -1})

	sys := &AuctionDemoSystem{}
	require.NoError(t, sys.Update(world))

	// 落札されれば Won、落札されなければ出品が解かれて在庫に戻る。どちらでも解決済みとみなす
	resolved := !world.Components.AuctionListing.Has(item) || world.Components.AuctionListing.Get(item).Won
	assert.True(t, resolved, "残り歩数0で落札か在庫戻りに解決する")
}

func TestAuctionDemoSystem_動かなければ落札しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	item, err := lifecycle.SpawnFieldItem(world, "angel_sword", 5, 6, 1)
	require.NoError(t, err)
	world.Components.AuctionListing.Add(item, &gc.AuctionListing{StepsLeft: 3, Slow: true, Announced: -1})

	sys := &AuctionDemoSystem{}
	require.NoError(t, sys.Update(world))
	require.NoError(t, sys.Update(world))

	l := world.Components.AuctionListing.Get(item)
	assert.False(t, l.Won, "歩いていなければ落札しない")
	assert.Equal(t, 3, l.StepsLeft, "動かなければ残り歩数は減らない")
}

func TestSeedAuctionDemo_品に出品相互作用と発送台を用意する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	require.NoError(t, SeedAuctionDemo(world))

	taggable := 0
	dispenser := 0
	portal := 0
	q := ecs.NewFilter1[gc.Interactable](world.ECS).Query()
	for q.Next() {
		for _, kind := range world.Components.Interactable.Get(q.Entity()).Interactions {
			switch kind {
			case gc.InteractionApplyListingTag:
				taggable++
			case gc.InteractionDispenseListingTag:
				dispenser++
			case gc.InteractionShip:
				portal++
			default:
				// 数える対象以外は無視する
			}
		}
	}
	assert.Equal(t, len(auctionDemoItems), taggable, "撒いた品は全て出品タグを貼れる")
	assert.Equal(t, 1, dispenser, "発券機が1つ置かれる")
	assert.Equal(t, 1, portal, "発送ポータルが1つ置かれる")
}

func TestAuctionDemoSystem_出品が無ければ何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	sys := &AuctionDemoSystem{}
	assert.NoError(t, sys.Update(world), "出品が無ければ即座に終わる")
}

// プレイヤーが実際に歩くと、出品の残り歩数が減って落札または在庫戻りに解決することを確かめる。
// ゲーム内でターンが進まず解決しないという不具合の回帰を防ぐ。ターン制でなく移動そのもので
// 進むので、どの場面でも歩けば必ず解決へ近づく。落札されるかはパラメータ次第なので、
// 解決とは落札または在庫戻りのいずれかを指す。
func TestAuctionDemoSystem_実移動で歩数が減り解決する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 25, Y: 25}, "ash")
	require.NoError(t, err)
	item, err := lifecycle.SpawnFieldItem(world, "angel_sword", 26, 25, 1)
	require.NoError(t, err)
	world.Components.AuctionListing.Add(item, &gc.AuctionListing{ID: 1, StepsLeft: query.AuctionSlowSteps, Slow: true, Announced: -1})

	resolved := func() bool {
		return !world.Components.AuctionListing.Has(item) || world.Components.AuctionListing.Get(item).Won
	}

	turnSys := &TurnSystem{}
	auctionSys := &AuctionDemoSystem{}
	moves := 0
	for i := 0; i < 60 && !resolved(); i++ {
		if query.CanPlayerAct(world) && !query.HasActivity(world, player) {
			dir := gc.DirectionRight
			if i%2 == 1 {
				dir = gc.DirectionLeft
			}
			require.NoError(t, activity.ExecuteMoveAction(world, dir))
			moves++
		}
		for range 3 {
			require.NoError(t, turnSys.Update(world))
			require.NoError(t, auctionSys.Update(world))
		}
	}

	assert.True(t, resolved(), "実際に歩けば残り歩数が減って解決する")
	assert.LessOrEqual(t, moves, query.AuctionSlowSteps+3, "移動数ぶんで解決する")
}
