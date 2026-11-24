package states

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	w "github.com/kijimaD/ruins/internal/world"
)

const (
	// titlePixel は背景のピクセル絵の1画素の辺。火の粉と明かりもこの粗さで描く
	titlePixel = 3
	// titleEmbers は同時に舞う火の粉の数
	titleEmbers = 14
	// titleGlowW と titleGlowH は明かりの画像の大きさを、背景の1画素の数で表したもの
	titleGlowW = 170
	titleGlowH = 110
)

// titleGlowAt は明かりを重ねる左上。背景の左下の炎を覆う
var titleGlowAt = image.Pt(24, 696-titleGlowH*titlePixel)

// titleBackground はタイトル画面の背景。墜落現場の静止画の上で、火の粉をゆっくり舞わせ、炎の明かりをわずかに揺らす。
// 絵そのものは動かさず、静けさを保ったまま炎が生きていることだけを示す
type titleBackground struct {
	tick  int
	glow  *ebiten.Image
	ember *ebiten.Image
}

// Update は時を1つ進める
func (b *titleBackground) Update() {
	b.tick++
}

// Draw は背景と明かりと火の粉を描く
func (b *titleBackground) Draw(world w.World, screen *ebiten.Image) error {
	bg, err := loadBackgroundImage(world, "title1")
	if err != nil {
		return err
	}
	screen.DrawImage(bg, nil)
	b.drawGlow(screen)
	b.drawEmbers(screen)
	return nil
}

// drawGlow は炎の上に暖色の光を重ね、濃さをゆっくり揺らす
func (b *titleBackground) drawGlow(screen *ebiten.Image) {
	if b.glow == nil {
		b.glow = newTitleGlow()
	}
	t := float64(b.tick)
	flicker := 0.9 + 0.06*math.Sin(t*0.013) + 0.04*math.Sin(t*0.031+1.3)
	op := &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter}
	op.GeoM.Scale(titlePixel, titlePixel)
	op.GeoM.Translate(float64(titleGlowAt.X), float64(titleGlowAt.Y))
	op.ColorScale.ScaleAlpha(float32(flicker))
	screen.DrawImage(b.glow, op)
}

// newTitleGlow は下寄りの中ほどが明るく、外へ薄れる暖色の光の画像を作る
func newTitleGlow() *ebiten.Image {
	img := ebiten.NewImageWithOptions(image.Rect(0, 0, titleGlowW, titleGlowH), &ebiten.NewImageOptions{Unmanaged: true})
	pix := make([]byte, titleGlowW*titleGlowH*4)
	for y := range titleGlowH {
		for x := range titleGlowW {
			dx := (float64(x) - titleGlowW*0.4) / (titleGlowW * 0.5)
			dy := (float64(y) - titleGlowH*0.85) / (titleGlowH * 0.6)
			a := max(0, 1-math.Hypot(dx, dy))
			a = a * a * 0.16
			i := (y*titleGlowW + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = byte(255*a), byte(110*a), byte(40*a), byte(255*a)
		}
	}
	img.WritePixels(pix)
	return img
}

// titleEmberColors は火の粉の色。生まれたては明るく、昇るほど赤く暗くなる
var titleEmberColors = []color.RGBA{
	{R: 255, G: 224, B: 138, A: 255},
	{R: 255, G: 179, B: 71, A: 255},
	{R: 255, G: 106, B: 31, A: 255},
	{R: 192, G: 68, B: 26, A: 255},
}

// drawEmbers は炎の上から火の粉をゆっくり昇らせる。火の粉ごとに決まった乱数で出る所、速さ、寿命を決め、
// 時が巡ると同じ所からまた昇る
func (b *titleBackground) drawEmbers(screen *ebiten.Image) {
	if b.ember == nil {
		b.ember = ebiten.NewImageWithOptions(image.Rect(0, 0, 1, 1), &ebiten.NewImageOptions{Unmanaged: true})
		b.ember.Fill(color.White)
	}
	for i := range titleEmbers {
		r := emberRand(uint32(i))
		life := 420 + int(r()*420)
		age := (b.tick + int(r()*float64(life))) % life
		// 大きい炎と小さい炎の上から出る
		x0 := 70 + r()*180
		if r() < 0.3 {
			x0 = 400 + r()*60
		}
		y0 := 600 - r()*50
		rise := 0.12 + r()*0.15
		k := float64(age) / float64(life)
		x := x0 + math.Sin(float64(age)*0.012+r()*6)*10 + k*24
		y := y0 - float64(age)*rise
		c := titleEmberColors[min(len(titleEmberColors)-1, int(k*float64(len(titleEmberColors))))]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(titlePixel, titlePixel)
		op.GeoM.Translate(math.Floor(x/titlePixel)*titlePixel, math.Floor(y/titlePixel)*titlePixel)
		op.ColorScale.ScaleWithColor(c)
		// 出たてと消え際を薄くし、急に現れたり消えたりしないようにする
		op.ColorScale.ScaleAlpha(float32(math.Sin(math.Pi * k)))
		screen.DrawImage(b.ember, op)
	}
}

// emberRand は火の粉 i の決まった乱数を、呼ぶたびに [0, 1) で返す関数にする
func emberRand(i uint32) func() float64 {
	s := i*0x9e3779b9 + 0x7f4a7c15
	return func() float64 {
		s ^= s << 13
		s ^= s >> 17
		s ^= s << 5
		return float64(s) / (1 << 32)
	}
}
