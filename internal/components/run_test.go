package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeathCause_DisplayName(t *testing.T) {
	t.Parallel()

	// 既知の死因はすべて表に登録されている。表への登録漏れを検知する。
	// 表示名の文字列そのものはデータなので照合しない
	for _, c := range []DeathCause{CauseFrozen, CauseIllness, CauseBloodLoss, CauseKilled, CauseDebug} {
		_, ok := deathCauseDisplayNames[c]
		assert.True(t, ok, "%s が表に登録されている", c)
	}

	// 未登録の死因は素のIDへ落とす。未信頼な旧セーブ値を受ける経路
	assert.Equal(t, "unknown", DeathCause("unknown").DisplayName())
}
