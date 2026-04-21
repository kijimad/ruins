package mapplanner

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsConnectivityError は接続性エラー判定をテストする
func TestIsConnectivityError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil エラー",
			err:      nil,
			expected: false,
		},
		{
			name:     "接続性エラー（直接）",
			err:      ErrConnectivity,
			expected: true,
		},
		{
			name:     "接続性エラー（計画検証エラー経由）",
			err:      fmt.Errorf("計画検証エラー: %w", ErrConnectivity),
			expected: true,
		},
		{
			name:     "プレイヤー配置エラー（直接）",
			err:      ErrPlayerPlacement,
			expected: true,
		},
		{
			name:     "プレイヤー配置エラー（計画検証エラー経由）",
			err:      fmt.Errorf("計画検証エラー: %w", ErrPlayerPlacement),
			expected: true,
		},
		{
			name:     "その他のエラー",
			err:      fmt.Errorf("MetaPlan構築エラー: 何らかの問題"),
			expected: false,
		},
		{
			name:     "計画検証エラー",
			err:      fmt.Errorf("計画検証エラー: 何らかの検証失敗"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isConnectivityError(tc.err)
			assert.Equal(t, tc.expected, result)
		})
	}
}
