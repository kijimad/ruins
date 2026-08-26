package resources

import (
	"log"

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
