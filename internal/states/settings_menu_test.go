package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/i18n"
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

	props, err := state.Fetch(world)
	require.NoError(t, err)

	require.Len(t, props.Items, 2)
	assert.Equal(t, "Language", props.Items[0].Label)
	assert.Equal(t, settingsItemLanguage, props.Items[0].Kind)
	// 現在言語の表示は UserSettings 由来にする。既定 en では英語表示
	assert.Equal(t, "English", props.Items[0].Value)
	assert.Equal(t, "Back", props.Items[1].Label)
	assert.Equal(t, settingsItemBack, props.Items[1].Kind)

	// UserSettings を ja へ切り替えると表示も追従する。
	// 期待値は ja 訳を i18n から導出し、ja.po の訳文更新でこのテストが drift しないようにする。
	// このテストの関心は表示が UserSettings 言語に追従することで、訳文の正当性は i18n の責務。
	query.GetUserSettings(world).Language = "ja"
	wantJa := world.Resources.I18N.Translate("ja", "Japanese")
	jaProps, err := state.Fetch(world)
	require.NoError(t, err)
	assert.Equal(t, wantJa, jaProps.Items[0].Value, "表示は config でなく UserSettings を引く")
}

func TestNewLanguageMenuState_選択メニューで言語プリセット分の選択肢を持つ(t *testing.T) {
	t.Parallel()

	state, err := NewLanguageMenuState()
	require.NoError(t, err)
	_, ok := state.(*ChoiceMenuState)
	assert.True(t, ok, "言語選択は ChoiceMenu で構成される")

	world := testutil.InitTestWorld(t)
	_, choices := languageChoices(world)
	assert.Len(t, choices, len(i18n.SupportedLangs()))
}

func TestLanguageChoices_選択で実行中シングルトンと設定を更新する(t *testing.T) {
	// SaveUserConfig の書き込み先を一時ディレクトリへ隔離する。t.Setenv があるので Parallel にしない
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	world := testutil.InitTestWorld(t)
	require.Equal(t, "en", query.GetUserSettings(world).Language, "既定は en")

	_, choices := languageChoices(world)
	// i18n.SupportedLangs() の並びは ja, en。ja を選んで切り替えを確かめる
	require.Len(t, choices, 2)
	transition, err := choices[0].Run(world)
	require.NoError(t, err)

	// 実行中のシングルトンが即時に切り替わる。これが再起動なしで表示が変わる経路
	assert.Equal(t, "ja", query.GetUserSettings(world).Language, "シングルトンが ja へ切り替わる")
	// 永続層の設定値も更新される
	assert.Equal(t, "ja", world.Config.User.Language, "config も ja へ更新される")
	assert.Equal(t, es.TransPop, transition.Type, "選択後は設定画面へ戻る")
}

func TestCurrentLanguageLabel(t *testing.T) {
	t.Parallel()

	// 表示名の msgid を返す。実際の訳は query.T が引く
	assert.Equal(t, "Japanese", currentLanguageLabel("ja"))
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
