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
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// TestGolden_Facility は施設ごとに、描いた絵を全て北向きのまま並べて固定する
func TestGolden_Facility(t *testing.T) {
	t.Parallel()

	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	for _, fac := range loader.Facilities() {
		t.Run(fac, func(t *testing.T) {
			t.Parallel()
			names := loader.FacilityChunkNames(fac)
			cells := make([][][]maptemplate.MapCell, 0, len(names))
			for _, name := range names {
				_, _, c, err := loader.LoadTemplateByName(name, 1)
				require.NoError(t, err)
				cells = append(cells, c)
			}
			assertFacilityGolden(t, fac, names, cells)
		})
	}
}

// TestGolden_FacilityVariants は施設ごとに seed を変えて6棟並べ、絵の選択、家具の候補、部品、戦利品、回転と反転が出す変化を固定する
func TestGolden_FacilityVariants(t *testing.T) {
	t.Parallel()

	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	for _, fac := range loader.Facilities() {
		t.Run(fac, func(t *testing.T) {
			t.Parallel()
			const n = 6
			names := make([]string, 0, n)
			cells := make([][][]maptemplate.MapCell, 0, n)
			for i := range n {
				seed := uint64(i + 1)
				chunk, _, err := loader.FacilityChunk(fac, seed)
				require.NoError(t, err)
				c, err := planFacility(loader, fac, seed, rand.New(rand.NewPCG(seed, 0x2)))
				require.NoError(t, err)
				names = append(names, chunk.Name)
				cells = append(cells, c)
			}
			assertFacilityGolden(t, fac, names, cells)
		})
	}
}

// assertFacilityGolden は施設の絵のマスを3列に並べて実体にし、スプライトで描いて golden と比べる。各棟に名前を添える
func assertFacilityGolden(t *testing.T, fac string, names []string, cells [][][]maptemplate.MapCell) {
	t.Helper()
	const cols = 3
	const lot = facilityLot
	rows := (len(cells) + cols - 1) / cols
	world := testutil.InitTestWorld(t)
	tiles := &tileIndex{world: world, loX: 0, hiX: lot * cols}
	for i, c := range cells {
		g := chunkGeom{offsetX: lot * consts.Tile(i%cols), offsetY: lot * consts.Tile(i/cols), chunkW: lot, chunkH: lot, tiles: tiles}
		_, _, err := spawnFacility(world, g, c, 0, 0, fac, 1, uint64(i+1))
		require.NoError(t, err)
	}
	img := renderOverworldRegion(t, world, 0, 0, int(lot)*cols, int(lot)*rows)
	for i, name := range names {
		drawLabel(img, (i%cols)*int(lot)*scatterCellPx+4, (i/cols)*int(lot)*scatterCellPx+14, name)
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
