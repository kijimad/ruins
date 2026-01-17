package hud

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/widgets/messagelog"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
)

// MessageAreaConfig はメッセージエリアの設定
type MessageAreaConfig struct {
	LogAreaHeight int // ログエリアの高さ
	MaxLogLines   int // 表示する最大行数
	LogAreaMargin int // 余白
	LineHeight    int // 1行の高さ
	YPadding      int // 下端の追加パディング
}

// DefaultMessageAreaConfig はデフォルトのメッセージエリア設定
var DefaultMessageAreaConfig = MessageAreaConfig{
	LogAreaHeight: 120, // 余裕を持たせて大きめに
	MaxLogLines:   5,   // 表示する最大行数
	LogAreaMargin: 8,   // 余白
	LineHeight:    20,  // 1行の高さ
	YPadding:      8,   // 下端の追加パディング
}

// MessageArea はHUDメッセージエリア
type MessageArea struct {
	widget  *messagelog.Widget
	config  MessageAreaConfig
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
			Top:    2,
			Bottom: 2,
			Left:   2,
			Right:  2,
		},
	}

	widget := messagelog.NewWidget(widgetConfig, world)
	widget.SetStore(gamelog.FieldLog)

	return &MessageArea{
		widget:  widget,
		config:  config,
		enabled: true,
	}
}

// SetStore はログストアを設定する
func (area *MessageArea) SetStore(store *gamelog.SafeSlice) {
	if area.widget == nil {
		return
	}
	area.widget.SetStore(store)
}

// SetConfig は設定を変更する
func (area *MessageArea) SetConfig(config MessageAreaConfig) {
	area.config = config
	// 注意: messagelog.Widget に設定更新メソッドがない場合は、
	// 新しいwidgetを作成する必要があります
	// ここでは設定値を保存するだけにしています
}

// GetConfig は現在の設定を取得する
func (area *MessageArea) GetConfig() MessageAreaConfig {
	return area.config
}

// Update はメッセージエリアを更新する
func (area *MessageArea) Update(_ w.World) {
	if !area.enabled || area.widget == nil {
		return
	}

	area.widget.Update()
}

// Draw はメッセージエリアを描画する
func (area *MessageArea) Draw(screen *ebiten.Image, data MessageData) {
	if !area.enabled || area.widget == nil {
		return
	}

	// 画面サイズを取得
	screenWidth := data.ScreenDimensions.Width
	screenHeight := data.ScreenDimensions.Height

	// ログエリアの位置とサイズを計算（画面下部、横幅いっぱい）
	logAreaX := 0
	logAreaWidth := screenWidth

	// 設定を使用してサイズを計算
	fixedHeight := area.config.LogAreaMargin*2 + area.config.MaxLogLines*area.config.LineHeight + area.config.YPadding*2
	logAreaY := screenHeight - fixedHeight

	// 背景を描画
	styled.DrawFramedBackground(screen, logAreaX, logAreaY, logAreaWidth, fixedHeight, styled.DefaultMessageBackgroundStyle())

	// オフスクリーンサイズ
	offscreenWidth := logAreaWidth - area.config.LogAreaMargin*2
	offscreenHeight := fixedHeight - area.config.LogAreaMargin*2

	// メッセージウィジェットを描画
	drawX := logAreaX + area.config.LogAreaMargin
	drawY := logAreaY + area.config.LogAreaMargin
	area.widget.Draw(screen, drawX, drawY, offscreenWidth, offscreenHeight)
}
