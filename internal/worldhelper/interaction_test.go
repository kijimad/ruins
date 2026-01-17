package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

// TestIsInActivationRange_SameTile は直上タイル判定のテスト
func TestIsInActivationRange_SameTile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		playerX         gc.Tile
		playerY         gc.Tile
		triggerX        gc.Tile
		triggerY        gc.Tile
		expectedInRange bool
	}{
		{
			name:            "同じタイル",
			playerX:         5,
			playerY:         5,
			triggerX:        5,
			triggerY:        5,
			expectedInRange: true,
		},
		{
			name:            "右隣",
			playerX:         5,
			playerY:         5,
			triggerX:        6,
			triggerY:        5,
			expectedInRange: false,
		},
		{
			name:            "左隣",
			playerX:         5,
			playerY:         5,
			triggerX:        4,
			triggerY:        5,
			expectedInRange: false,
		},
		{
			name:            "上",
			playerX:         5,
			playerY:         5,
			triggerX:        5,
			triggerY:        4,
			expectedInRange: false,
		},
		{
			name:            "下",
			playerX:         5,
			playerY:         5,
			triggerX:        5,
			triggerY:        6,
			expectedInRange: false,
		},
		{
			name:            "右上斜め",
			playerX:         5,
			playerY:         5,
			triggerX:        6,
			triggerY:        4,
			expectedInRange: false,
		},
		{
			name:            "遠い位置",
			playerX:         5,
			playerY:         5,
			triggerX:        10,
			triggerY:        10,
			expectedInRange: false,
		},
		{
			name:            "原点(0,0)同士",
			playerX:         0,
			playerY:         0,
			triggerX:        0,
			triggerY:        0,
			expectedInRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			playerGrid := &gc.GridElement{X: tt.playerX, Y: tt.playerY}
			triggerGrid := &gc.GridElement{X: tt.triggerX, Y: tt.triggerY}

			result := IsInActivationRange(playerGrid, triggerGrid, gc.ActivationRangeSameTile)
			assert.Equal(t, tt.expectedInRange, result)
		})
	}
}

// TestIsInActivationRange_Adjacent は隣接タイル判定のテスト（8近傍、同じタイルは除外）
func TestIsInActivationRange_Adjacent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		playerX         gc.Tile
		playerY         gc.Tile
		triggerX        gc.Tile
		triggerY        gc.Tile
		expectedInRange bool
	}{
		{
			name:            "同じタイル（除外されるべき）",
			playerX:         5,
			playerY:         5,
			triggerX:        5,
			triggerY:        5,
			expectedInRange: false, // 重要：Adjacentは同じタイルを含まない
		},
		{
			name:            "右隣",
			playerX:         5,
			playerY:         5,
			triggerX:        6,
			triggerY:        5,
			expectedInRange: true,
		},
		{
			name:            "左隣",
			playerX:         5,
			playerY:         5,
			triggerX:        4,
			triggerY:        5,
			expectedInRange: true,
		},
		{
			name:            "上",
			playerX:         5,
			playerY:         5,
			triggerX:        5,
			triggerY:        4,
			expectedInRange: true,
		},
		{
			name:            "下",
			playerX:         5,
			playerY:         5,
			triggerX:        5,
			triggerY:        6,
			expectedInRange: true,
		},
		{
			name:            "右上斜め",
			playerX:         5,
			playerY:         5,
			triggerX:        6,
			triggerY:        4,
			expectedInRange: true,
		},
		{
			name:            "右下斜め",
			playerX:         5,
			playerY:         5,
			triggerX:        6,
			triggerY:        6,
			expectedInRange: true,
		},
		{
			name:            "左上斜め",
			playerX:         5,
			playerY:         5,
			triggerX:        4,
			triggerY:        4,
			expectedInRange: true,
		},
		{
			name:            "左下斜め",
			playerX:         5,
			playerY:         5,
			triggerX:        4,
			triggerY:        6,
			expectedInRange: true,
		},
		{
			name:            "距離2（右）",
			playerX:         5,
			playerY:         5,
			triggerX:        7,
			triggerY:        5,
			expectedInRange: false,
		},
		{
			name:            "距離2（上）",
			playerX:         5,
			playerY:         5,
			triggerX:        5,
			triggerY:        3,
			expectedInRange: false,
		},
		{
			name:            "距離2（斜め）",
			playerX:         5,
			playerY:         5,
			triggerX:        7,
			triggerY:        7,
			expectedInRange: false,
		},
		{
			name:            "遠い位置",
			playerX:         5,
			playerY:         5,
			triggerX:        10,
			triggerY:        10,
			expectedInRange: false,
		},
		{
			name:            "原点(0,0)から(1,0)",
			playerX:         0,
			playerY:         0,
			triggerX:        1,
			triggerY:        0,
			expectedInRange: true,
		},
		{
			name:            "原点(0,0)から(1,1)",
			playerX:         0,
			playerY:         0,
			triggerX:        1,
			triggerY:        1,
			expectedInRange: true,
		},
		{
			name:            "負の座標でも動作（左上へ移動）",
			playerX:         5,
			playerY:         5,
			triggerX:        4,
			triggerY:        4,
			expectedInRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			playerGrid := &gc.GridElement{X: tt.playerX, Y: tt.playerY}
			triggerGrid := &gc.GridElement{X: tt.triggerX, Y: tt.triggerY}

			result := IsInActivationRange(playerGrid, triggerGrid, gc.ActivationRangeAdjacent)
			assert.Equal(t, tt.expectedInRange, result)
		})
	}
}

// TestIsInActivationRange_InvalidRange は無効な範囲タイプのテスト
func TestIsInActivationRange_InvalidRange(t *testing.T) {
	t.Parallel()

	playerGrid := &gc.GridElement{X: 5, Y: 5}
	triggerGrid := &gc.GridElement{X: 5, Y: 5}

	// 無効な範囲タイプ
	result := IsInActivationRange(playerGrid, triggerGrid, gc.ActivationRange("INVALID"))
	assert.False(t, result)
}

// TestIsInActivationRange_8Neighbors は8近傍が正しくカバーされているかの包括的テスト
func TestIsInActivationRange_8Neighbors(t *testing.T) {
	t.Parallel()

	// プレイヤーを中心(5,5)に配置
	playerGrid := &gc.GridElement{X: 5, Y: 5}

	// 8近傍の全タイル
	neighbors := []gc.GridElement{
		{X: 4, Y: 4}, // 左上
		{X: 5, Y: 4}, // 上
		{X: 6, Y: 4}, // 右上
		{X: 4, Y: 5}, // 左
		{X: 6, Y: 5}, // 右
		{X: 4, Y: 6}, // 左下
		{X: 5, Y: 6}, // 下
		{X: 6, Y: 6}, // 右下
	}

	for _, neighbor := range neighbors {
		result := IsInActivationRange(playerGrid, &neighbor, gc.ActivationRangeAdjacent)
		assert.True(t, result, "タイル(%d,%d)は8近傍に含まれるべき", neighbor.X, neighbor.Y)
	}

	// 中心タイル（含まれない）
	centerResult := IsInActivationRange(playerGrid, playerGrid, gc.ActivationRangeAdjacent)
	assert.False(t, centerResult, "中心タイル(5,5)は8近傍に含まれないべき")
}
