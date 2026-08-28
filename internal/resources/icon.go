package resources

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/draw"

	"github.com/kijimaD/ruins/internal/components"
)

// IconStore は一覧に並べるスプライトを表示サイズまで縮めた画像で保持する。
//
// GPU の端数倍率の縮小はテクセル境界の読みが揺れて決定的でないため、
// 縮小は CPU で一度だけ行い、描画側は等倍で置くだけにする。
//
// world ごとに持つ。グローバルな可変状態を作らず、並行して走るテストが互いのアイコンを
// 混ぜない
type IconStore struct {
	images map[iconKey]*ebiten.Image
}

// iconKey は縮小結果を一意に決める組。同じスプライトでも表示サイズが違えば別の画になる
type iconKey struct {
	sheet  string
	sprite string
	size   int
}

// NewIconStore は空のアイコン置き場を作る
func NewIconStore() *IconStore {
	return &IconStore{images: map[iconKey]*ebiten.Image{}}
}

// Sized は sr が指すスプライトを size の正方へ収まるまで縦横比を保って縮めた画像を返す。
// 解決できないスプライトには nil を返し、アイコンを持たない行として描かせる
func (s *IconStore) Sized(sheets map[string]components.SpriteSheet, sr *components.SpriteRender, size int) *ebiten.Image {
	if sr == nil || size <= 0 {
		return nil
	}
	key := iconKey{sheet: sr.SpriteSheetName, sprite: sr.SpriteKey, size: size}
	if img, ok := s.images[key]; ok {
		return img
	}
	sheet, rect, err := spriteRect(sheets, sr)
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

// shrinkToFit は tex の rect の部分を size の正方へ収まるまで縦横比を保って縮める。
// 拡大はしないので、すでに収まっている部分はテクスチャの部分画像をそのまま返す
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
