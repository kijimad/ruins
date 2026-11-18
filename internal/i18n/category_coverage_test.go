package i18n_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// TestCategoryTranslationCoverage は分類の名前が全て ja.po に訳を持つことを確かめる。見回しは分類の名前を
// 変数で query.T に渡すので、文字列リテラルの msgid を集める検査では訳の漏れが見つからない
func TestCategoryTranslationCoverage(t *testing.T) {
	t.Parallel()

	defined := parsePoMsgids(t)
	world := testutil.InitTestWorld(t)
	for key, cats := range world.Components.Categories() {
		for _, cat := range cats {
			assert.True(t, defined[cat.Name], "分類 %s の名前 %q に ja.po の訳が無い", key, cat.Name)
		}
	}
}
