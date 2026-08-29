package resources

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/draw"

	"github.com/kijimaD/ruins/internal/components"
)

// SpriteStore は SpriteRender が指すスプライトの画像を解決してキャッシュする。
// 等倍の部分画像と、表示サイズへ縮めた画像の両方を同じ置き場に持ち、スプライトから画像を得る道を1つにする。
//
// world ごとに持つ。グローバルな可変状態を作らず、並行して走るテストが互いの画像を混ぜない。
type SpriteStore struct {
	sheets map[string]components.SpriteSheet
	images map[spriteKey]*ebiten.Image
}

// spriteKey は解決結果を一意に決める組。size=0 は等倍、正の size は同じスプライトの縮小を表す。
type spriteKey struct {
	sheet  string
	sprite string
	size   int
}

// NewSpriteStore は空のスプライト置き場を作る。
func NewSpriteStore() *SpriteStore {
	return &SpriteStore{images: map[spriteKey]*ebiten.Image{}}
}

// SetSheets は解決元のスプライトシートを差し替える。Resources.SetSpriteSheets から呼ぶ。
// シートが変われば解決結果も変わるので、キャッシュは捨てる。
func (s *SpriteStore) SetSheets(sheets map[string]components.SpriteSheet) {
	s.sheets = sheets
	s.images = map[spriteKey]*ebiten.Image{}
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
	sheet, rect, err := spriteRect(s.sheets, sr)
	if err != nil {
		return nil
	}
	img := components.SubImage(sheet.Texture.Image, rect)
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
	sheet, rect, err := spriteRect(s.sheets, sr)
	if err != nil {
		return nil
	}
	img := shrinkToFit(sheet.Texture, rect, size)
	if img == nil {
		return nil
	}
	s.images[key] = img
	return img
}

// spriteRect は SpriteRender が指すシートと、その中のスプライトの矩形を返す。
// 矩形はテクスチャ範囲へクランプするので、範囲外を指す指定でも安全に切り出せる
func spriteRect(sheets map[string]components.SpriteSheet, sr *components.SpriteRender) (components.SpriteSheet, image.Rectangle, error) {
	if sr == nil {
		return components.SpriteSheet{}, image.Rectangle{}, fmt.Errorf("sprite render is nil")
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

// shrinkToFit は tex の rect の部分を size の正方へ収まるまで縦横比を保って縮める。
// 拡大はしないので、すでに収まっている部分はテクスチャの部分画像をそのまま返す。
//
// GPU の端数倍率の縮小はテクセル境界の読みが揺れて決定的でないため、縮小は CPU で一度だけ行う。
func shrinkToFit(tex components.Texture, rect image.Rectangle, size int) *ebiten.Image {
	sw, sh := rect.Dx(), rect.Dy()
	if sw <= 0 || sh <= 0 {
		return nil
	}
	if (sw <= size && sh <= size) || tex.Source == nil {
		return components.SubImage(tex.Image, rect)
	}
	dw, dh := size, size
	if sw >= sh {
		dh = max(1, sh*size/sw)
	} else {
		dw = max(1, sw*size/sh)
	}

	// image.RGBA は乗算済みアルファなので、透過の境界でも色がにじまない重み付けになる
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), tex.Source, rect, draw.Src, nil)

	out := ebiten.NewImage(dw, dh)
	out.WritePixels(dst.Pix)
	return out
}
