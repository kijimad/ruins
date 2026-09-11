package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestMarkRoadLShape_水平の直線は東西ビットを積む(t *testing.T) {
	t.Parallel()
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	markRoadLShape(overlay, consts.Coord[consts.Chunk]{X: 0, Y: 0}, consts.Coord[consts.Chunk]{X: 3, Y: 0})

	assert.Equal(t, RoadE, overlay[consts.Coord[consts.Chunk]{X: 0, Y: 0}], "西端は東の隣だけに繋がる")
	assert.Equal(t, RoadW|RoadE, overlay[consts.Coord[consts.Chunk]{X: 1, Y: 0}], "途中は両隣に繋がる")
	assert.Equal(t, RoadW|RoadE, overlay[consts.Coord[consts.Chunk]{X: 2, Y: 0}], "途中は両隣に繋がる")
	assert.Equal(t, RoadW, overlay[consts.Coord[consts.Chunk]{X: 3, Y: 0}], "東端は西の隣だけに繋がる")
}

func TestMarkRoadLShape_西進でも東西ビットは対称に積む(t *testing.T) {
	t.Parallel()
	// buildRoadOverlay は常に a.X < b.X で呼ぶが、markRoadLShape 自体は符号で一般化している。
	// 西進 a.X > b.X の分岐も東進と鏡像で同じ結線になることを固定する
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	markRoadLShape(overlay, consts.Coord[consts.Chunk]{X: 3, Y: 0}, consts.Coord[consts.Chunk]{X: 0, Y: 0})

	assert.Equal(t, RoadW, overlay[consts.Coord[consts.Chunk]{X: 3, Y: 0}], "西進の始点は西の隣だけに繋がる")
	assert.Equal(t, RoadW|RoadE, overlay[consts.Coord[consts.Chunk]{X: 2, Y: 0}], "途中は両隣に繋がる")
	assert.Equal(t, RoadW|RoadE, overlay[consts.Coord[consts.Chunk]{X: 1, Y: 0}], "途中は両隣に繋がる")
	assert.Equal(t, RoadE, overlay[consts.Coord[consts.Chunk]{X: 0, Y: 0}], "西進の終点は東の隣だけに繋がる")
}

func TestMarkRoadLShape_角は水平と垂直のビットが同一チャンクに積まれる(t *testing.T) {
	t.Parallel()
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	// a=(0,0) から b=(2,3)。水平辺は行0を x0..2、垂直辺は列2を y0..3、角は (2,0)
	markRoadLShape(overlay, consts.Coord[consts.Chunk]{X: 0, Y: 0}, consts.Coord[consts.Chunk]{X: 2, Y: 3})

	assert.Equal(t, RoadE, overlay[consts.Coord[consts.Chunk]{X: 0, Y: 0}], "水平の始点")
	assert.Equal(t, RoadW|RoadE, overlay[consts.Coord[consts.Chunk]{X: 1, Y: 0}], "水平の途中")
	assert.Equal(t, RoadW|RoadS, overlay[consts.Coord[consts.Chunk]{X: 2, Y: 0}], "角は西と南、L 字の折れ")
	assert.Equal(t, RoadN|RoadS, overlay[consts.Coord[consts.Chunk]{X: 2, Y: 1}], "垂直の途中")
	assert.Equal(t, RoadN|RoadS, overlay[consts.Coord[consts.Chunk]{X: 2, Y: 2}], "垂直の途中")
	assert.Equal(t, RoadN, overlay[consts.Coord[consts.Chunk]{X: 2, Y: 3}], "垂直の終点")
}

func TestMarkRoadLShape_交差は四方向ビットになる(t *testing.T) {
	t.Parallel()
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	// 行1を横切る水平路と、列1を貫く垂直路を (1,1) で交差させる
	markRoadLShape(overlay, consts.Coord[consts.Chunk]{X: 0, Y: 1}, consts.Coord[consts.Chunk]{X: 2, Y: 1})
	markRoadLShape(overlay, consts.Coord[consts.Chunk]{X: 1, Y: 0}, consts.Coord[consts.Chunk]{X: 1, Y: 3})

	assert.Equal(t, RoadN|RoadS|RoadE|RoadW, overlay[consts.Coord[consts.Chunk]{X: 1, Y: 1}], "交差は四方向すべてに繋がる")
}

func TestMarkRoadLShape_同一チャンクは道を積まない(t *testing.T) {
	t.Parallel()
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	// 始点と終点が同じなら水平も垂直もステップが無く、ループは即終了して無限ループしない
	markRoadLShape(overlay, consts.Coord[consts.Chunk]{X: 2, Y: 2}, consts.Coord[consts.Chunk]{X: 2, Y: 2})

	assert.Empty(t, overlay, "動かない経路は何も積まない")
}

func TestBuildRoadOverlay_表示範囲に道が現れ端点が集落に一致する(t *testing.T) {
	t.Parallel()
	const runSeed uint64 = 12345
	const rows consts.Chunk = 9
	// 集落は Spacing チャンクごとに1つ当たる。数リージョン跨ぐ表示範囲なら必ず道が出る
	area := MacroRange{OriginX: 0, Cols: 3 * settlementPlacement.Spacing, Rows: rows}
	overlay := buildRoadOverlay(runSeed, area, rows)

	assert.NotEmpty(t, overlay, "複数リージョンを覆う表示範囲には道が出る")
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
