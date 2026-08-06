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

func TestSettingsMenuState_FetchProps(t *testing.T) {
	t.Parallel()

	state := &SettingsMenuState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	props := state.Fetch(world)

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

func TestLanguageChoices_選択で実行中シングルトンと設定を更新する(t *testing.T) {
	// SaveUserConfig の書き込み先を一時ディレクトリへ隔離する。t.Setenv があるので Parallel にしない
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	world := testutil.InitTestWorld(t)
	require.Equal(t, "ja", query.GetUserSettings(world).Language, "既定は ja")

	_, choices := languageChoices(world)
	// languagePresets の並びは ja, en。en を選ぶ
	require.Len(t, choices, 2)
	transition, err := choices[1].Run(world)
	require.NoError(t, err)

	// 実行中のシングルトンが即時に切り替わる。これが再起動なしで表示が変わる経路
	assert.Equal(t, "en", query.GetUserSettings(world).Language, "シングルトンが en へ切り替わる")
	// 永続層の設定値も更新される
	assert.Equal(t, "en", world.Config.User.Language, "config も en へ更新される")
	assert.Equal(t, es.TransPop, transition.Type, "選択後は設定画面へ戻る")
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
