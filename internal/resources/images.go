package resources

import (
	stdimage "image"
	"image/draw"
	_ "image/png" // image.Decode に PNG デコーダを登録する
	"log"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
	i, _, err := ebitenutil.NewImageFromReader(f)
	return i, err
}

func loadGraphicImages(idle string, disabled string) (*widget.GraphicImage, error) {
	idleImage, err := newImageFromFile(idle)
	if err != nil {
		return nil, err
	}

	var disabledImage *ebiten.Image
	if disabled != "" {
		disabledImage, err = newImageFromFile(disabled)
		if err != nil {
			return nil, err
		}
	}

	return &widget.GraphicImage{
		Idle:     idleImage,
		Disabled: disabledImage,
	}, nil
}

func loadImageNineSlice(path string, centerWidth int, centerHeight int) (*image.NineSlice, error) {
	i, err := newImageFromFile(path)
	if err != nil {
		return nil, err
	}
	return newNineSlice(i, centerWidth, centerHeight), nil
}

// loadImageNineSliceOpaque は PNG を読み、透明でない画素を完全不透明にしたナインスライスを返す。
// メニューをメニュー層で不透明に描くために使う。パネルのアルファはメニュー層と世界の一度きりの
// 合成に一元化するので、パネル自体は不透明にして、重なった下メニューが透けないようにする。
func loadImageNineSliceOpaque(path string, centerWidth int, centerHeight int) (*image.NineSlice, error) {
	i, err := loadOpaqueImage(path)
	if err != nil {
		return nil, err
	}
	return newNineSlice(i, centerWidth, centerHeight), nil
}

// newNineSlice は画像の中央部を指定サイズにしてナインスライスを組む。
func newNineSlice(i *ebiten.Image, centerWidth int, centerHeight int) *image.NineSlice {
	w := i.Bounds().Dx()
	h := i.Bounds().Dy()
	return image.NewNineSlice(i,
		[3]int{(w - centerWidth) / 2, centerWidth, w - (w-centerWidth)/2 - centerWidth},
		[3]int{(h - centerHeight) / 2, centerHeight, h - (h-centerHeight)/2 - centerHeight})
}

// loadOpaqueImage は PNG を CPU 側でデコードし、透明でない画素のアルファを 255 にして返す。
// 完全透明の画素は透明のまま残す。ReadPixels は起動前に呼べないため、GPU 読み戻しを避けて
// CPU の画素を加工する。NRGBA は非乗算なのでアルファを上げても見た目の色はそのまま保たれる。
func loadOpaqueImage(path string) (*ebiten.Image, error) {
	f, err := embeddedAssets.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Print(err)
		}
	}()

	src, _, err := stdimage.Decode(f)
	if err != nil {
		return nil, err
	}
	// NRGBA へ変換してから、透明でない画素のアルファだけ 255 にする。
	b := src.Bounds()
	dst := stdimage.NewNRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	for i := 3; i < len(dst.Pix); i += 4 {
		if dst.Pix[i] != 0 {
			dst.Pix[i] = 255
		}
	}

	return ebiten.NewImageFromImage(dst), nil
}
