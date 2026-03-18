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

func TestInventoryMenuState_OnStart(t *testing.T) {
	t.Parallel()

	state := &InventoryMenuState{}
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)
	assert.NotNil(t, state.menuMount, "menuMountが初期化されている")
	assert.NotNil(t, state.windowMount, "windowMountが初期化されている")
}

func TestInventoryMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &InventoryMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)

	assert.Equal(t, 3, len(props.Tabs), "タブは3つ（道具、武器、防具）")
	assert.Equal(t, "items", props.Tabs[0].ID)
	assert.Equal(t, "weapons", props.Tabs[1].ID)
	assert.Equal(t, "wearables", props.Tabs[2].ID)
}

func TestInventoryMenuState_TabNavigation(t *testing.T) {
	t.Parallel()

	state := &InventoryMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)
	state.menuMount.SetProps(props)

	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	hooks.UseTabMenu(state.menuMount.Store(), "inventory", hooks.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: itemCounts,
	})
	state.menuMount.Update()

	// 初期状態
	tabIndex, _ := hooks.GetState[int](state.menuMount, "inventory_tabIndex")
	assert.Equal(t, 0, tabIndex, "初期タブインデックスは0")

	// 右に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuTabNext)
	tabIndex, _ = hooks.GetState[int](state.menuMount, "inventory_tabIndex")
	assert.Equal(t, 1, tabIndex, "右移動後は1")

	// 左に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuTabPrev)
	tabIndex, _ = hooks.GetState[int](state.menuMount, "inventory_tabIndex")
	assert.Equal(t, 0, tabIndex, "左移動後は0")
}

func TestInventoryMenuState_DoAction_Cancel(t *testing.T) {
	t.Parallel()

	state := &InventoryMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "キャンセルでTransPop")
}

func TestInventoryMenuState_DoAction_CloseMenu(t *testing.T) {
	t.Parallel()

	state := &InventoryMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionCloseMenu)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "CloseMenuでTransPop")
}

func TestInventoryMenuState_DoAction_Navigation(t *testing.T) {
	t.Parallel()

	state := &InventoryMenuState{}
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

func TestInventoryMenuState_WindowProps(t *testing.T) {
	t.Parallel()

	state := &InventoryMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 初期状態ではメニューモード
	assert.Equal(t, invSubStateMenu, state.subState, "初期状態ではメニューモード")
}

func TestInventoryMenuState_GetActionItems(t *testing.T) {
	t.Parallel()

	state := &InventoryMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// エンティティが0の場合は空のリスト
	actions := state.getActionItems(world, 0)
	assert.Empty(t, actions, "エンティティが0の場合は空")
}

func TestInventoryMenuState_DoAction_WindowMode(t *testing.T) {
	t.Parallel()

	state := &InventoryMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// ウィンドウを開く
	state.subState = invSubStateWindow
	state.windowMount.SetProps(windowProps{
		SelectedEntity: 1, // ダミーエンティティ
	})

	// ウィンドウモードでのキャンセル
	transition, err := state.DoAction(world, inputmapper.ActionWindowCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type, "ウィンドウキャンセルはTransNone")

	// ウィンドウが閉じている
	assert.Equal(t, invSubStateMenu, state.subState, "キャンセル後はメニューモード")
}

func TestInventoryMenuState_DoAction_WindowNavigation(t *testing.T) {
	t.Parallel()

	state := &InventoryMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// ウィンドウを開く
	state.subState = invSubStateWindow
	state.windowMount.SetProps(windowProps{
		SelectedEntity: 1,
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

func TestNewInventoryMenuState(t *testing.T) {
	t.Parallel()

	factory := NewInventoryMenuState
	state := factory()

	assert.NotNil(t, state, "Stateが作成される")
	_, ok := state.(*InventoryMenuState)
	assert.True(t, ok, "InventoryMenuState型である")
}
