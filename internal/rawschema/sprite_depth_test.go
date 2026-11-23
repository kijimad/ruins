package rawschema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpriteDepth_Rank(t *testing.T) {
	t.Parallel()

	ranks := make([]int, 0, len(SpriteDepthValues()))
	for _, d := range SpriteDepthValues() {
		ranks = append(ranks, d.Rank())
	}
	assert.ElementsMatch(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}, ranks, "生成された全ての段が depthOrder に1つずつある")
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}, []int{
		SpriteDepthFloor.Rank(), SpriteDepthSnow.Rank(), SpriteDepthRemains.Rank(), SpriteDepthDecal.Rank(),
		SpriteDepthRug.Rank(), SpriteDepthShadow.Rank(), SpriteDepthTaller.Rank(), SpriteDepthItem.Rank(), SpriteDepthPlayer.Rank(),
	}, "床から手前へ並ぶ")
	assert.Equal(t, SpriteDepthFloor.Rank(), SpriteDepth("").Rank(), "段を持たない行は床の順にする")
}
