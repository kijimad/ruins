package interior

import (
	"image"
	"image/color"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/sebdah/goldie/v2"
)

// scatterDotPx は ScatterArea 分布 VRT の1タイルの辺長。分布の粗密が読める程度に小さくする。
const scatterDotPx = 10

// scatterGoldenSeed は分布 VRT の固定 seed。散布は純関数なので同じ seed で golden が安定する。
const scatterGoldenSeed uint64 = 20260802

// TestGolden_ScatterArea は密度場の芯 ScatterArea の点選定分布を固定する。密度を上げると候補域が
// hash ジッタで格子状にならず均等に埋まること、accept 述語で候補を絞れること、外周1タイルは
// interiorTiles が除いて常に空くこと、を目視で捕らえる。同じ seed と入力で golden が安定する。
// この芯は overworld の屋外散布と interior の全域散布の双方が共有するので、分布の退行はここで止める。
func TestGolden_ScatterArea(t *testing.T) {
	t.Parallel()

	area := Rect{X: 0, Y: 0, W: 32, H: 32}
	all := func(Vec) bool { return true }
	parity := func(v Vec) bool { return (v.X+v.Y)%2 == 0 }

	panels := []*image.RGBA{
		renderScatterPanel(area, all, 90),
		renderScatterPanel(area, all, 300),
		renderScatterPanel(area, all, 630),
		renderScatterPanel(area, parity, 300),
	}
	labels := []string{"all n=90", "all n=300", "all n=630", "parity n=300"}

	g := goldie.New(t, goldie.WithNameSuffix(".png"), goldie.WithEqualFn(testutil.PNGPixelEqual))
	g.Assert(t, t.Name(), montage(t, int(area.W)*scatterDotPx, int(area.H)*scatterDotPx, 2, labels, panels))
}

// renderScatterPanel は area に対する ScatterArea の選定結果を1枚のグリッドへ描く。候補域は下地色、
// 選定タイルは明色、外周の非候補は暗色で塗り、密度場の広がりを可視化する。
func renderScatterPanel(area Rect, accept func(Vec) bool, count int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(area.W)*scatterDotPx, int(area.H)*scatterDotPx))
	sel := make(map[Vec]bool)
	for _, v := range ScatterArea(area, accept, scatterGoldenSeed, count) {
		sel[v] = true
	}
	var (
		bgOuter = color.RGBA{R: 28, G: 26, B: 22, A: 255}   // 外周の非候補
		bgCand  = color.RGBA{R: 60, G: 54, B: 44, A: 255}   // 候補域の下地
		selCol  = color.RGBA{R: 132, G: 168, B: 82, A: 255} // 選定タイル
	)
	for y := range int(area.H) {
		for x := range int(area.W) {
			v := Vec{X: area.X + consts.Tile(x), Y: area.Y + consts.Tile(y)}
			c := bgOuter
			if x > 0 && x < int(area.W)-1 && y > 0 && y < int(area.H)-1 {
				c = bgCand
			}
			if sel[v] {
				c = selCol
			}
			fillScatterDot(img, x*scatterDotPx, y*scatterDotPx, c)
		}
	}
	return img
}

// fillScatterDot は px,py を左上に scatterDotPx 四方のセルを c で塗る。パッケージ既定の fillCell は
// 32px 固定なので、分布 VRT 向けの小さいセルにはこの専用の塗りを使う。
func fillScatterDot(img *image.RGBA, px, py int, c color.RGBA) {
	for yy := py; yy < py+scatterDotPx; yy++ {
		for xx := px; xx < px+scatterDotPx; xx++ {
			img.SetRGBA(xx, yy, c)
		}
	}
}
