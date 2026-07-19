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

// WindowSize はウィンドウサイズの設定
type WindowSize struct {
	Width  int
	Height int
}

// Padding は内側余白の設定
type Padding struct {
	Top    int
	Bottom int
	Left   int
	Right  int
}

// WindowStyle はウィンドウの外観設定
type WindowStyle struct {
	BackgroundColor color.Color
	BorderColor     color.Color
	BorderWidth     int
	Padding         Padding
}

// TextStyle はテキストの外観設定
type TextStyle struct {
	Color      color.RGBA
	LineHeight int
}

// ActionStyle はアクション表示の外観設定
type ActionStyle struct {
	ShowCloseButton bool
	CloseButtonText string
	ActionAreaColor color.Color
	ActionTextColor color.RGBA
}

// Config はメッセージウィンドウの設定
type Config struct {
	// レイアウト設定
	Size   WindowSize
	Center bool // 画面中央に配置するか

	// 外観設定
	WindowStyle WindowStyle
	TextStyle   TextStyle
	ActionStyle ActionStyle

	// 動作設定
	SkippableKeys  []ebiten.Key
	CloseOnClick   bool // ウィンドウ外クリックで閉じる
	ShowBackground bool // 背景オーバーレイを表示
}

// DefaultConfig はデフォルト設定を返す
func DefaultConfig() Config {
	return Config{
		Size: WindowSize{
			Width:  MinWidth,
			Height: MinHeight,
		},
		Center: true,

		WindowStyle: WindowStyle{
			BackgroundColor: theme.WindowBackground,
			BorderColor:     theme.WindowBorder,
			BorderWidth:     2,
			Padding: Padding{
				Top:    20,
				Bottom: 20,
				Left:   20,
				Right:  20,
			},
		},

		TextStyle: TextStyle{
			Color:      theme.TextPrimary,
			LineHeight: 24,
		},

		ActionStyle: ActionStyle{
			ShowCloseButton: true,
			CloseButtonText: "閉じる [Enter/Escape]",
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
