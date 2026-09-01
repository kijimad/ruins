package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeathCause_DisplayName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "froze to death", CauseFrozen.DisplayName())
	assert.Equal(t, "died of illness", CauseIllness.DisplayName())
	assert.Equal(t, "bled out", CauseBloodLoss.DisplayName())
	assert.Equal(t, "killed in battle", CauseKilled.DisplayName())
	assert.Equal(t, "debug", CauseDebug.DisplayName())
	assert.Equal(t, "unknown", DeathCause("unknown").DisplayName(), "未知は素の文字列へ落とす")
}
