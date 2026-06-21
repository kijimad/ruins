package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActivityState_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state ActivityState
		want  string
	}{
		{ActivityStateRunning, "Running"},
		{ActivityStatePaused, "Paused"},
		{ActivityStateCompleted, "Completed"},
		{ActivityStateCanceled, "Canceled"},
		{ActivityState(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}
