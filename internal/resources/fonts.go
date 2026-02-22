package resources

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type fonts struct {
	smallFace     text.Face
	bodyFace      text.Face
	titleFontFace text.Face
}

func loadFonts(tfs *text.GoTextFaceSource) *fonts {
	smallFace := loadFont(tfs, 16)
	bodyFace := loadFont(tfs, 20)
	titleFontFace := loadFont(tfs, 32)

	return &fonts{
		smallFace:     smallFace,
		bodyFace:      bodyFace,
		titleFontFace: titleFontFace,
	}
}

func loadFont(tfs *text.GoTextFaceSource, size float64) text.Face {
	return &text.GoTextFace{
		Source: tfs,
		Size:   size,
	}
}
