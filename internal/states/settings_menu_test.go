package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
