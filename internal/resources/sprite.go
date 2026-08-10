package resources

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/components"
)

// SpriteImage は SpriteRender が指すスプライトの部分画像を返す。
// シートかキーが見つからないか、テクスチャ画像が無ければ ok=false。
// 矩形はテクスチャ範囲へクランプするので、範囲外を指す指定でも安全に切り出す。
// スプライトシートというリソースに対する解決処理なのでこのパッケージに置く
func SpriteImage(sheets map[string]components.SpriteSheet, sr *components.SpriteRender) (*ebiten.Image, bool) {
	if sr == nil {
		return nil, false
	}
	sheet, ok := sheets[sr.SpriteSheetName]
	if !ok {
		return nil, false
	}
	sprite, ok := sheet.Sprites[sr.SpriteKey]
	if !ok {
		return nil, false
	}
	if sheet.Texture.Image == nil {
		return nil, false
	}
	w := sheet.Texture.Image.Bounds().Dx()
	h := sheet.Texture.Image.Bounds().Dy()
	rect := image.Rect(
		max(0, sprite.X),
		max(0, sprite.Y),
		min(w, sprite.X+sprite.Width),
		min(h, sprite.Y+sprite.Height),
	)
	return components.SubImage(sheet.Texture.Image, rect), true
}
