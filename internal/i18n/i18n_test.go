package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslator_T_ja訳を引く(t *testing.T) {
	t.Parallel()
	tr, err := New("ja")
	require.NoError(t, err)

	cases := map[string]string{
		"Start":    "開始",
		"Demo":     "デモ",
		"Load":     "読込",
		"Settings": "設定",
		"Quit":     "終了",
	}
	for msgid, want := range cases {
		assert.Equal(t, want, tr.T(msgid), "ja.po の訳を返す: %s", msgid)
	}
}

func TestTranslator_T_未訳は原文へフォールバックする(t *testing.T) {
	t.Parallel()
	tr, err := New("ja")
	require.NoError(t, err)

	assert.Equal(t, "Unknown Label", tr.T("Unknown Label"), "ja.po に無い msgid は原文をそのまま返す")
}

func TestTranslator_T_enは原文をそのまま返す(t *testing.T) {
	t.Parallel()
	tr, err := New("en")
	require.NoError(t, err)

	// 英語は原文が msgid なので PO を持たず、常に原文が返る
	assert.Equal(t, "Start", tr.T("Start"), "en は原文の英語を返す")
	assert.Equal(t, "Settings", tr.T("Settings"), "en は原文の英語を返す")
}

func TestTranslator_SetLanguage_言語を切り替える(t *testing.T) {
	t.Parallel()
	tr, err := New("ja")
	require.NoError(t, err)
	assert.Equal(t, "開始", tr.T("Start"), "初期は ja")

	require.NoError(t, tr.SetLanguage("en"))
	assert.Equal(t, "en", tr.Language(), "言語コードが切り替わる")
	assert.Equal(t, "Start", tr.T("Start"), "en 切替後は原文を返す")

	require.NoError(t, tr.SetLanguage("ja"))
	assert.Equal(t, "開始", tr.T("Start"), "ja へ戻せる")
}

func TestTranslator_SetLanguage_未対応言語はエラー(t *testing.T) {
	t.Parallel()
	_, err := New("fr")
	require.Error(t, err, "未対応の言語はエラーにする")
}

func TestNewDefault_既定はja(t *testing.T) {
	t.Parallel()
	tr := NewDefault()
	assert.Equal(t, "ja", tr.Language(), "既定言語は ja")
	assert.Equal(t, "開始", tr.T("Start"), "既定で日本語訳を返す")
}

func TestTranslator_独立インスタンスは干渉しない(t *testing.T) {
	t.Parallel()
	ja, err := New("ja")
	require.NoError(t, err)
	en, err := New("en")
	require.NoError(t, err)

	// 一方の言語を切り替えても他方に影響しない。並行テストの独立性の根拠
	require.NoError(t, en.SetLanguage("ja"))
	assert.Equal(t, "開始", ja.T("Start"), "ja インスタンスは不変")
	assert.Equal(t, "開始", en.T("Start"), "en インスタンスだけが切り替わる")
}
