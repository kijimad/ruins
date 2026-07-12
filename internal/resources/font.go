package resources

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/assets"
)

// Font はロード済みのフォント
type Font struct {
	Font       text.Face
	FaceSource *text.GoTextFaceSource // コピーが禁止されていて参照渡ししかできない
}

// NewFont はフォントパスからフォントを読み込む
func NewFont(fontPath string) (Font, error) {
	fontFile, err := assets.FS.Open(fontPath)
	if err != nil {
		return Font{}, err
	}

	s, err := text.NewGoTextFaceSource(fontFile)
	if err != nil {
		return Font{}, err
	}

	return Font{
		FaceSource: s,
		Font:       &text.GoTextFace{Source: s, Size: 24},
	}, nil
}
