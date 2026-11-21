package resources

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/draw"

	"github.com/kijimaD/ruins/internal/components"
)

// errNilSpriteRender は SpriteRender が nil のときに返す
var errNilSpriteRender = errors.New("sprite render is nil")

// SpriteStore は SpriteRender が指すスプライトの画像を解決してキャッシュする。
// 等倍の部分画像と、表示サイズへ縮めた画像の両方を同じ置き場に持ち、スプライトから画像を得る道を1つにする。
//
// world ごとに持つ。グローバルな可変状態を作らず、並行して走るテストが互いの画像を混ぜない。
type SpriteStore struct {
	sheets map[string]components.SpriteSheet
	images map[spriteKey]*ebiten.Image
	gauges map[gaugeKey]*ebiten.Image
}

// gaugeKey はゲージを焼いた縮小画像を一意に決める組
type gaugeKey struct {
	sprite spriteKey
	gauge  Gauge
}

// Gauge はアイコンの下端に焼く段階のゲージ。Filled 段を Fill の色で、残りを Empty の色で描く
type Gauge struct {
	Filled int
	Total  int
	Fill   color.RGBA
	Empty  color.RGBA
}

// spriteKey は解決結果を一意に決める組。size=0 は等倍、正の size は同じスプライトの縮小を表す。
type spriteKey struct {
	sheet  string
	sprite string
	size   int
}

// NewSpriteStore は空のスプライト置き場を作る。
func NewSpriteStore() *SpriteStore {
	return &SpriteStore{images: map[spriteKey]*ebiten.Image{}, gauges: map[gaugeKey]*ebiten.Image{}}
}

// SetSheets は解決元のスプライトシートを差し替える。Resources.SetSpriteSheets から呼ぶ。
// シートが変われば解決結果も変わるので、キャッシュは捨てる。
func (s *SpriteStore) SetSheets(sheets map[string]components.SpriteSheet) {
	s.sheets = sheets
	s.images = map[spriteKey]*ebiten.Image{}
	s.gauges = map[gaugeKey]*ebiten.Image{}
}

// Image は sr が指すスプライトの等倍の部分画像を返す。解決できなければ nil を返し、描かない行として扱わせる。
func (s *SpriteStore) Image(sr *components.SpriteRender) *ebiten.Image {
	if sr == nil {
		return nil
	}
	key := spriteKey{sheet: sr.SpriteSheetName, sprite: sr.SpriteKey}
	if img, ok := s.images[key]; ok {
		return img
	}
	tex, rect, ok := s.Rect(sr)
	if !ok {
		return nil
	}
	img := components.SubImage(tex.Image, rect)
	s.images[key] = img
	return img
}

// Sized は sr が指すスプライトを size の正方へ収まるまで縦横比を保って縮めた画像を返す。解決できなければ nil。
func (s *SpriteStore) Sized(sr *components.SpriteRender, size int) *ebiten.Image {
	if sr == nil || size <= 0 {
		return nil
	}
	key := spriteKey{sheet: sr.SpriteSheetName, sprite: sr.SpriteKey, size: size}
	if img, ok := s.images[key]; ok {
		return img
	}
	tex, rect, ok := s.Rect(sr)
	if !ok {
		return nil
	}
	img := shrinkToFit(tex, rect, size)
	if img == nil {
		return nil
	}
	s.images[key] = img
	return img
}

// SizedGauge は Sized と同じ縮小画像の下端に gauge を焼いた画像を返す。スプライトと段階の組ごとに一度だけ作る。
// CPU 側の元画像が無いスプライトはゲージを焼けないので、Sized の画像を返す
func (s *SpriteStore) SizedGauge(sr *components.SpriteRender, size int, gauge Gauge) *ebiten.Image {
	if sr == nil || size <= 0 {
		return nil
	}
	key := gaugeKey{sprite: spriteKey{sheet: sr.SpriteSheetName, sprite: sr.SpriteKey, size: size}, gauge: gauge}
	if img, ok := s.gauges[key]; ok {
		return img
	}
	tex, rect, ok := s.Rect(sr)
	if !ok {
		return nil
	}
	if tex.Source == nil {
		return s.Sized(sr, size)
	}
	dst := shrinkToRGBA(tex, rect, size)
	if dst == nil {
		return nil
	}
	drawGauge(dst, gauge)
	img := ebiten.NewImage(dst.Bounds().Dx(), dst.Bounds().Dy())
	img.WritePixels(dst.Pix)
	s.gauges[key] = img
	return img
}

