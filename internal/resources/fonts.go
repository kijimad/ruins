package resources

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type fonts struct {
	smallFace     text.Face
	bodyFace      text.Face
	titleFontFace text.Face
}

// loadFonts は指定されたサイズでフォントフェイスを作成する
// 複数のFaceSourceを指定した場合、順番にフォールバックする
func loadFonts(sources []*text.GoTextFaceSource) *fonts {
	smallFace := loadFont(sources, 16)
	bodyFace := loadFont(sources, 20)
	titleFontFace := loadFont(sources, 32)

	return &fonts{
		smallFace:     smallFace,
		bodyFace:      bodyFace,
		titleFontFace: titleFontFace,
	}
}

func loadFont(sources []*text.GoTextFaceSource, size float64) text.Face {
	if len(sources) == 0 {
		return nil
	}

	faces := make([]text.Face, 0, len(sources))
	for _, src := range sources {
		if src != nil {
			faces = append(faces, &text.GoTextFace{
				Source: src,
				Size:   size,
			})
		}
	}

	if len(faces) == 0 {
		return nil
	}
	if len(faces) == 1 {
		return faces[0]
	}

	multiFace, err := text.NewMultiFace(faces...)
	if err != nil {
		return faces[0]
	}
	return multiFace
}
