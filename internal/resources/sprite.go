package resources

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/components"
)

// SpriteImage は SpriteRender が指すスプライトの部分画像を返す。
// シートかキーが見つからない、テクスチャ画像が無いなど解決できない場合は理由を error で返す。
// 矩形はテクスチャ範囲へクランプするので、範囲外を指す指定でも安全に切り出す。
// スプライトシートというリソースに対する解決処理なのでこのパッケージに置く
func SpriteImage(sheets map[string]components.SpriteSheet, sr *components.SpriteRender) (*ebiten.Image, error) {
	sheet, rect, err := spriteRect(sheets, sr)
	if err != nil {
		return nil, err
	}
	return components.SubImage(sheet.Texture.Image, rect), nil
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
