package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
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

	props := state.Fetch(world)

	assert.Len(t, props.Tabs, 2, "タブは2つ（購入、売却）")
	assert.Equal(t, "buy", props.Tabs[0].ID)
	assert.Equal(t, "sell", props.Tabs[1].ID)
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

func TestShopMenuState_DoAction_Navigation(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// ナビゲーションアクションはTransNoneを返す
	actions := []inputmapper.ActionID{
		inputmapper.ActionMenuUp,
		inputmapper.ActionMenuDown,
		inputmapper.ActionMenuLeft,
		inputmapper.ActionMenuRight,
		inputmapper.ActionMenuTabNext,
		inputmapper.ActionMenuTabPrev,
	}

	for _, action := range actions {
		transition, err := state.DoAction(world, action)
		require.NoError(t, err)
		assert.Equal(t, es.TransNone, transition.Type, "ナビゲーションはTransNone: %s", action)
	}
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

// TestShopMenuState_buildItemContainer_商品ありでpanicしない は、売買メニューが商品を
// 1件以上持つとき列数不整合で落ちないことを固定する。行のセル数が列幅数と合わないと
// renderMenuList が panic する。golden は既定タブが空で行に到達せず覆えないため、
// 実体を直接渡してこの経路を覆う。
func TestShopMenuState_buildItemContainer_商品ありでpanicしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.UIResources = vrt.SharedUIResources(t)

	e, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3)
	require.NoError(t, err)

	tabs := []shopTabData{{ID: "sell", Items: []shopItemData{{Entity: e, Price: 10}}}}
	st := &ShopMenuState{}

	assert.NotPanics(t, func() {
		vrt.WithUILock(func() {
			_ = st.buildItemContainer(world, tabs, 0, 0, world.Resources.UIResources)
		})
	})
}
