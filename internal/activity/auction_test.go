package activity

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

func TestExecuteApplyListingTag_出品タグを貼ると出品状態が付く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	// 出品タグを持ち物に用意する。タグは同一の消耗品で識別子を持たない
	tagEntity := world.ECS.NewEntity()
	world.Components.AuctionTag.Add(tagEntity, &gc.AuctionTag{})
	world.Components.LocationInBackpack.Add(tagEntity, &gc.LocationInBackpack{Owner: player})

	item, err := lifecycle.SpawnFieldItem(world, "angel_sword", 11, 10, 1)
	require.NoError(t, err)

	_, err = executeApplyListingTag(player, item, world)
	require.NoError(t, err)

	require.True(t, world.Components.AuctionListing.Has(item), "貼ると出品状態が付く")
	l := world.Components.AuctionListing.Get(item)
	assert.Positive(t, l.ID, "貼るときに一意識別子が採番される")
	assert.Equal(t, query.AuctionSlowSteps, l.StepsLeft, "入札期間が設定される")
	assert.False(t, world.ECS.Alive(tagEntity), "貼った出品タグは消費される")
}

func TestExecuteApplyListingTag_タグが無ければ出品しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	item, err := lifecycle.SpawnFieldItem(world, "angel_sword", 11, 10, 1)
	require.NoError(t, err)

	_, err = executeApplyListingTag(player, item, world)
	require.NoError(t, err)

	assert.False(t, world.Components.AuctionListing.Has(item), "出品タグを持っていなければ出品されない")
}

func TestExecuteShip_落札品を発送して財布へ入れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	before := query.GetCurrency(world, player)

	item, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	world.Components.AuctionListing.Add(item, &gc.AuctionListing{ID: 3, Won: true, Bid: 400})

	_, err = executeShip(player, world)
	require.NoError(t, err)

	assert.False(t, world.ECS.Alive(item), "発送した落札品は消える")
	assert.Greater(t, query.GetCurrency(world, player), before, "手取りが財布へ加わる")
}

func TestExecuteShip_落札していない品は発送しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	before := query.GetCurrency(world, player)

	item, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	world.Components.AuctionListing.Add(item, &gc.AuctionListing{ID: 3, Won: false, StepsLeft: 3})

	_, err = executeShip(player, world)
	require.NoError(t, err)

	assert.True(t, world.ECS.Alive(item), "落札前の品は発送されない")
	assert.Equal(t, before, query.GetCurrency(world, player), "手取りは動かない")
}
