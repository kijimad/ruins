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

func TestMainMenuState_OnStart(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)
	assert.NotNil(t, state.menuMount, "menuMountが初期化されている")
}

func TestMainMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps()

	assert.Equal(t, 3, len(props.Items), "メニュー項目は3つ")
	assert.Equal(t, "開始", props.Items[0].Label)
	assert.Equal(t, "読込", props.Items[1].Label)
	assert.Equal(t, "終了", props.Items[2].Label)
}

func TestMainMenuState_Navigation(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps()
	state.menuMount.SetProps(props)
	hooks.UseTabMenu(state.menuMount.Store(), "menu", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	state.menuMount.Update()

	// 初期状態
	menuState, ok := hooks.GetState[hooks.TabMenuState](state.menuMount, "menu")
	assert.True(t, ok)
	assert.Equal(t, 0, menuState.ItemIndex, "初期インデックスは0")

	// 下に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuDown)
	state.menuMount.Update()
	menuState, _ = hooks.GetState[hooks.TabMenuState](state.menuMount, "menu")
	assert.Equal(t, 1, menuState.ItemIndex, "下移動後はインデックス1")

	// 上に移動
	state.menuMount.Dispatch(inputmapper.ActionMenuUp)
	state.menuMount.Update()
	menuState, _ = hooks.GetState[hooks.TabMenuState](state.menuMount, "menu")
	assert.Equal(t, 0, menuState.ItemIndex, "上移動後はインデックス0")
}

func TestMainMenuState_CircularNavigation(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps()
	state.menuMount.SetProps(props)
	hooks.UseTabMenu(state.menuMount.Store(), "menu", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	state.menuMount.Update()

	// 最初の項目から上に移動すると最後の項目に
	state.menuMount.Dispatch(inputmapper.ActionMenuUp)
	state.menuMount.Update()
	menuState, _ := hooks.GetState[hooks.TabMenuState](state.menuMount, "menu")
	assert.Equal(t, 2, menuState.ItemIndex, "循環して最後の項目に移動")

	// 最後の項目から下に移動すると最初の項目に
	state.menuMount.Dispatch(inputmapper.ActionMenuDown)
	state.menuMount.Update()
	menuState, _ = hooks.GetState[hooks.TabMenuState](state.menuMount, "menu")
	assert.Equal(t, 0, menuState.ItemIndex, "循環して最初の項目に移動")
}

func TestMainMenuState_DoAction_Cancel(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransQuit, transition.Type, "キャンセルでTransQuit")
}

func TestMainMenuState_DoAction_CloseMenu(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionCloseMenu)
	require.NoError(t, err)
	assert.Equal(t, es.TransQuit, transition.Type, "CloseMenuでTransQuit")
}

func TestMainMenuState_DoAction_Navigation(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

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

func TestMainMenuState_Selection_Start(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps()
	state.menuMount.SetProps(props)
	hooks.UseTabMenu(state.menuMount.Store(), "menu", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	state.menuMount.Update()

	// 「開始」を選択（インデックス0）
	transition, err := state.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransReplace, transition.Type, "開始でTransReplace")
}

func TestMainMenuState_Selection_Load(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps()
	state.menuMount.SetProps(props)
	hooks.UseTabMenu(state.menuMount.Store(), "menu", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	state.menuMount.Update()

	// 「読込」に移動して選択（インデックス1）
	state.menuMount.Dispatch(inputmapper.ActionMenuDown)
	state.menuMount.Update()

	transition, err := state.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransPush, transition.Type, "読込でTransPush")
}

func TestMainMenuState_Selection_Exit(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps()
	state.menuMount.SetProps(props)
	hooks.UseTabMenu(state.menuMount.Store(), "menu", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	state.menuMount.Update()

	// 「終了」に移動して選択（インデックス2）
	state.menuMount.Dispatch(inputmapper.ActionMenuDown)
	state.menuMount.Dispatch(inputmapper.ActionMenuDown)
	state.menuMount.Update()

	transition, err := state.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransQuit, transition.Type, "終了でTransQuit")
}

func TestNewMainMenuState(t *testing.T) {
	t.Parallel()

	factory := NewMainMenuState
	state := factory()

	assert.NotNil(t, state, "Stateが作成される")
	_, ok := state.(*MainMenuState)
	assert.True(t, ok, "MainMenuState型である")
}

func TestMainMenuState_HandleInput(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// HandleInputはHandleMenuInputを呼び出す
	_, _ = state.HandleInput(world.Config)
}

func TestMainMenuState_String(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	assert.Equal(t, "MainMenu", state.String())
}
