package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeathCause_DisplayName(t *testing.T) {
	t.Parallel()

	// 登録済みの死因は表示名を返す。死因を足して表示名を忘れると素のIDが返り、全文一致で露見する。
	// 公開 API の DisplayName だけを見て、非公開の表には触れない
	for _, tt := range []struct {
		cause DeathCause
		want  string
	}{
		{CauseFrozen, "froze to death"},
		{CauseIllness, "died of illness"},
		{CauseBloodLoss, "bled out"},
		{CauseKilled, "killed in battle"},
		{CauseDebug, "debug"},
	} {
		assert.Equal(t, tt.want, tt.cause.DisplayName(), "%s の表示名", tt.cause)
	}

	// 未登録の死因は素のIDへ落とす。未信頼な旧セーブ値を受ける経路
	assert.Equal(t, "unknown", DeathCause("unknown").DisplayName())
}
