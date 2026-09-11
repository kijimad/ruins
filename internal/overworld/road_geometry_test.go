package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestRoadSegments_水平辺は始点の行_垂直辺は終点の列(t *testing.T) {
	t.Parallel()
	segs := roadSegments(consts.Coord[consts.Chunk]{X: 0, Y: 0}, consts.Coord[consts.Chunk]{X: 2, Y: 3})

	assert.Equal(t, roadSeg{horizontal: true, fixed: 0, lo: 0, hi: 2}, segs[0], "水平辺は始点 a の行 a.Y、列は a.X..b.X")
	assert.Equal(t, roadSeg{horizontal: false, fixed: 2, lo: 0, hi: 3}, segs[1], "垂直辺は終点 b の列 b.X、行は a.Y..b.Y。角は (b.X, a.Y)")
}

func TestRoadSegments_端点は大小に依らず正規化される(t *testing.T) {
	t.Parallel()
	// a が b より東かつ南でも、水平辺は a の行、垂直辺は b の列で、範囲は min/max に揃う
	segs := roadSegments(consts.Coord[consts.Chunk]{X: 4, Y: 3}, consts.Coord[consts.Chunk]{X: 1, Y: 1})

	assert.Equal(t, roadSeg{horizontal: true, fixed: 3, lo: 1, hi: 4}, segs[0], "水平辺の列は min/max で正規化")
	assert.Equal(t, roadSeg{horizontal: false, fixed: 1, lo: 1, hi: 3}, segs[1], "垂直辺の行は min/max で正規化")
}

func TestRoadSegTileSpan_固定軸と可変軸を正しいチャンク寸法で中心へ寄せる(t *testing.T) {
	t.Parallel()
	// chunkW=24, chunkH=16 と非対称にして軸の取り違えを検出する
	const chunkW, chunkH consts.Tile = 24, 16

	// 水平辺は行を固定→Y は chunkH、可変の列→X は chunkW
	hFixed, hLo, hHi := roadSeg{horizontal: true, fixed: 2, lo: 0, hi: 3}.tileSpan(chunkW, chunkH)
	assert.Equal(t, consts.Tile(2*16+8), hFixed, "固定行のタイルYはチャンクH中心")
	assert.Equal(t, consts.Tile(0*24+12), hLo, "可変列のタイルXはチャンクW中心")
	assert.Equal(t, consts.Tile(3*24+12), hHi, "可変列のタイルXはチャンクW中心")

	// 垂直辺は列を固定→X は chunkW、可変の行→Y は chunkH
	vFixed, vLo, vHi := roadSeg{horizontal: false, fixed: 2, lo: 0, hi: 3}.tileSpan(chunkW, chunkH)
	assert.Equal(t, consts.Tile(2*24+12), vFixed, "固定列のタイルXはチャンクW中心")
	assert.Equal(t, consts.Tile(0*16+8), vLo, "可変行のタイルYはチャンクH中心")
	assert.Equal(t, consts.Tile(3*16+8), vHi, "可変行のタイルYはチャンクH中心")
}
