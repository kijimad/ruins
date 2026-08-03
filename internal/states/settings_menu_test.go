package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &SettingsMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.fetch(world)

	require.Len(t, props.Items, 2)
	assert.Equal(t, "言語", props.Items[0].Label)
	assert.Equal(t, settingsItemLanguage, props.Items[0].Kind)
	assert.Equal(t, "戻る", props.Items[1].Label)
	assert.Equal(t, settingsItemBack, props.Items[1].Kind)
}

func TestNewLanguageMenuState_選択メニューで言語プリセット分の選択肢を持つ(t *testing.T) {
	t.Parallel()

	state, err := NewLanguageMenuState()
	require.NoError(t, err)
	_, ok := state.(*ChoiceMenuState)
	assert.True(t, ok, "言語選択は ChoiceMenu で構成される")

	world := testutil.InitTestWorld(t)
	_, choices := languageChoices(world)
	assert.Len(t, choices, len(languagePresets))
}

func TestCurrentLanguageLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "日本語", currentLanguageLabel("ja"))
	assert.Equal(t, "English", currentLanguageLabel("en"))
	// 一覧に無いコードはそのまま返す
	assert.Equal(t, "fr", currentLanguageLabel("fr"))
}

func TestSettingsMenuState_キャンセルでpopする(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	state := &SettingsMenuState{}
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type)
}
