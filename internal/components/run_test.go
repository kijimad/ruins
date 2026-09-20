package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeathCause_DisplayName(t *testing.T) {
	t.Parallel()

	// 登録済みの死因は表示名を返す。死因を足して表示名を忘れると素のIDが返り、全文一致で露見する。
	// 公開 API の DisplayName だけを見て、非公開の表には触れない
	tests := []struct {
		cause DeathCause
		want  string
	}{
		{CauseFrozen, "froze to death"},
		{CauseIllness, "died of illness"},
		{CauseBloodLoss, "bled out"},
		{CauseKilled, "killed in battle"},
		{CauseDebug, "debug"},
	}
	for _, tt := range tests {
		t.Run(string(tt.cause), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.cause.DisplayName())
		})
	}

	// 未登録の死因は素のIDへ落とす。未信頼な旧セーブ値を受ける経路
	assert.Equal(t, "unknown", DeathCause("unknown").DisplayName())
}
