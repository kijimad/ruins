package hud

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
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
func (c *CurrencyDisplay) Draw(cv ui.Canvas, data CurrencyData) {
	if !c.enabled {
		return
	}

	// 画面サイズを取得
	screenWidth := data.ScreenDimensions.Width
	screenHeight := data.ScreenDimensions.Height

	// 通貨テキスト
	currencyText := data.Currency.String()

	// テキストのサイズを計算
	textWidth, textHeight := ui.MeasureText(currencyText, c.face)

	// メッセージウィンドウの位置を計算
	fixedHeight := data.Config.Height()
	logAreaY := screenHeight - fixedHeight

	// ログ領域の上端の上へ置く。文字の下端が余白ぶん上に来るようにする
	currencyX := screenWidth - data.Config.LogAreaMargin - textWidth
	currencyY := logAreaY - textHeight - theme.Space4

	drawOutlinedText(cv, currencyText, c.face, image.Pt(currencyX, currencyY), theme.TextPrimary)
}
