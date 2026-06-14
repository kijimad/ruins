package hud

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kijimaD/ruins/internal/consts"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// GameInfo はHUDの基本ゲーム情報エリア
type GameInfo struct {
	bodyFace    text.Face
	headingFace text.Face     // 階層表示用の大きなフォント
	gaugeFill   *ebiten.Image // ゲージ埋め。縦方向グラデーション
	enabled     bool
}

// NewGameInfo は新しいHUDGameInfoを作成する
func NewGameInfo(bodyFace text.Face, headingFace text.Face, gaugeFill *ebiten.Image) *GameInfo {
	return &GameInfo{
		bodyFace:    bodyFace,
		headingFace: headingFace,
		gaugeFill:   gaugeFill,
		enabled:     true,
	}
}

// Update はゲーム情報エリアを更新する
func (info *GameInfo) Update(_ w.World) {
	// 現在は更新処理なし
}

// Draw はゲーム情報エリアを描画する
func (info *GameInfo) Draw(screen *ebiten.Image, data GameInfoData) {
	if !info.enabled {
		return
	}

	// HP情報
	info.drawHealthBar(screen, data.PlayerHP, data.PlayerMaxHP)

	// 所持重量表示（右下）
	info.drawWeightDisplay(screen, data)

	// フロア情報（最後に描画して最前面に表示）
	info.drawFloorNumber(screen, data)
}

// drawFloorNumber は階層番号を描画する
func (info *GameInfo) drawFloorNumber(screen *ebiten.Image, data GameInfoData) {
	const (
		marginRight = 10.0
		marginTop   = 10.0
	)

	floorText := fmt.Sprintf("%3dF", data.FloorNumber)

	// テキストの幅を測定
	textWidth, _ := text.Measure(floorText, info.headingFace, 0)

	// 右上に配置
	x := float64(data.ScreenDimensions.Width) - textWidth - marginRight
	y := marginTop

	drawOutlinedText(screen, floorText, info.headingFace, x, y, theme.TextPrimary)
}

// ゲージ共通のレイアウト定数
const (
	gaugeBaseX      = 30.0                           // 左マージン
	gaugeBaseY      = 10.0                           // 最初のゲージの上マージン
	gaugeWidth      = 180.0                          // ゲージの幅
	gaugeBorderH    = 2.0                            // 白枠線の合計（上1 + 下1）
	gaugeFillHeight = 12.0                           // ゲージ塗り部分の高さ
	gaugeHeight     = gaugeBorderH + gaugeFillHeight // 白枠 + 塗り
	gaugeSpacing    = 4.0                            // ゲージ間の間隔
)

// drawHealthBar はプレイヤーの体力ゲージを描画する
func (info *GameInfo) drawHealthBar(screen *ebiten.Image, currentHP, maxHP int) {
	x := gaugeBaseX
	y := gaugeBaseY

	// HP比率を計算
	hpRatio := float64(0)
	if maxHP > 0 {
		hpRatio = float64(currentHP) / float64(maxHP)
		hpRatio = max(0, min(1, hpRatio))
	}

	// HP残量に応じた塗り色を決定
	var fillColor color.RGBA
	if hpRatio > 0.5 {
		intensity := uint8((1.0 - hpRatio) * 2.0 * 255)
		fillColor = color.RGBA{intensity, 255, 0, 255}
	} else {
		intensity := uint8(hpRatio * 2.0 * 255)
		fillColor = color.RGBA{255, intensity, 0, 255}
	}

	info.drawGaugeBar(screen, x, y, gaugeWidth, hpRatio, fillColor, theme.HUDGaugeBorder)
}

// セパレーターライン・枠線がゲージ塗りから左右にはみ出す量
const gaugeOverhang = 6.0

// drawGaugeBar はゲージバーを描画する。
// 上下にグラデーションセパレーターライン、その間に白枠線で囲まれたグラデーション塗りを描画する。
// セパレーターラインと枠線はゲージ塗りより左右に少しはみ出す
func (info *GameInfo) drawGaugeBar(screen *ebiten.Image, x, y, width, ratio float64, fillColor, borderColor color.RGBA) {
	fy := float32(y)
	frameX := float32(x - gaugeOverhang)
	frameW := float32(width + gaugeOverhang*2)
	fillAreaH := float32(gaugeBorderH + gaugeFillHeight)

	// 白い枠線（上辺と下辺のみ）
	vector.FillRect(screen, frameX, fy, frameW, 1, borderColor, false)
	vector.FillRect(screen, frameX, fy+fillAreaH-1, frameW, 1, borderColor, false)

	// 塗り（縦方向グラデーション: 上が明るく下が暗い光沢効果）
	if ratio > 0 && info.gaugeFill != nil {
		fillW := width * ratio
		srcH := float64(info.gaugeFill.Bounds().Dy())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(fillW, gaugeFillHeight/srcH)
		op.GeoM.Translate(x, float64(fy)+1)
		op.ColorScale.ScaleWithColor(color.NRGBA(fillColor))
		screen.DrawImage(info.gaugeFill, op)
	}
}

// drawWeightDisplay はプレイヤーの所持重量を右下に描画する
func (info *GameInfo) drawWeightDisplay(screen *ebiten.Image, data GameInfoData) {
	const (
		marginRight  = 10.0 // 右マージン
		marginBottom = 10.0 // 下マージン
	)

	// 所持重量テキストを作成
	weightText := fmt.Sprintf("%.2f / %.2f%s", data.PlayerWeight, data.PlayerMaxWeight, consts.IconKg)

	// テキストの幅を測定
	textWidth, _ := text.Measure(weightText, info.bodyFace, 0)

	// メッセージエリアの高さを取得
	messageAreaHeight := float64(data.MessageAreaHeight)

	// 画面右下に配置（メッセージエリアの上）
	screenWidth := float64(data.ScreenDimensions.Width)
	screenHeight := float64(data.ScreenDimensions.Height)
	x := screenWidth - textWidth - marginRight
	y := screenHeight - messageAreaHeight - marginBottom

	// 重量比率を計算して色を決定
	var textColor color.RGBA
	if data.PlayerMaxWeight > 0 {
		ratio := data.PlayerWeight / data.PlayerMaxWeight
		if ratio > 1.0 {
			// 超過: 赤
			textColor = theme.HUDWeightDanger
		} else if ratio > 0.8 {
			// 80%以上: 黄色
			textColor = theme.HUDWeightWarning
		} else {
			// 通常: 白
			textColor = theme.TextPrimary
		}
	} else {
		textColor = theme.TextPrimary
	}

	drawOutlinedText(screen, weightText, info.bodyFace, x, y, textColor)
}
