package overworld

import (
	"image"
	"image/color"
	"image/draw"
	"strconv"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// RenderNorthward は world に地上の定義の帯の形で、開始の行から北へ north 行、南へ south 行のチャンクを生成し、
// 1タイル tilePx 四方で描いてチャンクの座標を書き込んだ画像にする。生成の自然さや面白さを人が目で確かめるのに使う。
// world は生成するチャンクを持たない、raw と定義を読んだだけのものを渡す
func RenderNorthward(world w.World, runSeed uint64, north, south consts.Chunk, tilePx int) (*image.RGBA, error) {
	if err := generateNorthward(world, runSeed, north, south); err != nil {
		return nil, err
	}
	chunkW, chunkH, cols, _ := dungeon.DungeonOverworld.BandShape()
	rows := north + south + 1
	m, err := RenderRegion(world, image.Rect(0, 0, int(cols.Tiles(chunkW)), int(rows.Tiles(chunkH))), tilePx)
	if err != nil {
		return nil, err
	}
	return withChunkGrid(m, int(chunkW)*tilePx, int(chunkH)*tilePx, int(cols), int(rows), int(north)), nil
}

// generateNorthward は開始の行から北へ north 行、南へ south 行のチャンクを world に生成する。北端の行を帯ローカルの0行目に置き、
// ゲームが北へ進んだときと同じく帯の NorthIndex で絶対の行と結ぶ。ゲームは帯の行を北へ1行ずつ足していくが、ここでは北端から
// 一度に並べる。チャンクの生成が位置と seed だけで決まり、隣とのつながりをオートタイルの計算し直しで合わせる間は、順が違っても同じ地図になる
func generateNorthward(world w.World, runSeed uint64, north, south consts.Chunk) error {
	chunkW, chunkH, cols, rows := dungeon.DungeonOverworld.BandShape()
	sb := query.EnsureSeamlessBand(world)
	*sb = gc.SeamlessBand{Active: true, RunSeed: runSeed, ChunkW: chunkW, ChunkH: chunkH, Cols: cols, Rows: rows}
	spawnRow := sb.SpawnChunkY()
	top := spawnRow - north
	sb.NorthIndex = -top

	gen := NewChunkGen(world, runSeed, chunkW, chunkH, cols, mapplanner.PlannerTypeOverworldField)
	for cy := top; cy <= spawnRow+south; cy++ {
		for cx := range cols {
			if err := gen(consts.Coord[consts.Chunk]{X: cx, Y: cy}, cx.Tiles(chunkW), (cy - top).Tiles(chunkH)); err != nil {
				return err
			}
		}
	}
	return nil
}

const (
	// gridMarginLeft と gridMarginTop は地図の左と上に空ける、行と列の番号を書く余白
	gridMarginLeft = 40
	gridMarginTop  = 18
)

var (
	gridLineColor  = color.NRGBA{R: 255, G: 255, B: 255, A: 160}
	gridLabelColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	gridLabelBack  = color.NRGBA{A: 200}
	gridMarginBack = color.NRGBA{R: 32, G: 32, B: 36, A: 255}
)

// withChunkGrid は地図 m の周りに余白を足し、チャンクの境目に線を引いて、チャンクごとに座標を書く。列は西から A, B, C、
// 行は開始の行を 0 として北へ 1, 2, 3 と数える。一番上の行は north。どのチャンクにも C5 のような座標を書く
func withChunkGrid(m *image.RGBA, chunkPxW, chunkPxH, cols, rows, north int) *image.RGBA {
	b := m.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, gridMarginLeft+b.Dx(), gridMarginTop+b.Dy()))
	draw.Draw(img, img.Bounds(), image.NewUniform(gridMarginBack), image.Point{}, draw.Src)
	draw.Draw(img, b.Add(image.Pt(gridMarginLeft, gridMarginTop)), m, image.Point{}, draw.Src)

	// 線は各チャンクの北と西の辺に引く。東端と南端は画像の端なので引かない
	line := image.NewUniform(gridLineColor)
	for c := range cols {
		x := gridMarginLeft + c*chunkPxW
		draw.Draw(img, image.Rect(x, gridMarginTop, x+1, img.Bounds().Dy()), line, image.Point{}, draw.Over)
	}
	for r := range rows {
		y := gridMarginTop + r*chunkPxH
		draw.Draw(img, image.Rect(gridMarginLeft, y, img.Bounds().Dx(), y+1), line, image.Point{}, draw.Over)
	}
	for c := range cols {
		drawGridLabel(img, gridMarginLeft+c*chunkPxW+chunkPxW/2-3, 13, columnName(c), false)
	}
	for r := range rows {
		n := strconv.Itoa(north - r)
		drawGridLabel(img, 4, gridMarginTop+r*chunkPxH+chunkPxH/2+4, n, false)
		for c := range cols {
			drawGridLabel(img, gridMarginLeft+c*chunkPxW+4, gridMarginTop+r*chunkPxH+14, columnName(c)+n, true)
		}
	}
	return img
}

// columnName は西から c 番目の列の名前を A から返す
func columnName(c int) string {
	return string(rune('A' + c))
}

// drawGridLabel は (x, y) を文字の基準線の左端として s を書く。back が立っていれば文字の下に暗い帯を敷く
func drawGridLabel(img *image.RGBA, x, y int, s string, back bool) {
	face := basicfont.Face7x13
	if back {
		r := image.Rect(x-2, y-11, x+len(s)*face.Advance+2, y+3)
		draw.Draw(img, r, image.NewUniform(gridLabelBack), image.Point{}, draw.Over)
	}
	d := font.Drawer{Dst: img, Src: image.NewUniform(gridLabelColor), Face: face, Dot: fixed.P(x, y)}
	d.DrawString(s)
}
