package resources

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// UIResources はUIリソースを管理する。フォントのフェイスと、直接描画で使う素材画像を持つ。
// ウィジェットの組み立ては internal/ui のツリーが担うので、ここは描画の土台になる
// フェイスと画像だけを提供する
type UIResources struct {
	Fonts *fonts

	GradientLine *ebiten.Image
	GaugeFill    *ebiten.Image

	Text *TextResources
}

// TextResources はテキスト描画に使うフェイス一式を管理する
type TextResources struct {
	SmallFace text.Face
	BodyFace  text.Face
	// KeycapFace はキーキャップの箱に描くグリフ専用のフェイス。本文よりひと回り大きい。
	// 大きいフェイスは行の高さも広げるため、通常の本文には使わないこと
	KeycapFace     text.Face
	TitleFontFace  text.Face
	SplashFontFace text.Face
}

// NewUIResources は新しいUIリソースを作成する。フォントと直接描画の素材だけを読み込む
func NewUIResources(sources []*text.GoTextFaceSource) (UIResources, error) {
	fonts, err := loadFonts(sources)
	if err != nil {
		return UIResources{}, err
	}

	gradientLine, err := newImageFromFile("assets/graphics/gradient-line.png")
	if err != nil {
		return UIResources{}, err
	}

	gaugeFill, err := newImageFromFile("assets/graphics/gauge-fill.png")
	if err != nil {
		return UIResources{}, err
	}

	return UIResources{
		Fonts: fonts,

		GradientLine: gradientLine,
		GaugeFill:    gaugeFill,

		Text: &TextResources{
			SmallFace:      fonts.smallFace,
			BodyFace:       fonts.bodyFace,
			KeycapFace:     fonts.keycapFace,
			TitleFontFace:  fonts.titleFontFace,
			SplashFontFace: fonts.splashFontFace,
		},
	}, nil
}
