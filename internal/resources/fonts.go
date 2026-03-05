package resources

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type fonts struct {
	smallFace     text.Face
	bodyFace      text.Face
	titleFontFace text.Face
}

// loadFonts は指定されたサイズでフォントフェイスを作成する
// 複数のFaceSourceを指定した場合、順番にフォールバックする
func loadFonts(sources []*text.GoTextFaceSource) (*fonts, error) {
	smallFace, err := loadFont(sources, 16)
	if err != nil {
		return nil, fmt.Errorf("failed to load small font: %w", err)
	}
	bodyFace, err := loadFont(sources, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to load body font: %w", err)
	}
	titleFontFace, err := loadFont(sources, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to load title font: %w", err)
	}

	return &fonts{
		smallFace:     smallFace,
		bodyFace:      bodyFace,
		titleFontFace: titleFontFace,
	}, nil
}

func loadFont(sources []*text.GoTextFaceSource, size float64) (text.Face, error) {
	if len(sources) == 0 {
		return nil, nil
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
		return nil, nil
	}
	if len(faces) == 1 {
		return faces[0], nil
	}

	multiFace, err := text.NewMultiFace(faces...)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi face: %w", err)
	}
	return multiFace, nil
}
