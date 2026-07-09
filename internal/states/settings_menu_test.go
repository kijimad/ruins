package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSettingsMenu は tabmenu 状態を初期化した SettingsMenuState を返す
func setupSettingsMenu(t *testing.T, world w.World) *SettingsMenuState {
	t.Helper()
	state := &SettingsMenuState{}
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)
	state.menuMount.SetProps(props)
	hooks.UseTabMenu(state.menuMount.Store(), "menu", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})
	state.menuMount.Update()
	return state
}

func TestSettingsMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &SettingsMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetchProps(world)

	require.Len(t, props.Items, 2)
	assert.Equal(t, "言語", props.Items[0].Label)
	assert.Equal(t, settingsItemLanguage, props.Items[0].Kind)
	assert.Equal(t, "戻る", props.Items[1].Label)
	assert.Equal(t, settingsItemBack, props.Items[1].Kind)
}

func TestLanguageMenuState_選択肢が言語プリセット分ある(t *testing.T) {
	t.Parallel()

	state, err := NewLanguageMenuState()
	require.NoError(t, err)
	require.NotNil(t, state)

	ms, ok := state.(*MessageState)
	require.True(t, ok)
	assert.Len(t, ms.messageData.Choices, len(languagePresets))
}

func TestCurrentLanguageLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "日本語", currentLanguageLabel("ja"))
	assert.Equal(t, "English", currentLanguageLabel("en"))
	// 一覧に無いコードはそのまま返す
	assert.Equal(t, "fr", currentLanguageLabel("fr"))
}

func TestSettingsMenuState_言語項目の選択でモーダルをpushする(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	state := setupSettingsMenu(t, world)

	// 初期カーソルは言語項目（index 0）
	transition, err := state.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransPush, transition.Type)
	require.Len(t, transition.NewStateFuncs, 1)
}

func TestSettingsMenuState_戻る項目の選択でpopする(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	state := setupSettingsMenu(t, world)

	// 戻る項目（index 1）へ移動して選択
	state.menuMount.Dispatch(inputmapper.ActionMenuDown)
	state.menuMount.Update()

	transition, err := state.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type)
}

func TestSettingsMenuState_キャンセルでpopする(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	state := setupSettingsMenu(t, world)

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type)
}
