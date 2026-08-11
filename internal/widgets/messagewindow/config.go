package messagewindow

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/widgets/theme"
)

// 最小サイズの定数
const (
	MinWidth  = 600 // 最小幅
	MinHeight = 300 // 最小高さ
)

// windowSize はウィンドウサイズの設定
type windowSize struct {
	Width  int
	Height int
}

// windowPadding は内側余白の設定
type windowPadding struct {
	Top    int
	Bottom int
	Left   int
	Right  int
}

// windowStyle はウィンドウの外観設定
type windowStyle struct {
	BackgroundColor color.Color
	BorderColor     color.Color
	BorderWidth     int
	windowPadding   windowPadding
}

// textStyle はテキストの外観設定
type textStyle struct {
	Color      color.RGBA
	LineHeight int
}

// actionStyle はアクション表示の外観設定
type actionStyle struct {
	ShowCloseButton bool
	CloseButtonText string
	ActionAreaColor color.Color
	ActionTextColor color.RGBA
}

// windowConfig はメッセージウィンドウの設定
type windowConfig struct {
	// レイアウト設定
	Size   windowSize
	Center bool // 画面中央に配置するか

	// 外観設定
	windowStyle windowStyle
	textStyle   textStyle
	actionStyle actionStyle

	// 動作設定
	SkippableKeys  []ebiten.Key
	CloseOnClick   bool // ウィンドウ外クリックで閉じる
	ShowBackground bool // 背景オーバーレイを表示
}

// defaultWindowConfig はデフォルト設定を返す
func defaultWindowConfig() windowConfig {
	return windowConfig{
		Size: windowSize{
			Width:  MinWidth,
			Height: MinHeight,
		},
		Center: true,

		windowStyle: windowStyle{
			BackgroundColor: theme.WindowBackground,
			BorderColor:     theme.WindowBorder,
			BorderWidth:     2,
			windowPadding: windowPadding{
				Top:    20,
				Bottom: 20,
				Left:   20,
				Right:  20,
			},
		},

		textStyle: textStyle{
			Color:      theme.TextPrimary,
			LineHeight: 24,
		},

		actionStyle: actionStyle{
			ShowCloseButton: true,
			CloseButtonText: "Close [Enter/Escape]",
			ActionAreaColor: theme.WindowActionBg,
			ActionTextColor: theme.WindowActionText,
		},

		SkippableKeys: []ebiten.Key{
			ebiten.KeyEnter,
			ebiten.KeyEscape,
			ebiten.KeySpace,
		},
		CloseOnClick:   false,
		ShowBackground: true,
	}
}
