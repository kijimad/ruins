package worldstream_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/stretchr/testify/assert"
)

func TestFront_ColdZoneWest(t *testing.T) {
	t.Parallel()

	f := worldstream.Front{East: 500, ColdWidth: 30}
	assert.Equal(t, worldstream.AbsTileX(470), f.ColdZoneWest(), "西端 = East - ColdWidth")
}

func TestFront_InColdZone(t *testing.T) {
	t.Parallel()

	f := worldstream.Front{East: 500, ColdWidth: 30} // ゾーンは (470, 500]

	assert.False(t, f.InColdZone(470), "西端は含まない（進入不可ライン）")
	assert.True(t, f.InColdZone(471), "西端の東は極寒")
	assert.True(t, f.InColdZone(490), "ゾーン内")
	assert.True(t, f.InColdZone(500), "東端は含む")
	assert.False(t, f.InColdZone(501), "前線より東は平常")
	assert.False(t, f.InColdZone(400), "はるか西（破棄済み側）は極寒ゾーン外")
}

func TestFront_IsWestOfFront(t *testing.T) {
	t.Parallel()

	f := worldstream.Front{East: 500, ColdWidth: 30} // 破棄ライン = 470

	assert.True(t, f.IsWestOfFront(470), "西端ちょうどは破棄側（進入不可）")
	assert.True(t, f.IsWestOfFront(400), "西端より西は破棄側")
	assert.False(t, f.IsWestOfFront(480), "極寒ゾーン内は破棄側でない")
	assert.False(t, f.IsWestOfFront(600), "前線より東は破棄側でない")
}

func TestFront_Advance(t *testing.T) {
	t.Parallel()

	f := worldstream.Front{East: 500, ColdWidth: 30}
	f2 := f.Advance(10)

	assert.Equal(t, worldstream.AbsTileX(510), f2.East, "East が東進する")
	assert.Equal(t, consts.Tile(30), f2.ColdWidth, "幅は不変")
	assert.Equal(t, worldstream.AbsTileX(500), f.East, "元の値は不変（値型）")
}
