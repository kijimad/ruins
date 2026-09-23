package hud

import (
	"image"

	"github.com/kijimaD/ruins/internal/widgets/messagelog"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
)

// MessageAreaConfig はメッセージエリアの設定
type MessageAreaConfig struct {
	MaxLogLines   int // 表示する最大行数
	LogAreaMargin int // 余白
	LineHeight    int // 1行の高さ
	YPadding      int // 下端の追加パディング
}

// Height はログ領域の描画高さを返す。上下の余白と、最大行数ぶんの行高から決まる。
// ログの上端を基準に置く側は、この高さを画面高から引いて求める
func (c MessageAreaConfig) Height() int {
	return c.LogAreaMargin*2 + c.MaxLogLines*c.LineHeight + c.YPadding*2
}

// DefaultMessageAreaConfig はデフォルトのメッセージエリア設定
var DefaultMessageAreaConfig = MessageAreaConfig{
	MaxLogLines:   5,            // 表示する最大行数
	LogAreaMargin: theme.Space3, // 余白
	LineHeight:    LineH,        // 1行の高さ。世界の上に重ねるパネルと揃える
	YPadding:      12,           // 下端の追加パディング
}

// MessageArea はHUDメッセージエリア
type MessageArea struct {
	widget  *messagelog.Widget
	config  MessageAreaConfig
	chrome  Chrome
	enabled bool
}

// NewMessageArea はデフォルト設定でHUDMessageAreaを作成する
func NewMessageArea(world w.World) *MessageArea {
	config := DefaultMessageAreaConfig

	widgetConfig := messagelog.WidgetConfig{
		MaxLines:   config.MaxLogLines,
		LineHeight: config.LineHeight,
		Spacing:    3,
		Padding: messagelog.Insets{
			Top:    theme.Space2,
			Bottom: theme.Space2,
			Left:   theme.Space3,
			Right:  theme.Space3,
		},
	}

	widget := messagelog.NewWidget(widgetConfig, world)

	return &MessageArea{
		widget:  widget,
		config:  config,
		chrome:  Chrome{},
		enabled: true,
	}
}

// Update はメッセージエリアを更新する
func (area *MessageArea) Update() {
	if !area.enabled || area.widget == nil {
		return
	}

	area.widget.Update()
}

// Draw はメッセージエリアを画面下部へ左右マージン付きで描画する。下端固定は FlexColumn の先頭 Grow
// スペーサに委ね、画面高からの逆算をしない。
func (area *MessageArea) Draw(cv uicore.Canvas, data MessageData) {
	if !area.enabled || area.widget == nil {
		return
	}

	boxMargin := theme.Space3
	panel := &messagePanelWidget{chrome: area.chrome, widget: area.widget, margin: area.config.LogAreaMargin}
	items := []uicore.FlexItem{
		{Grow: true},
		{W: panel, Height: area.config.Height()},
		{Height: boxMargin},
	}
	inner := image.Rect(boxMargin, 0, data.ScreenDimensions.Width-boxMargin, data.ScreenDimensions.Height)
	uicore.FlexColumn(inner, items)
	drawFlexItems(cv, items)
}

// messagePanelWidget はメッセージログのパネル。与えられた矩形へ背景枠を敷き、内側余白ぶん縮めた領域へ
// ログ本体を描く。パネルの画面内配置は FlexColumn へ委ね、内部は矩形基準で自己完結する。
type messagePanelWidget struct {
	rect   image.Rectangle
	chrome Chrome
	widget *messagelog.Widget
	margin int
}

// Layout は uicore.Widget を満たす。
func (m *messagePanelWidget) Layout(r image.Rectangle) { m.rect = r }

// Draw は uicore.Widget を満たす。背景枠と、内側へ寄せたログ本体を描く。
func (m *messagePanelWidget) Draw(cv uicore.Canvas) {
	m.chrome.Panel(cv, m.rect)
	m.widget.Draw(cv, m.rect.Min.X+m.margin, m.rect.Min.Y+m.margin, m.rect.Dx()-m.margin*2, m.rect.Dy()-m.margin*2)
}

// Children は uicore.Widget を満たす。子は持たない。
func (m *messagePanelWidget) Children() []uicore.Widget { return nil }
