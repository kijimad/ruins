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

func TestNextLanguage(t *testing.T) {
	t.Parallel()

	// プリセットは {ja, 日本語}, {en, English} の2つを前提とする
	cases := []struct {
		name     string
		code     string
		dir      int
		wantCode string
	}{
		{"jaから次へ", "ja", 1, "en"},
		{"enから次へは循環してja", "en", 1, "ja"},
		{"jaから前へは循環してen", "ja", -1, "en"},
		{"一覧に無いコードは先頭を起点に次へ", "fr", 1, "en"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.wantCode, nextLanguage(tc.code, tc.dir).Code)
		})
	}
}

func TestCurrentLanguageLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "日本語", currentLanguageLabel("ja"))
	assert.Equal(t, "English", currentLanguageLabel("en"))
	// 一覧に無いコードはそのまま返す
	assert.Equal(t, "fr", currentLanguageLabel("fr"))
}
