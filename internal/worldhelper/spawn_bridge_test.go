package worldhelper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateBridgeSeed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		baseDepth int
		gameSeed  uint64
		bridgeID  string
		expected  uint64
	}{
		{
			name:      "橋A: シード計算",
			baseDepth: 1,
			gameSeed:  12345,
			bridgeID:  "A",
			expected:  13346, // 1 + 12345 + 1000
		},
		{
			name:      "橋B: シード計算",
			baseDepth: 1,
			gameSeed:  12345,
			bridgeID:  "B",
			expected:  14346, // 1 + 12345 + 2000
		},
		{
			name:      "橋C: シード計算",
			baseDepth: 1,
			gameSeed:  12345,
			bridgeID:  "C",
			expected:  15346, // 1 + 12345 + 3000
		},
		{
			name:      "橋D: シード計算",
			baseDepth: 1,
			gameSeed:  12345,
			bridgeID:  "D",
			expected:  16346, // 1 + 12345 + 4000
		},
		{
			name:      "深い階層での計算",
			baseDepth: 10,
			gameSeed:  12345,
			bridgeID:  "A",
			expected:  13355, // 10 + 12345 + 1000
		},
		{
			name:      "未知の橋ID",
			baseDepth: 1,
			gameSeed:  12345,
			bridgeID:  "Z",
			expected:  22345, // 1 + 12345 + 9999
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := CalculateBridgeSeed(tt.baseDepth, tt.gameSeed, tt.bridgeID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateBridgeSeed_UniquenessPerBridge(t *testing.T) {
	t.Parallel()

	baseDepth := 1
	gameSeed := uint64(12345)

	seedA := CalculateBridgeSeed(baseDepth, gameSeed, "A")
	seedB := CalculateBridgeSeed(baseDepth, gameSeed, "B")
	seedC := CalculateBridgeSeed(baseDepth, gameSeed, "C")
	seedD := CalculateBridgeSeed(baseDepth, gameSeed, "D")

	// 全ての橋のシードが異なることを確認
	require.NotEqual(t, seedA, seedB, "橋AとBのシードは異なるべき")
	require.NotEqual(t, seedA, seedC, "橋AとCのシードは異なるべき")
	require.NotEqual(t, seedA, seedD, "橋AとDのシードは異なるべき")
	require.NotEqual(t, seedB, seedC, "橋BとCのシードは異なるべき")
	require.NotEqual(t, seedB, seedD, "橋BとDのシードは異なるべき")
	require.NotEqual(t, seedC, seedD, "橋CとDのシードは異なるべき")
}
