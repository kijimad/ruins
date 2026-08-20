package resources

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// errNoFontSource は利用可能なフォントソースが無い場合に返す
var errNoFontSource = errors.New("no available font source")

type fonts struct {
	smallFace      text.Face
	bodyFace       text.Face
	keycapFace     text.Face
	titleFontFace  text.Face
	splashFontFace text.Face
}

// loadFonts は指定されたサイズでフォントフェイスを作成する
// 複数のFaceSourceを指定した場合、順番にフォールバックする
func loadFonts(sources []*text.GoTextFaceSource) (*fonts, error) {
	smallFace, err := loadFont(sources, 13)
	if err != nil {
		return nil, fmt.Errorf("failed to load small font: %w", err)
	}
	bodyFace, err := loadFont(sources, 16)
	if err != nil {
		return nil, fmt.Errorf("failed to load body font: %w", err)
	}
	// キーキャップの箱に入れるグリフ用。アイコンフォントの plain 変種は em 内の余白が広く
	// 本文サイズでは小さく見えるため、ひと回り大きい em で描いて字面を出す
	keycapFace, err := loadFont(sources, 24)
	if err != nil {
		return nil, fmt.Errorf("failed to load keycap font: %w", err)
	}
	titleFontFace, err := loadFont(sources, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to load title font: %w", err)
	}
	splashFontFace, err := loadFont(sources, 48)
	if err != nil {
		return nil, fmt.Errorf("failed to load splash font: %w", err)
	}

	return &fonts{
		smallFace:      smallFace,
		bodyFace:       bodyFace,
		keycapFace:     keycapFace,
		titleFontFace:  titleFontFace,
		splashFontFace: splashFontFace,
	}, nil
}

// iconScale はフォールバックのアイコンフォントに掛ける倍率。キーキャップのように
// 箱の中へ字形が入るグリフは同サイズだと本文より小さく見えるため、アイコン側だけ拡大して
// 本文と釣り合わせる。文字コードも呼び出し側も変えず、フォント合成の1点で効かせる
const iconScale = 1.5

func loadFont(sources []*text.GoTextFaceSource, size float64) (text.Face, error) {
	if len(sources) == 0 {
		return nil, errNoFontSource
	}

	faces := make([]text.Face, 0, len(sources))
	for i, src := range sources {
		if src != nil {
			faceSize := size
			if i > 0 {
				faceSize = size * iconScale
			}
			faces = append(faces, &text.GoTextFace{
				Source: src,
				Size:   faceSize,
			})
		}
	}

	if len(faces) == 0 {
		return nil, errNoFontSource
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
