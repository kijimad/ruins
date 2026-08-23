package steam

import (
	"testing"

	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeSteamLang_対応表で内部コードへ写す(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ja", normalizeSteamLang("japanese"))
	assert.Equal(t, "en", normalizeSteamLang("english"))
	assert.Empty(t, normalizeSteamLang("french"), "対応表に無い Steam 語は空を返す")
}

func TestSteamLangToCode_対応言語すべてに写像がある(t *testing.T) {
	t.Parallel()

	// i18n.SupportedLangs の各コードが steamLangToCode の値に現れることを固定する。
	// 対応言語を増やして steamLangToCode の更新を忘れるドリフトを機械的に弾く。
	codes := make(map[string]bool, len(steamLangToCode))
	for _, code := range steamLangToCode {
		codes[code] = true
	}
	for _, l := range i18n.SupportedLangs() {
		assert.Truef(t, codes[l.Code], "対応言語 %q に対応する Steam 語の写像が steamLangToCode に無い", l.Code)
	}
}
