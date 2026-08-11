package systems

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// lightGradientSize は光だまりグラデーション画像の一辺。
const lightGradientSize = 128

// lightGradientImage は放射状グラデーションのキャッシュ。中心1、外周0へ滑らかに落ちる。
var lightGradientImage *ebiten.Image

// lightGradient はソフトな放射状グラデーションを返す。光源1つぶんの光だまりの元になる。
func lightGradient() *ebiten.Image {
	if lightGradientImage != nil {
		return lightGradientImage
	}
	img := ebiten.NewImage(lightGradientSize, lightGradientSize)
	c := float64(lightGradientSize) / 2
	pix := make([]byte, lightGradientSize*lightGradientSize*4)
	for y := range lightGradientSize {
		for x := range lightGradientSize {
			dx := (float64(x) + 0.5 - c) / c
			dy := (float64(y) + 0.5 - c) / c
			d := math.Hypot(dx, dy)
			// ガウシアン状の減衰。端で 0 に落ちきらず薄く残るため、明暗の境界が曖昧になる
			v := math.Exp(-2.3 * d * d)
			i := (y*lightGradientSize + x) * 4
			b := byte(v * 255)
			pix[i] = b
			pix[i+1] = b
			pix[i+2] = b
			pix[i+3] = 255
		}
	}
	img.WritePixels(pix)
	lightGradientImage = img
	return img
}

// BuildLightMap は乗算用のライトマップを dst へ描く。
// 全面を darkness に応じた環境光で塗り、有効な各 LightSource の位置へ色つきの
// ソフトな光だまりを加算する。世界フレームにこれを乗算すると、配置された光源が
// そのまま雰囲気の光になる。darkness=0 なら全面白で世界は素のまま。
// フォグと壁遮蔽は世界フレーム側が黒く落としているので、乗算すれば保たれる。
func BuildLightMap(world w.World, dst *ebiten.Image, darkness float64) {
	if dst == nil {
		return
	}
	darkness = math.Max(0, math.Min(1, darkness))

	// 環境光。darkness=0 で白（素通し）、1 で寒色の暗がり
	lerp := func(bright, dark float64) byte { return byte(bright + (dark-bright)*darkness) }
	dst.Fill(color.RGBA{
		R: lerp(255, 70),
		G: lerp(255, 82),
		B: lerp(255, 105),
		A: 255,
	})

	// 昼は光だまり不要。素通しにする
	if darkness <= 0 {
		return
	}

	camera := getCamera(world)
	grad := lightGradient()
	gsize := float64(lightGradientSize)
	ts := float64(consts.TileSize)

	lightQuery := query.ActiveFilter2[gc.LightSource, gc.GridElement](world).Query()
	for lightQuery.Next() {
		entity := lightQuery.Entity()
		ls := world.Components.LightSource.Get(entity)
		if !ls.Enabled {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		center := consts.TileCenterToWorld(grid.Coord)
		radiusWorld := float64(ls.Radius) * ts
		if radiusWorld <= 0 {
			continue
		}

		op := &ebiten.DrawImageOptions{}
		// グラデーションを光源半径の直径へ拡大し、光源中心へ置く
		scale := (radiusWorld * 2) / gsize
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(float64(center.X)-radiusWorld, float64(center.Y)-radiusWorld)
		setTranslate(world, op, camera)
		// 加算合成で光だまりを重ねる
		op.Blend = ebiten.BlendLighter
		// 光源色を掛ける。darkness を強度に使い、昼は光だまりを出さない
		op.ColorScale.Scale(
			float32(float64(ls.Color.R)/255*darkness),
			float32(float64(ls.Color.G)/255*darkness),
			float32(float64(ls.Color.B)/255*darkness),
			1,
		)
		dst.DrawImage(grad, op)
	}
}
