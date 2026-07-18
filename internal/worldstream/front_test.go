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

func TestFrontAt(t *testing.T) {
	t.Parallel()

	cfg := worldstream.FrontConfig{StartEast: -100, ColdWidth: 40, AdvanceTurns: 5, Step: 3}

	// 総ターン数から階段状に前進する
	assert.Equal(t, worldstream.AbsTileX(-100), worldstream.FrontAt(cfg, 0).East, "0ターンは開始位置")
	assert.Equal(t, worldstream.AbsTileX(-100), worldstream.FrontAt(cfg, 4).East, "AdvanceTurns 未満は前進しない")
	assert.Equal(t, worldstream.AbsTileX(-97), worldstream.FrontAt(cfg, 5).East, "5ターンで Step=3 前進")
	assert.Equal(t, worldstream.AbsTileX(-97), worldstream.FrontAt(cfg, 9).East, "階段状: 5〜9ターンは同じ")
	assert.Equal(t, worldstream.AbsTileX(-94), worldstream.FrontAt(cfg, 10).East, "10ターンで 2段め")
	assert.Equal(t, consts.Tile(40), worldstream.FrontAt(cfg, 10).ColdWidth, "ColdWidth は config どおり")

	// 決定的: totalTurns から一意に定まる。42ターン → 42/5=8段 × Step3 = 24 前進
	assert.Equal(t, worldstream.AbsTileX(-76), worldstream.FrontAt(cfg, 42).East, "42ターンは -100+24=-76")

	// 前進しない設定は開始位置に留まる
	still := worldstream.FrontConfig{StartEast: 0, ColdWidth: 10, AdvanceTurns: 0, Step: 3}
	assert.Equal(t, worldstream.AbsTileX(0), worldstream.FrontAt(still, 1000).East, "AdvanceTurns=0 は前進しない")
}
