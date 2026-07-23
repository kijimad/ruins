package consts_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestChunk_Tiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		c         consts.Chunk
		chunkSize consts.Tile
		want      consts.Tile
	}{
		{"1チャンク", 1, 16, 16},
		{"複数チャンク", 3, 16, 48},
		{"ゼロチャンク", 0, 16, 0},
		{"チャンクサイズが異なる場合", 3, 2, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.c.Tiles(tt.chunkSize))
		})
	}
}
