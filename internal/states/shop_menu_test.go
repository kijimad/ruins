package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShopMenuState_OnStart(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)
	assert.False(t, state.detail.Active(), "初期状態で詳細モーダルは閉じている")
}

func TestShopMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	// ショップはプレイヤーが居て初めて開く。価格はプレイヤーの交渉スキルから決まるため必須
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	props, err := state.Fetch(world)
	require.NoError(t, err)

	assert.Len(t, props.Tabs, 2, "タブは2つ（購入、売却）")
	assert.Equal(t, "buy", props.Tabs[0].ID)
	assert.Equal(t, "sell", props.Tabs[1].ID)
}

// TestShopMenuState_FetchProps_同一スタックは1行で額と個数は全量 は、店頭の束ね表示を固定する。
// 在庫5個が1行に束ねられ、行の額は全量、購入可否も全量の額で判定される。
// 表示個数と操作個数の一致は gameaction 層で固定済みで、ここは表示側の束ねを固定する
func TestShopMenuState_FetchProps_同一スタックは1行で額と個数は全量(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	merchant := world.ECS.NewEntity()
	state := &ShopMenuState{merchant: merchant}
	require.NoError(t, state.OnStart(world))

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 同一品種5個を在庫へ。買いタブでは1行に束ねられる
	rep, err := lifecycle.SpawnStorageItem(world, "healing_potion", 5, merchant)
	require.NoError(t, err)

	// 全量の額に1だけ足りない所持金にし、購入不可判定が全量基準であることを見る
	total := query.BuyPrice(world, player, rep)
	world.Components.Wallet.Get(player).Currency = total - 1

	props, err := state.Fetch(world)
	require.NoError(t, err)

	buy := props.Tabs[0]
	require.Len(t, buy.Items, 1, "同一スタックは1行に束ねる")
	assert.Equal(t, 5, buy.Items[0].Count, "行の個数は束の全量")
	assert.Equal(t, total, buy.Items[0].Price, "行の額は全量")
	assert.True(t, buy.Items[0].Disabled, "全量の額に届かなければ購入不可")

	// 全量ちょうど持てば購入可
	world.Components.Wallet.Get(player).Currency = total
	props, err = state.Fetch(world)
	require.NoError(t, err)
	assert.False(t, props.Tabs[0].Items[0].Disabled, "全量の額があれば購入可")
}

func TestShopMenuState_DoAction_Cancel(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "キャンセルでTransPop")
}

func TestShopMenuState_DoAction_CloseMenu(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionCloseMenu)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "CloseMenuでTransPop")
}

func TestShopMenuState_DoAction_未選択のSelectは売買せず何もしない(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// Update を回していないのでカーソルは未確定。商品未選択の Select は売買せず TransNone を返す。
	// カーソルを載せてからの売買の副作用は結合テストで別途検証する
	transition, err := state.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type, "未選択の選択はTransNone")
}

func TestShopMenuState_detailContent_選択なしは表示しない(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	state.detail.Open(world)
	assert.False(t, state.detail.Active(), "商品未選択では詳細モーダルを出さない")
}

func TestNewShopMenuState(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	merchant := world.ECS.NewEntity()

	state, err := NewShopMenuState(merchant)
	require.NoError(t, err)
	assert.NotNil(t, state, "Stateが作成される")
	_, ok := state.(*ShopMenuState)
	assert.True(t, ok, "ShopMenuState型である")
}

// TestShopMenuState_buildItemListUI_商品ありで行を組む は、売買メニューが商品を1件以上持つとき
// 行が組まれることを固定する。golden は既定タブが空で行に到達せず覆えないため、実体を直接渡す。
func TestShopMenuState_buildItemListUI_商品ありで行を組む(t *testing.T) {
	t.Parallel()

	world := vrt.InitUIWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	e, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3)
	require.NoError(t, err)

	tabs := []shopTabData{{ID: "sell", Items: []shopItemData{{Entity: e, Price: 10}}}}
	st := &ShopMenuState{}

	items, _ := st.buildItemListUI(world, tabs, 0, 0, 10, world.Resources.UIResources)
	assert.NotEmpty(t, items, "商品ありで行が組まれる")
}
