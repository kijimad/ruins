package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

// TestGetWorseningMessage は低体温の悪化メッセージを重症度ごとに固定する。
// 低体温以外や重症度なしは空文字になる分岐も検証する。
func TestGetWorseningMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		condType gc.ConditionType
		severity gc.Severity
		want     string
	}{
		{"低体温なしは空", gc.ConditionHypothermia, gc.SeverityNone, ""},
		{"低体温軽", gc.ConditionHypothermia, gc.SeverityMinor, "The cold is setting in"},
		{"低体温中", gc.ConditionHypothermia, gc.SeverityMedium, "You are quite cold"},
		{"低体温重", gc.ConditionHypothermia, gc.SeveritySevere, "The cold is dangerous"},
		{"低体温以外は空", gc.ConditionFracture, gc.SeveritySevere, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, getWorseningMessage(tt.condType, tt.severity))
		})
	}
}

// TestGetRecoveryMessage は低体温の回復メッセージを重症度ごとに固定する。
// 重症のまま回復した場合や低体温以外は空文字になる分岐も検証する。
func TestGetRecoveryMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		condType gc.ConditionType
		severity gc.Severity
		want     string
	}{
		{"低体温なしまで回復", gc.ConditionHypothermia, gc.SeverityNone, "You have warmed up"},
		{"低体温軽", gc.ConditionHypothermia, gc.SeverityMinor, "You are warming up a little"},
		{"低体温中", gc.ConditionHypothermia, gc.SeverityMedium, "Still cold, but a little better"},
		{"低体温重は空", gc.ConditionHypothermia, gc.SeveritySevere, ""},
		{"低体温以外は空", gc.ConditionFracture, gc.SeverityMinor, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, getRecoveryMessage(tt.condType, tt.severity))
		})
	}
}
