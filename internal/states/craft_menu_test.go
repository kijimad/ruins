package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCraftMenuState_OnStart(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)
	assert.False(t, state.detail.Active(), "初期状態で詳細モーダルは閉じている")
	assert.False(t, state.result.Active(), "初期状態で結果モーダルは閉じている")
}

func TestCraftMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetch(world)

	assert.Len(t, props.Tabs, 3, "タブは3つ（道具、武器、防具）")
	assert.Equal(t, "consumables", props.Tabs[0].ID)
	assert.Equal(t, "weapons", props.Tabs[1].ID)
	assert.Equal(t, "wearables", props.Tabs[2].ID)
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
		inputmapper.ActionMenuTabNext,
		inputmapper.ActionMenuTabPrev,
	}

	for _, action := range actions {
		transition, err := state.DoAction(world, action)
		require.NoError(t, err)
		assert.Equal(t, es.TransNone, transition.Type, "ナビゲーションはTransNone: %s", action)
	}
}

func TestCraftMenuState_DoAction_MenuSelectで合成を試みる(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 選択で即合成を試みる。合成できるレシピが無ければ何もせず結果モーダルも開かない
	transition, err := state.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type, "選択はTransNone")
	assert.False(t, state.result.Active(), "合成できなければ結果モーダルは開かない")
}

func TestCraftMenuState_detailContent_選択なしは表示しない(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	_, ok := state.detailContent(world)
	assert.False(t, ok, "レシピ未選択では詳細モーダルを出さない")
}

func TestCraftMenuState_resultDetailContent_合成前は表示しない(t *testing.T) {
	t.Parallel()

	state := &CraftMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	_, ok := state.resultDetailContent(world)
	assert.False(t, ok, "合成前は結果モーダルを出さない")
}

func TestNewCraftMenuState(t *testing.T) {
	t.Parallel()

	factory := NewCraftMenuState
	state, err := factory()
	require.NoError(t, err)
	assert.NotNil(t, state, "Stateが作成される")
	_, ok := state.(*CraftMenuState)
	assert.True(t, ok, "CraftMenuState型である")
}
