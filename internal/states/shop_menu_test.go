package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShopMenuState_OnStart(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)
	assert.NotNil(t, state.menuMount, "menuMountが初期化されている")
	assert.NotNil(t, state.windowMount, "windowMountが初期化されている")
}

func TestShopMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)

	assert.Equal(t, 2, len(props.Tabs), "タブは2つ（購入、売却）")
	assert.Equal(t, "buy", props.Tabs[0].ID)
	assert.Equal(t, "sell", props.Tabs[1].ID)
}

func TestShopMenuState_TabNavigation(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)
	state.menuMount.SetProps(props)

	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	hooks.UseTabMenu(state.menuMount.Store(), "shop", hooks.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: itemCounts,
	})
	state.menuMount.Update()

	// 初期状態
	tabIndex, _ := hooks.GetState[int](state.menuMount, "shop_tabIndex")
	assert.Equal(t, 0, tabIndex, "初期タブインデックスは0")

	// 右に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuRight)
	tabIndex, _ = hooks.GetState[int](state.menuMount, "shop_tabIndex")
	assert.Equal(t, 1, tabIndex, "右移動後は1")

	// 循環して戻る
	state.menuMount.Dispatch(inputmapper.ActionMenuRight)
	tabIndex, _ = hooks.GetState[int](state.menuMount, "shop_tabIndex")
	assert.Equal(t, 0, tabIndex, "循環して0に戻る")
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
	}

	for _, action := range actions {
		transition, err := state.DoAction(world, action)
		require.NoError(t, err)
		assert.Equal(t, es.TransNone, transition.Type, "ナビゲーションはTransNone: %s", action)
	}
}

func TestShopMenuState_WindowProps(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 初期状態ではメニューモード
	assert.Equal(t, shopSubStateMenu, state.subState, "初期状態ではメニューモード")
}

func TestShopMenuState_DoAction_WindowMode(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// ウィンドウを開く
	state.subState = shopSubStateWindow
	state.windowMount.SetProps(shopWindowProps{
		SelectedItem: shopItemData{
			Label: "テスト",
			Price: 100,
			IsBuy: true,
		},
	})

	// ウィンドウモードでのキャンセル
	transition, err := state.DoAction(world, inputmapper.ActionWindowCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type, "ウィンドウキャンセルはTransNone")

	// ウィンドウが閉じている
	assert.Equal(t, shopSubStateMenu, state.subState, "キャンセル後はメニューモード")
}

func TestShopMenuState_DoAction_WindowNavigation(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// ウィンドウを開く
	state.subState = shopSubStateWindow
	state.windowMount.SetProps(shopWindowProps{
		SelectedItem: shopItemData{
			Label: "テスト",
			Price: 100,
			IsBuy: true,
		},
	})

	// ウィンドウ用のUseStateを登録
	state.setupWindowState(world)
	state.windowMount.Update()

	// ウィンドウモードでの上下移動
	transition, err := state.DoAction(world, inputmapper.ActionWindowDown)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type)

	transition, err = state.DoAction(world, inputmapper.ActionWindowUp)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type)
}

func TestNewShopMenuState(t *testing.T) {
	t.Parallel()

	factory := NewShopMenuState
	state := factory()

	assert.NotNil(t, state, "Stateが作成される")
	_, ok := state.(*ShopMenuState)
	assert.True(t, ok, "ShopMenuState型である")
}

func TestShopMenuState_GetActionItems(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 空のアイテムの場合は閉じるのみ
	actions := state.getActionItems(world, shopItemData{})
	assert.Equal(t, []string{TextClose}, actions, "空のアイテムは閉じるのみ")

	// 売却アイテムの場合
	sellActions := state.getActionItems(world, shopItemData{
		Label: "テスト",
		Price: 100,
		IsBuy: false,
	})
	assert.Contains(t, sellActions, "売却する", "売却オプションがある")
	assert.Contains(t, sellActions, TextClose, "閉じるオプションがある")
}
