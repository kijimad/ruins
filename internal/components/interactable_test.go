package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteractionKind_Config は各種類のConfigが正しいことを確認
func TestInteractionKind_Config(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind      InteractionKind
		wantRange ActivationRange
		wantWay   ActivationWay
	}{
		{InteractionDoor, ActivationRangeAdjacent, ActivationWayOnCollision},
		{InteractionTalk, ActivationRangeAdjacent, ActivationWayOnCollision},
		{InteractionMelee, ActivationRangeAdjacent, ActivationWayOnCollision},
		{InteractionItem, ActivationRangeSameTile, ActivationWayManual},
		{InteractionItemAll, ActivationRangeSameTile, ActivationWayManual},
		{InteractionPortalNext, ActivationRangeSameTile, ActivationWayManual},
		{InteractionDungeonGate, ActivationRangeSameTile, ActivationWayManual},
		{InteractionStorage, ActivationRangeAdjacent, ActivationWayManual},
		{InteractionDoorLock, ActivationRangeSameTile, ActivationWayAuto},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			t.Parallel()
			config := tt.kind.Config()
			assert.Equal(t, tt.wantRange, config.ActivationRange)
			assert.Equal(t, tt.wantWay, config.ActivationWay)
		})
	}
}

// TestInteractionKind_Config_Unknown は未知の種類がゼロ値（無効）のConfigを返すことを確認
func TestInteractionKind_Config_Unknown(t *testing.T) {
	t.Parallel()

	config := InteractionKind("UNKNOWN").Config()
	require.Error(t, config.ActivationRange.Valid(), "未知の種類は無効なConfigを返す")
	require.Error(t, config.ActivationWay.Valid(), "未知の種類は無効なConfigを返す")
}

// TestActivationRange_Valid は有効なActivationRangeの検証
func TestActivationRange_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		activationRange ActivationRange
		expectValid     bool
	}{
		{
			name:            "SameTile は有効",
			activationRange: ActivationRangeSameTile,
			expectValid:     true,
		},
		{
			name:            "Adjacent は有効",
			activationRange: ActivationRangeAdjacent,
			expectValid:     true,
		},
		{
			name:            "空文字列は無効",
			activationRange: ActivationRange(""),
			expectValid:     false,
		},
		{
			name:            "未定義の値は無効",
			activationRange: ActivationRange("INVALID_RANGE"),
			expectValid:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.activationRange.Valid()
			if tt.expectValid {
				assert.NoError(t, err, "Valid()はエラーを返さないべき")
			} else {
				assert.Error(t, err, "Valid()はエラーを返すべき")
			}
		})
	}
}

// TestActivationWay_Valid は有効なActivationWayの検証
func TestActivationWay_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		activationWay ActivationWay
		expectValid   bool
	}{
		{
			name:          "Auto は有効",
			activationWay: ActivationWayAuto,
			expectValid:   true,
		},
		{
			name:          "Manual は有効",
			activationWay: ActivationWayManual,
			expectValid:   true,
		},
		{
			name:          "OnCollision は有効",
			activationWay: ActivationWayOnCollision,
			expectValid:   true,
		},
		{
			name:          "空文字列は無効",
			activationWay: ActivationWay(""),
			expectValid:   false,
		},
		{
			name:          "未定義の値は無効",
			activationWay: ActivationWay("INVALID_MODE"),
			expectValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.activationWay.Valid()
			if tt.expectValid {
				assert.NoError(t, err, "Valid()はエラーを返さないべき")
			} else {
				assert.Error(t, err, "Valid()はエラーを返すべき")
			}
		})
	}
}

// TestInteractionKind_ConfigConsistency は既知の全種類のConfigが有効な値を返すことを確認
func TestInteractionKind_ConfigConsistency(t *testing.T) {
	t.Parallel()

	kinds := []InteractionKind{
		InteractionPortalNext, InteractionPortalPrev, InteractionDungeonGate, InteractionDoor, InteractionDoorLock,
		InteractionTalk, InteractionItem, InteractionItemAll, InteractionStorage, InteractionMelee,
	}

	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			config := kind.Config()
			require.NoError(t, config.ActivationRange.Valid(), "%s のActivationRangeは有効でなければならない", kind)
			require.NoError(t, config.ActivationWay.Valid(), "%s のActivationWayは有効でなければならない", kind)
		})
	}
}
