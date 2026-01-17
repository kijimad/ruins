package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

// CurrencyDisplay は地髄表示ウィジェット
type CurrencyDisplay struct {
	face    text.Face
	enabled bool
}

// NewCurrencyDisplay は新しいCurrencyDisplayを作成する
func NewCurrencyDisplay(face text.Face) *CurrencyDisplay {
	return &CurrencyDisplay{
		face:    face,
		enabled: true,
	}
}

// SetEnabled は表示の有効/無効を設定する
func (c *CurrencyDisplay) SetEnabled(enabled bool) {
	c.enabled = enabled
}

// Update は更新処理（現在は何もしない）
func (c *CurrencyDisplay) Update(_ w.World) {
	// 必要に応じて更新処理を追加
}

// Draw は地髄を描画する
func (c *CurrencyDisplay) Draw(screen *ebiten.Image, data CurrencyData) {
	if !c.enabled {
		return
	}

	// 画面サイズを取得
	screenWidth := data.ScreenDimensions.Width
	screenHeight := data.ScreenDimensions.Height

	// 通貨テキスト
	currencyText := worldhelper.FormatCurrency(data.Currency)

	// テキストの幅を計算
	textWidth, _ := text.Measure(currencyText, c.face, 0)

	// メッセージウィンドウの位置を計算
	fixedHeight := data.Config.LogAreaMargin*2 + data.Config.MaxLogLines*data.Config.LineHeight + data.Config.YPadding*2
	logAreaY := screenHeight - fixedHeight

	// メッセージウィンドウの上端の上に配置（右寄せ）
	currencyX := float64(screenWidth-data.Config.LogAreaMargin) - textWidth
	currencyY := float64(logAreaY - 25) // メッセージウィンドウの上に十分なスペースを取って表示

	drawOutlinedText(screen, currencyText, c.face, currencyX, currencyY, color.White)
}
