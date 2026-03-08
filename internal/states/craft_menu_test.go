package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCraftMenuState_OnStart(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)
	assert.NotNil(t, state.menuMount, "menuMountが初期化されている")
	assert.NotNil(t, state.windowMount, "windowMountが初期化されている")
	assert.NotNil(t, state.resultMount, "resultMountが初期化されている")
}

func TestCraftMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)

	assert.Equal(t, 3, len(props.Tabs), "タブは3つ（道具、武器、装備）")
	assert.Equal(t, "consumables", props.Tabs[0].ID)
	assert.Equal(t, "weapons", props.Tabs[1].ID)
	assert.Equal(t, "wearables", props.Tabs[2].ID)
}

func TestCraftMenuState_TabNavigation(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)
	state.menuMount.SetProps(props)

	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	ui.UseTabMenu(state.menuMount.Store(), "craft", ui.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: itemCounts,
	})
	state.menuMount.Update()

	// 初期状態
	tabIndex, _ := ui.GetState[int](state.menuMount, "craft_tabIndex")
	assert.Equal(t, 0, tabIndex, "初期タブインデックスは0")

	// 右に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuRight)
	tabIndex, _ = ui.GetState[int](state.menuMount, "craft_tabIndex")
	assert.Equal(t, 1, tabIndex, "右移動後は1")

	// さらに右に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuRight)
	tabIndex, _ = ui.GetState[int](state.menuMount, "craft_tabIndex")
	assert.Equal(t, 2, tabIndex, "右移動後は2")

	// 循環して戻る
	state.menuMount.Dispatch(inputmapper.ActionMenuRight)
	tabIndex, _ = ui.GetState[int](state.menuMount, "craft_tabIndex")
	assert.Equal(t, 0, tabIndex, "循環して0に戻る")
}

func TestCraftMenuState_DoAction_Cancel(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "キャンセルでTransPop")
}

func TestCraftMenuState_DoAction_CloseMenu(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionCloseMenu)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "CloseMenuでTransPop")
}

func TestCraftMenuState_DoAction_Navigation(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
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

func TestCraftMenuState_WindowProps(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 初期状態ではメニューモード
	assert.Equal(t, craftSubStateMenu, state.subState, "初期状態ではメニューモード")
}

func TestCraftMenuState_DoAction_WindowMode(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// ウィンドウを開く
	state.subState = craftSubStateWindow
	state.windowMount.SetProps(craftWindowProps{
		RecipeName: "テストレシピ",
	})

	// ウィンドウモードでのキャンセル
	transition, err := state.DoAction(world, inputmapper.ActionWindowCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type, "ウィンドウキャンセルはTransNone")

	// ウィンドウが閉じている
	assert.Equal(t, craftSubStateMenu, state.subState, "キャンセル後はメニューモード")
}

func TestCraftMenuState_DoAction_ResultMode(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 結果ウィンドウを開く
	state.subState = craftSubStateResult
	state.resultMount.SetProps(craftResultProps{
		ResultEntity: 1,
	})

	// 結果ウィンドウモードでのキャンセル
	transition, err := state.DoAction(world, inputmapper.ActionWindowCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type, "結果ウィンドウキャンセルはTransNone")

	// 結果ウィンドウが閉じている
	assert.Equal(t, craftSubStateMenu, state.subState, "キャンセル後はメニューモード")
}

func TestCraftMenuState_DoAction_WindowNavigation(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// ウィンドウを開く
	state.subState = craftSubStateWindow
	state.windowMount.SetProps(craftWindowProps{
		RecipeName: "テストレシピ",
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

func TestNewCraftMenuState(t *testing.T) {
	t.Parallel()

	factory := NewCraftMenuState
	state := factory()

	assert.NotNil(t, state, "Stateが作成される")
	_, ok := state.(*CraftMenuState)
	assert.True(t, ok, "CraftMenuState型である")
}

func TestCraftMenuState_GetActionItems(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 空のレシピ名の場合は閉じるのみ
	actions := state.getActionItems(world, "")
	assert.Equal(t, []string{TextClose}, actions, "空のレシピ名は閉じるのみ")
}
