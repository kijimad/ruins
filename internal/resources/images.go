package resources

import (
	"image"
	"log"

	// UI テクスチャは PNG。image.Decode が読めるよう復号器を登録する
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

func newImageFromFile(path string) (*ebiten.Image, error) {
	f, err := embeddedAssets.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			// ログ出力するが、エラーは元の処理に影響させない
			// この関数は読み取り専用なので、Close失敗は通常問題ない
			log.Print(err)
		}
	}()
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	// アトラス配置に画素が左右されないよう unmanaged で作る。機構は components.Texture を参照
	return ebiten.NewImageFromImageWithOptions(src, &ebiten.NewImageFromImageOptions{Unmanaged: true}), nil
}

// newNineSliceTex はテクスチャを読み込み、中央サイズから9スライスの分割幅を導いて返す。
// 両端の幅は画像サイズと中央サイズの差を半分ずつに割り、端数は右下側へ寄せる
func newNineSliceTex(path string, centerW, centerH int) (*NineSliceTex, error) {
	img, err := newImageFromFile(path)
	if err != nil {
		return nil, err
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	return &NineSliceTex{
		Image: img,
		BX:    [3]int{(w - centerW) / 2, centerW, w - (w-centerW)/2 - centerW},
		BY:    [3]int{(h - centerH) / 2, centerH, h - (h-centerH)/2 - centerH},
	}, nil
}
