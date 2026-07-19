package systems

import (
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
)

// FrostRenderSystem は寒波前線の極低温ゾーンを氷白のオーバーレイで描く Renderer。
//
// 前線は SeamlessBand に絶対軸で公開されている。ゾーン内タイルを半透明の氷で覆い、
// 西へ深いほど濃く、進入不可ライン以西は凍結壁として最も濃く塗る。「迫る霜の壁」を可視化する。
// タイル描画の後・HUD の前に走らせて地形とキャラの上に載せる。前線が無効な通常ダンジョンでは何もしない。
type FrostRenderSystem struct{}

// String はシステム名を返す。w.Renderer interface を実装する
func (sys FrostRenderSystem) String() string { return "FrostRenderSystem" }

// frostTileImage は1タイルぶんの氷白画像。アルファは描画時に ColorScale で調整する。
// 並列 golden テストが同時に Draw を叩きうるため Once で保護する
var (
	frostTileImage     *ebiten.Image
	frostTileImageOnce sync.Once
)

func initFrostImage() {
	frostTileImageOnce.Do(func() {
		ts := int(consts.TileSize)
		if ts <= 0 {
			return
		}
		frostTileImage = ebiten.NewImage(ts, ts)
		// 青みのある氷色。白系だと明るいだけになり霜に見えないため寒色を入れる
		frostTileImage.Fill(color.RGBA{130, 205, 240, 255})
	})
}

// Draw は極低温ゾーンに氷のオーバーレイを描く。
func (sys *FrostRenderSystem) Draw(world w.World, screen *ebiten.Image) error {
	sb := query.GetDungeon(world).SeamlessBand
	if !sb.Front.Active {
		return nil
	}
	initFrostImage()
	if frostTileImage == nil {
		return nil
	}

	camera := getCamera(world)
	minX, maxX, minY, maxY := viewportTileBounds(world, viewportCullMargin, camera)

	// 帯の範囲でクランプして帯外を塗らない
	level := query.GetDungeon(world).Level
	bandW, bandH := int(level.TileWidth), int(level.TileHeight)
	minX, minY = max(minX, 0), max(minY, 0)
	maxX, maxY = min(maxX, bandW-1), min(maxY, bandH-1)

	frontEast := int(sb.Front.EastAbsX)
	coldZoneWest := int(sb.Front.ColdZoneWest())
	ts := int(consts.TileSize)

	for x := minX; x <= maxX; x++ {
		alpha, draw := frostAlpha(frontEast, coldZoneWest, int(sb.LocalToAbsX(consts.Tile(x))))
		if !draw {
			continue
		}
		for y := minY; y <= maxY; y++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x*ts), float64(y*ts))
			setTranslate(world, op, camera)
			op.ColorScale.ScaleAlpha(alpha)
			screen.DrawImage(frostTileImage, op)
		}
	}
	return nil
}

// frostAlpha は絶対 X の列に塗る氷のアルファと、塗るかどうかを返す純関数。
//
// 前線より東は塗らない。極低温ゾーン内は前線東端で薄く西へ深いほど濃い。
// 進入不可ライン ColdZoneWest 以西は凍結壁として最も濃い。
func frostAlpha(frontEast, coldZoneWest, absX int) (alpha float32, draw bool) {
	if absX > frontEast {
		return 0, false // 前線より東は平常
	}
	if absX <= coldZoneWest {
		return 0.9, true // 破棄/進入不可の凍結壁
	}
	zoneWidth := max(1, frontEast-coldZoneWest)
	depth := float32(frontEast-absX) / float32(zoneWidth) // 0(東端)〜1(西端寄り)
	return 0.25 + 0.5*depth, true
}
