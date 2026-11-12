package overworld

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// TestGolden_FacilityHouse は家の絵を seed を変えて4棟並べ、実体にしたタイルと prop をスプライトで描いて固定する。
// 絵の選択、家具の候補、部品、回転と反転がどう変化を出すかを1枚で見る。棟ごとに選ばれた絵の名前を添える
func TestGolden_FacilityHouse(t *testing.T) {
	t.Parallel()

	const cols, rows = 2, 2
	const lot consts.Tile = 20
	world := testutil.InitTestWorld(t)
	loader, err := facilityTemplates()
	require.NoError(t, err)

	g := chunkGeom{chunkW: lot * cols, chunkH: lot * rows, tiles: &tileIndex{world: world, loX: 0, hiX: lot * cols}}
	labels := make([]string, 0, cols*rows)
	for i := range cols * rows {
		seed := uint64(i + 1)
		chunk, _, err := loader.FacilityChunk("house", seed)
		require.NoError(t, err)
		cells, err := planFacility(loader, "house", seed, rand.New(rand.NewPCG(seed, 0x2)))
		require.NoError(t, err)
		ox, oy := lot*consts.Tile(i%cols), lot*consts.Tile(i/cols)
		cg := chunkGeom{offsetX: ox, offsetY: oy, chunkW: lot, chunkH: lot, tiles: g.tiles}
		_, _, err = spawnFacility(world, cg, cells, 0, 0, "house", 1, seed)
		require.NoError(t, err)
		labels = append(labels, chunk.Name)
	}

	img := renderOverworldRegion(t, world, 0, 0, int(lot)*cols, int(lot)*rows)
	for i, label := range labels {
		drawLabel(img, (i%cols)*int(lot)*scatterCellPx+4, (i/cols)*int(lot)*scatterCellPx+14, label)
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	goldie.New(t, goldie.WithNameSuffix(".png")).Assert(t, t.Name(), buf.Bytes())
}

// drawLabel は画像の (x, y) に白い文字を黒い帯の上へ書く
func drawLabel(img *image.RGBA, x, y int, s string) {
	draw.Draw(img, image.Rect(x-2, y-12, x+len(s)*7+2, y+3), image.NewUniform(color.Black), image.Point{}, draw.Src)
	d := font.Drawer{Dst: img, Src: image.NewUniform(color.White), Face: basicfont.Face7x13, Dot: fixed.P(x, y)}
	d.DrawString(s)
}
