package resources

import (
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
	w := i.Bounds().Dx()
	h := i.Bounds().Dy()
	return image.NewNineSlice(i,
			[3]int{(w - centerWidth) / 2, centerWidth, w - (w-centerWidth)/2 - centerWidth},
			[3]int{(h - centerHeight) / 2, centerHeight, h - (h-centerHeight)/2 - centerHeight}),
		nil
}
