package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	t.Parallel()
	// 定数の値をテスト
	assert.Equal(t, 960, GameWidth, "GameWidthの値が正しくない")
	assert.Equal(t, 720, GameHeight, "GameHeightの値が正しくない")
	assert.Equal(t, 32, int(TileSize), "TileSizeの値が正しくない")
}
