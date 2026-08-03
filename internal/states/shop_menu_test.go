package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
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
	assert.False(t, state.actionWin.Active(), "初期状態でアクション窓は閉じている")
}

func TestShopMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetch(world)

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

func TestShopMenuState_DoAction_MenuSelectでアクション窓を開く(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type, "選択はTransNone")
	assert.True(t, state.actionWin.Active(), "アクション選択ウィンドウが開く")
}

func TestShopMenuState_actionWindowContent_選択なしは表示しない(t *testing.T) {
	t.Parallel()

	state := &ShopMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	_, _, ok := state.actionWindowContent(world)
	assert.False(t, ok, "商品未選択ではアクション窓を出さない")
}

func TestNewShopMenuState(t *testing.T) {
	t.Parallel()

	factory := NewShopMenuState
	state, err := factory()
	require.NoError(t, err)
	assert.NotNil(t, state, "Stateが作成される")
	_, ok := state.(*ShopMenuState)
	assert.True(t, ok, "ShopMenuState型である")
}
