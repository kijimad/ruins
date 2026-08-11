package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainMenuState_OnStart(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)

	require.NoError(t, state.OnStart(world))
}

func TestMainMenuState_項目と遷移の対応(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.Fetch(world)

	require.Len(t, props.Items, 5, "メニュー項目は5つ")
	assert.Equal(t, "Start", props.Items[0].Label)
	assert.Equal(t, es.TransReplace, props.Items[0].Transition.Type, "開始は Replace")
	assert.Equal(t, "Demo", props.Items[1].Label)
	assert.Equal(t, es.TransReplace, props.Items[1].Transition.Type, "デモは Replace")
	assert.Equal(t, "Load", props.Items[2].Label)
	assert.Equal(t, es.TransPush, props.Items[2].Transition.Type, "読込は Push")
	assert.Equal(t, "Settings", props.Items[3].Label)
	assert.Equal(t, es.TransPush, props.Items[3].Transition.Type, "設定は Push")
	assert.Equal(t, "Quit", props.Items[4].Label)
	assert.Equal(t, es.TransQuit, props.Items[4].Transition.Type, "終了は Quit")
}

func TestMainMenuState_言語切替でラベルが変わる(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 既定 en では原文の英語ラベルを返す
	assert.Equal(t, "Start", state.Fetch(world).Items[0].Label, "既定 en は英語原文")

	// UserSettings の言語を ja へ書き換えると日本語ラベルになる。query.T が ja.po を引く経路
	query.GetUserSettings(world).Language = "ja"
	ja := state.Fetch(world)
	assert.Equal(t, "開始", ja.Items[0].Label, "ja は日本語")
	assert.Equal(t, "設定", ja.Items[3].Label, "ja は日本語")
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

func TestMainMenuState_DoAction_Navigation(t *testing.T) {
	t.Parallel()

	state := &MainMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	for _, action := range []inputmapper.ActionID{
		inputmapper.ActionMenuUp,
		inputmapper.ActionMenuDown,
		inputmapper.ActionMenuLeft,
		inputmapper.ActionMenuRight,
		inputmapper.ActionMenuTabNext,
		inputmapper.ActionMenuTabPrev,
	} {
		transition, err := state.DoAction(world, action)
		require.NoError(t, err)
		assert.Equal(t, es.TransNone, transition.Type, "ナビゲーションはTransNone: %s", action)
	}
}

func TestNewMainMenuState(t *testing.T) {
	t.Parallel()

	state, err := NewMainMenuState()
	require.NoError(t, err)
	_, ok := state.(*MainMenuState)
	assert.True(t, ok, "MainMenuState型である")
}
