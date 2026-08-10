package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCatalog_Translate_ja訳を引く(t *testing.T) {
	t.Parallel()
	c := NewCatalog()

	cases := map[string]string{
		"Start":    "開始",
		"Demo":     "デモ",
		"Load":     "読込",
		"Settings": "設定",
		"Quit":     "終了",
	}
	for msgid, want := range cases {
		assert.Equal(t, want, c.Translate("ja", msgid), "ja.po の訳を返す: %s", msgid)
	}
}

func TestCatalog_Translate_未訳は原文へフォールバックする(t *testing.T) {
	t.Parallel()
	c := NewCatalog()

	assert.Equal(t, "Unknown Label", c.Translate("ja", "Unknown Label"), "ja.po に無い msgid は原文をそのまま返す")
}

func TestCatalog_Translate_enは原文をそのまま返す(t *testing.T) {
	t.Parallel()
	c := NewCatalog()

	// 英語は原文が msgid なので PO を持たず、常に原文が返る
	assert.Equal(t, "Start", c.Translate("en", "Start"))
	assert.Equal(t, "Settings", c.Translate("en", "Settings"))
}

func TestCatalog_Translate_未知の言語は原文へフォールバックする(t *testing.T) {
	t.Parallel()
	c := NewCatalog()

	// カタログに無い言語は既定言語 en へフォールバックし、原文を返す
	assert.Equal(t, "Start", c.Translate("fr", "Start"))
	assert.Equal(t, "開始", c.Translate("ja", "Start"), "他言語の引きに影響しない")
}

func TestIsSupportedLang_カタログの言語と過不足なく一致する(t *testing.T) {
	t.Parallel()
	c := NewCatalog()

	// カタログが持つ言語は全て対応言語として検証を通る
	for code := range c.langs {
		assert.True(t, IsSupportedLang(code), "カタログの言語 %q は対応言語のはず", code)
	}
	// 対応言語は全てカタログに存在する。両者の drift をこのテストで止める
	for code := range supportedLangs {
		_, ok := c.langs[code]
		assert.True(t, ok, "対応言語 %q はカタログに存在するはず", code)
	}
	// 未対応の言語は false を返す
	assert.False(t, IsSupportedLang("zh"), "未対応の言語コードは false")
	assert.False(t, IsSupportedLang(""), "空文字は false")
}
