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

	// 9スライスで描く UI テクスチャ。窓・タイトルバー・入力枠・選択バー・パネルの背景に使う。
	// plain な *ebiten.Image なので外部ライブラリに依存しない
	WindowBG     *NineSliceTex
	PanelBG      *NineSliceTex
	TitleBar     *NineSliceTex
	InputBG      *NineSliceTex
	SelectionBar *NineSliceTex

	Text *TextResources
}

// NineSliceTex は9スライス描画のためのテクスチャと分割幅。BX・BY はソースの左中右・上中下の幅。
type NineSliceTex struct {
	Image *ebiten.Image
	BX    [3]int
	BY    [3]int
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

	// 窓・タイトルバー・選択バーは中央サイズから対称に分割する。ebitenui 時代の
	// loadImageNineSlice と同じ中央サイズにそろえ、見た目を保つ
	windowBG, err := newNineSliceTex("assets/graphics/list-idle-trans.png", 40, 40)
	if err != nil {
		return UIResources{}, err
	}
	panelBG, err := newNineSliceTex("assets/graphics/panel-idle.png", 40, 40)
	if err != nil {
		return UIResources{}, err
	}
	titleBar, err := newNineSliceTex("assets/graphics/titlebar-idle.png", 40, 24)
	if err != nil {
		return UIResources{}, err
	}
	selectionBar, err := newNineSliceTex("assets/graphics/selection-bar.png", 56, 10)
	if err != nil {
		return UIResources{}, err
	}
	// 入力枠は左右非対称の分割。ebitenui 時代の {9,14,6} をそのまま使う
	inputImg, err := newImageFromFile("assets/graphics/text-input-idle.png")
	if err != nil {
		return UIResources{}, err
	}
	inputBG := &NineSliceTex{Image: inputImg, BX: [3]int{9, 14, 6}, BY: [3]int{9, 14, 6}}

	return UIResources{
		Fonts: fonts,

		GradientLine: gradientLine,
		GaugeFill:    gaugeFill,

		WindowBG:     windowBG,
		PanelBG:      panelBG,
		TitleBar:     titleBar,
		InputBG:      inputBG,
		SelectionBar: selectionBar,

		Text: &TextResources{
			SmallFace:      fonts.smallFace,
			BodyFace:       fonts.bodyFace,
			KeycapFace:     fonts.keycapFace,
			TitleFontFace:  fonts.titleFontFace,
			SplashFontFace: fonts.splashFontFace,
		},
	}, nil
}
