package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

// ch はテストのチャンク座標を短く書くヘルパー。
func ch(x, y consts.Chunk) consts.Coord[consts.Chunk] {
	return consts.Coord[consts.Chunk]{X: x, Y: y}
}

func TestMarkRoadLShape_水平の直線は東西ビットを積む(t *testing.T) {
	t.Parallel()
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	markRoadLShape(overlay, ch(0, 0), ch(3, 0))

	assert.Equal(t, RoadE, overlay[ch(0, 0)], "西端は東の隣だけに繋がる")
	assert.Equal(t, RoadW|RoadE, overlay[ch(1, 0)], "途中は両隣に繋がる")
	assert.Equal(t, RoadW|RoadE, overlay[ch(2, 0)], "途中は両隣に繋がる")
	assert.Equal(t, RoadW, overlay[ch(3, 0)], "東端は西の隣だけに繋がる")
}

func TestMarkRoadLShape_角は水平と垂直のビットが同一チャンクに積まれる(t *testing.T) {
	t.Parallel()
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	// a=(0,0) から b=(2,3)。水平辺は行0を x0..2、垂直辺は列2を y0..3、角は (2,0)
	markRoadLShape(overlay, ch(0, 0), ch(2, 3))

	assert.Equal(t, RoadE, overlay[ch(0, 0)], "水平の始点")
	assert.Equal(t, RoadW|RoadE, overlay[ch(1, 0)], "水平の途中")
	assert.Equal(t, RoadW|RoadS, overlay[ch(2, 0)], "角は西と南、L 字の折れ")
	assert.Equal(t, RoadN|RoadS, overlay[ch(2, 1)], "垂直の途中")
	assert.Equal(t, RoadN|RoadS, overlay[ch(2, 2)], "垂直の途中")
	assert.Equal(t, RoadN, overlay[ch(2, 3)], "垂直の終点")
}

func TestMarkRoadLShape_交差は四方向ビットになる(t *testing.T) {
	t.Parallel()
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	// 行1を横切る水平路と、列1を貫く垂直路を (1,1) で交差させる
	markRoadLShape(overlay, ch(0, 1), ch(2, 1))
	markRoadLShape(overlay, ch(1, 0), ch(1, 3))

	assert.Equal(t, RoadN|RoadS|RoadE|RoadW, overlay[ch(1, 1)], "交差は四方向すべてに繋がる")
}

func TestMarkRoadLShape_同一チャンクは道を積まない(t *testing.T) {
	t.Parallel()
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	// 始点と終点が同じなら水平も垂直もステップが無く、ループは即終了して無限ループしない
	markRoadLShape(overlay, ch(2, 2), ch(2, 2))

	assert.Empty(t, overlay, "動かない経路は何も積まない")
}

func TestBuildRoadOverlay_窓の範囲に道が現れ端点が集落に一致する(t *testing.T) {
	t.Parallel()
	const runSeed uint64 = 12345
	const rows consts.Chunk = 9
	// 集落は Spacing チャンクごとに1つ当たる。数リージョン跨ぐ窓なら必ず道が出る
	win := MacroWindow{OriginX: 0, Cols: 3 * settlementPlacement.Spacing, Rows: rows}
	overlay := buildRoadOverlay(runSeed, win, rows)

	assert.NotEmpty(t, overlay, "複数リージョンを覆う窓には道が出る")
	for c, dir := range overlay {
		assert.NotZero(t, dir, "道ありのキーはビットが立つ")
		assert.Zero(t, dir&^(RoadN|RoadS|RoadE|RoadW), "未定義ビットは立たない")
		assert.GreaterOrEqual(t, c.Y, consts.Chunk(0), "行は帯の内側")
		assert.Less(t, c.Y, rows, "行は帯の内側")
	}

	// 当選集落の当該チャンクは端点として道に含まれる。隣接リージョンを結ぶので道の端に来る
	a := settlementPlacement.WinnerOf(runSeed, 1, rows)
	assert.NotZero(t, overlay[a], "当選集落チャンクには街道が届く")
}
