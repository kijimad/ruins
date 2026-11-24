package states

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	w "github.com/kijimaD/ruins/internal/world"
)

const (
	// titlePixel は背景のピクセル絵の1画素の辺。雪もこの粗さで描く
	titlePixel = 3
	// titleSnowflakes は同時に降る雪の粒の数
	titleSnowflakes = 40
)

// titleBackground はタイトル画面の背景。墜落から時間がたった後の静止画の上に、雪をゆっくり降らせる
type titleBackground struct {
	tick  int
	flake *ebiten.Image
}

// Update は時を1つ進める
func (b *titleBackground) Update() {
	b.tick++
}

// Draw は背景と雪を描く
func (b *titleBackground) Draw(world w.World, screen *ebiten.Image) error {
	bg, err := loadBackgroundImage(world, "title1")
	if err != nil {
		return err
	}
	screen.DrawImage(bg, nil)
	b.drawSnow(screen)
	return nil
}

// drawSnow は雪の粒を上から下へゆっくり降らせる。粒ごとに決まった乱数で列、奥行き、揺れを決め、時刻だけから位置を求める。
// 奥の粒ほど暗く遅く、手前の粒ほど明るく速い
func (b *titleBackground) drawSnow(screen *ebiten.Image) {
	if b.flake == nil {
		b.flake = ebiten.NewImageWithOptions(image.Rect(0, 0, 1, 1), &ebiten.NewImageOptions{Unmanaged: true})
		b.flake.Fill(color.White)
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	for i := range titleSnowflakes {
		r := flakeRand(uint32(i))
		depth := r()
		fall := 0.2 + depth*0.3
		x0 := r() * float64(sw)
		phase := r() * float64(sh)
		sway := r() * 6
		y := math.Mod(phase+float64(b.tick)*fall, float64(sh+titlePixel)) - titlePixel
		x := x0 + math.Sin(float64(b.tick)*0.01+sway)*12
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(titlePixel, titlePixel)
		op.GeoM.Translate(math.Floor(x/titlePixel)*titlePixel, math.Floor(y/titlePixel)*titlePixel)
		op.ColorScale.ScaleAlpha(float32(0.35 + depth*0.5))
		screen.DrawImage(b.flake, op)
	}
}

// flakeRand は雪の粒 i の決まった乱数を、呼ぶたびに [0, 1) で返す関数にする
func flakeRand(i uint32) func() float64 {
	s := i*0x9e3779b9 + 0x7f4a7c15
	return func() float64 {
		s ^= s << 13
		s ^= s >> 17
		s ^= s << 5
		return float64(s) / (1 << 32)
	}
}