// drawGauge は dst の下端に段階のゲージを描く。段のあいだは1ピクセル空ける
func drawGauge(dst *image.RGBA, g Gauge) {
	if g.Total <= 0 {
		return
	}
	const height = 3
	b := dst.Bounds()
	segW := max(1, (b.Dx()-(g.Total-1))/g.Total)
	left := (b.Dx() - (segW*g.Total + g.Total - 1)) / 2
	top := b.Dy() - height
	for i := range g.Total {
		c := g.Empty
		if i < g.Filled {
			c = g.Fill
		}
		x := left + i*(segW+1)
		draw.Draw(dst, image.Rect(x, top, x+segW, b.Dy()), image.NewUniform(c), image.Point{}, draw.Src)
	}
}

// Rect は sr が指すスプライトのテクスチャと、その中の切り出し矩形を返す。解決できなければ ok=false。
// 矩形はテクスチャ範囲へクランプした絶対座標で、SubImage の切り出しにも、アトラス全体を
// 使うときの UV 座標にも使える。スプライトの矩形を求める箇所はこれを通し、解決を1つに保つ。
func (s *SpriteStore) Rect(sr *components.SpriteRender) (components.Texture, image.Rectangle, bool) {
	sheet, rect, err := spriteRect(s.sheets, sr)
	if err != nil {
		return components.Texture{}, image.Rectangle{}, false
	}
	return sheet.Texture, rect, true
}

// spriteRect は SpriteRender が指すシートと、その中のスプライトの矩形を返す。
// 矩形はテクスチャ範囲へクランプするので、範囲外を指す指定でも安全に切り出せる
func spriteRect(sheets map[string]components.SpriteSheet, sr *components.SpriteRender) (components.SpriteSheet, image.Rectangle, error) {
	if sr == nil {
		return components.SpriteSheet{}, image.Rectangle{}, errNilSpriteRender
	}
	sheet, ok := sheets[sr.SpriteSheetName]
	if !ok {
		return components.SpriteSheet{}, image.Rectangle{}, fmt.Errorf("sprite sheet %q not found", sr.SpriteSheetName)
	}
	sprite, ok := sheet.Sprites[sr.SpriteKey]
	if !ok {
		return components.SpriteSheet{}, image.Rectangle{}, fmt.Errorf("sprite key %q not found in sheet %q", sr.SpriteKey, sr.SpriteSheetName)
	}
	if sheet.Texture.Image == nil {
		return components.SpriteSheet{}, image.Rectangle{}, fmt.Errorf("sprite sheet %q has no texture image", sr.SpriteSheetName)
	}
	w := sheet.Texture.Image.Bounds().Dx()
	h := sheet.Texture.Image.Bounds().Dy()
	rect := image.Rect(
		max(0, sprite.X),
		max(0, sprite.Y),
		min(w, sprite.X+sprite.Width),
		min(h, sprite.Y+sprite.Height),
	)
	return sheet, rect, nil
}

// shrinkToRGBA は元画像から rect を切り出し、size の正方へ縦横比を保って収めた CPU 側の画像を返す。
// size より小さいスプライトは縮めずに写す
func shrinkToRGBA(tex components.Texture, rect image.Rectangle, size int) *image.RGBA {
	sw, sh := rect.Dx(), rect.Dy()
	if sw <= 0 || sh <= 0 {
		return nil
	}
	dw, dh := sw, sh
	if sw > size || sh > size {
		dw, dh = size, size
		if sw >= sh {
			dh = max(1, sh*size/sw)
		} else {
			dw = max(1, sw*size/sh)
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	if dw == sw && dh == sh {
		draw.Draw(dst, dst.Bounds(), tex.Source, rect.Min, draw.Src)
		return dst
	}
	// image.RGBA は乗算済みアルファなので、透過の境界でも色がにじまない重み付けになる
	draw.CatmullRom.Scale(dst, dst.Bounds(), tex.Source, rect, draw.Src, nil)
	return dst
}

// shrinkToFit は tex の rect の部分を size の正方へ収まるまで縦横比を保って縮める。
// 拡大はしないので、すでに収まっている部分はテクスチャの部分画像をそのまま返す。
//
// GPU の端数倍率の縮小はテクセル境界の読みが揺れて決定的でないため、縮小は CPU で一度だけ行う。
func shrinkToFit(tex components.Texture, rect image.Rectangle, size int) *ebiten.Image {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return nil
	}
	if (rect.Dx() <= size && rect.Dy() <= size) || tex.Source == nil {
		return components.SubImage(tex.Image, rect)
	}
	dst := shrinkToRGBA(tex, rect, size)
	out := ebiten.NewImage(dst.Bounds().Dx(), dst.Bounds().Dy())
	out.WritePixels(dst.Pix)
	return out
}
