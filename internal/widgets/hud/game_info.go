package hud

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
)

// GameInfo はHUDの基本ゲーム情報エリア
type GameInfo struct {
	bodyFace    text.Face
	headingFace text.Face // 階層表示用の大きなフォント
	enabled     bool
}

// NewGameInfo は新しいHUDGameInfoを作成する
func NewGameInfo(bodyFace text.Face, headingFace text.Face) *GameInfo {
	return &GameInfo{
		bodyFace:    bodyFace,
		headingFace: headingFace,
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

	// SP情報
	info.drawStaminaBar(screen, data.PlayerSP, data.PlayerMaxSP)

	// EP情報
	info.drawElectricityBar(screen, data.PlayerEP, data.PlayerMaxEP)

	// ターン情報
	drawOutlinedText(screen, fmt.Sprintf("turn: %d", data.TurnNumber), info.bodyFace, 0, 150, color.White)

	// 残りアクションポイント
	drawOutlinedText(screen, fmt.Sprintf("AP: %d", data.PlayerMoves), info.bodyFace, 0, 170, color.White)

	// ステータス表示（左下）
	info.drawStatusEffects(screen, data)

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

	drawOutlinedText(screen, floorText, info.headingFace, x, y, color.White)
}

// drawHealthBar はプレイヤーの体力ゲージを描画する
func (info *GameInfo) drawHealthBar(screen *ebiten.Image, currentHP, maxHP int) {
	// HPゲージの設定
	const (
		baseX    = 10.0  // 左マージン
		y        = 50.0  // 上マージン
		width    = 120.0 // ゲージの幅
		height   = 20.0  // ゲージの高さ
		labelGap = 4.0   // ラベルとゲージの間隔
	)

	// ゲージの開始位置
	gageX := float32(baseX)

	// 背景（暗い赤い領域）を描画
	vector.FillRect(screen, gageX, float32(y), float32(width), float32(height), color.RGBA{100, 0, 0, 255}, false)

	// HP比率を計算
	if maxHP > 0 {
		hpRatio := float32(currentHP) / float32(maxHP)
		if hpRatio > 1.0 {
			hpRatio = 1.0
		}
		if hpRatio < 0.0 {
			hpRatio = 0.0
		}

		// 現在のHP（緑〜赤のグラデーション）
		var barColor color.RGBA
		if hpRatio > 0.5 {
			// 緑から黄色へ（HP 50%以上）
			intensity := uint8((1.0 - hpRatio) * 2.0 * 255)
			barColor = color.RGBA{intensity, 255, 0, 255}
		} else {
			// 黄色から赤へ（HP 50%以下）
			intensity := uint8(hpRatio * 2.0 * 255)
			barColor = color.RGBA{255, intensity, 0, 255}
		}

		// 現在のHPバーを描画
		currentWidth := float32(width) * hpRatio
		vector.FillRect(screen, gageX, float32(y), currentWidth, float32(height), barColor, false)
	}

	// 数値をゲージの中央に描画
	hpText := fmt.Sprintf("%d/%d", currentHP, maxHP)
	textWidth, _ := text.Measure(hpText, info.bodyFace, 0)
	textX := float64(gageX) + float64(width)/2 - textWidth/2
	textY := y + float64(height)/2 - 6.0 // フォントサイズ16の場合の調整値
	drawOutlinedText(screen, hpText, info.bodyFace, textX, textY, color.White)
}

// drawStaminaBar はプレイヤーのスタミナポイントゲージを描画する
func (info *GameInfo) drawStaminaBar(screen *ebiten.Image, currentSP, maxSP int) {
	// SPゲージの設定
	const (
		baseX    = 10.0  // 左マージン
		y        = 76.0  // 上マージン
		width    = 120.0 // ゲージの幅
		height   = 20.0  // ゲージの高さ
		labelGap = 4.0   // ラベルとゲージの間隔
	)

	// ゲージの開始位置
	gageX := float32(baseX)

	// 背景（暗いグレー領域）を描画
	vector.FillRect(screen, gageX, float32(y), float32(width), float32(height), color.RGBA{100, 100, 100, 255}, false)

	// SP比率を計算
	if maxSP > 0 {
		spRatio := float32(currentSP) / float32(maxSP)
		if spRatio > 1.0 {
			spRatio = 1.0
		}
		if spRatio < 0.0 {
			spRatio = 0.0
		}

		var barColor color.RGBA
		if spRatio > 0.5 {
			// 明るい黄色・オレンジ（SP 50%以上）
			barColor = color.RGBA{255, 200, 0, 255}
		} else {
			// やや暗い黄色・オレンジ（SP 50%以下）
			intensity := uint8(spRatio * 2.0 * 200)
			barColor = color.RGBA{255, intensity, 0, 255}
		}

		// 現在のSPバーを描画
		currentWidth := float32(width) * spRatio
		vector.FillRect(screen, gageX, float32(y), currentWidth, float32(height), barColor, false)
	}

	// 数値をゲージの中央に描画（垂直方向にも中央配置）
	spText := fmt.Sprintf("%d/%d", currentSP, maxSP)
	textWidth, _ := text.Measure(spText, info.bodyFace, 0)
	textX := float64(gageX) + float64(width)/2 - textWidth/2
	textY := y + float64(height)/2 - 6.0 // フォントサイズ16の場合の調整値
	drawOutlinedText(screen, spText, info.bodyFace, textX, textY, color.White)
}

// drawElectricityBar はプレイヤーの電力ポイントゲージを描画する
func (info *GameInfo) drawElectricityBar(screen *ebiten.Image, currentEP, maxEP int) {
	// EPゲージの設定
	const (
		baseX    = 10.0  // 左マージン
		y        = 102.0 // 上マージン
		width    = 120.0 // ゲージの幅
		height   = 20.0  // ゲージの高さ
		labelGap = 4.0   // ラベルとゲージの間隔
	)

	// ゲージの開始位置
	gageX := float32(baseX)

	// 背景（暗い青い領域）を描画
	vector.FillRect(screen, gageX, float32(y), float32(width), float32(height), color.RGBA{0, 0, 80, 255}, false)

	// EP比率を計算
	if maxEP > 0 {
		epRatio := float32(currentEP) / float32(maxEP)
		if epRatio > 1.0 {
			epRatio = 1.0
		}
		if epRatio < 0.0 {
			epRatio = 0.0
		}

		// 現在のEP（青系のグラデーション、電力らしい色）
		var barColor color.RGBA
		if epRatio > 0.5 {
			// シアンから青へ（EP 50%以上）
			intensity := uint8((1.0 - epRatio) * 2.0 * 100)
			barColor = color.RGBA{intensity, 200, 255, 255}
		} else {
			// 青から暗い青へ（EP 50%以下）
			intensity := uint8(epRatio * 2.0 * 200)
			barColor = color.RGBA{0, intensity, 100 + uint8(epRatio*155), 255}
		}

		// 現在のEPバーを描画
		currentWidth := float32(width) * epRatio
		vector.FillRect(screen, gageX, float32(y), currentWidth, float32(height), barColor, false)
	}

	// 数値をゲージの中央に描画（垂直方向にも中央配置）
	epText := fmt.Sprintf("%d/%d", currentEP, maxEP)
	textWidth, _ := text.Measure(epText, info.bodyFace, 0)
	textX := float64(gageX) + float64(width)/2 - textWidth/2
	textY := y + float64(height)/2 - 6.0 // フォントサイズ16の場合の調整値
	drawOutlinedText(screen, epText, info.bodyFace, textX, textY, color.White)
}

// drawStatusEffects はプレイヤーのステータス効果を左下に縦に並べて描画する
func (info *GameInfo) drawStatusEffects(screen *ebiten.Image, data GameInfoData) {
	const (
		marginLeft   = 10.0 // 左マージン
		marginBottom = 10.0 // 下マージン
		lineHeight   = 20.0 // 行の高さ
	)

	// ステータス一覧を下から積み上げるように描画
	var statuses []statusDisplay

	// 空腹度をステータスとして追加（普通以外の場合のみ表示）
	if data.HungerLevel != gc.HungerNormal {
		statusColor := getHungerColor(data.HungerLevel)
		statuses = append(statuses, statusDisplay{
			text:  data.HungerLevel.String(),
			color: statusColor,
		})
	}

	// TODO(kijima): 将来的に他のステータスもここに追加する
	// 例: 濡れ、重い、など

	// メッセージエリアの高さを計算（message_area.goと同じ計算式）
	messageAreaHeight := float64(data.MessageAreaHeight)

	// メッセージエリアの上に表示するため、その分だけ上にオフセット
	screenHeight := float64(data.ScreenDimensions.Height)
	baseY := screenHeight - messageAreaHeight - marginBottom

	// 下から上に向かって描画
	for i, status := range statuses {
		// テキストサイズを測定
		textWidth, _ := text.Measure(status.text, info.bodyFace, 0)

		// 背景矩形のパディング
		paddingX := 4.0
		paddingY := 4.0

		// フォントサイズ16の実際の描画高さ
		textHeight := 16.0

		// 背景矩形の高さ（パディングを含む）
		bgHeight := float32(textHeight + paddingY*2)

		// 背景矩形のY位置（下から積み上げる）
		bgY := float32(baseY - float64(i+1)*lineHeight)

		// テキストのY位置（背景矩形の中央に配置）
		textY := float64(bgY) + paddingY

		// 背景矩形を描画
		bgX := float32(marginLeft - paddingX)
		bgWidth := float32(textWidth + paddingX*2)
		vector.FillRect(screen, bgX, bgY, bgWidth, bgHeight, status.color, false)

		// 白文字でテキストを描画
		drawOutlinedText(screen, status.text, info.bodyFace, marginLeft, textY, color.White)
	}
}

// statusDisplay はステータス表示の情報
type statusDisplay struct {
	text  string
	color color.RGBA
}

// getHungerColor は空腹度に応じた色を返す
func getHungerColor(hungerLevel gc.HungerLevel) color.RGBA {
	switch hungerLevel {
	case gc.HungerSatiated:
		return color.RGBA{100, 200, 100, 255} // 緑（満腹）
	case gc.HungerHungry:
		return color.RGBA{255, 200, 0, 255} // 黄色（空腹）
	case gc.HungerStarving:
		return color.RGBA{255, 50, 50, 255} // 赤（飢餓）
	default:
		return color.RGBA{255, 255, 255, 255} // 白（通常）
	}
}

// drawWeightDisplay はプレイヤーの所持重量を右下に描画する
func (info *GameInfo) drawWeightDisplay(screen *ebiten.Image, data GameInfoData) {
	const (
		marginRight  = 10.0 // 右マージン
		marginBottom = 10.0 // 下マージン
	)

	// 所持重量テキストを作成
	weightText := fmt.Sprintf("%.2f / %.2f kg", data.PlayerWeight, data.PlayerMaxWeight)

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
			textColor = color.RGBA{255, 50, 50, 255}
		} else if ratio > 0.8 {
			// 80%以上: 黄色
			textColor = color.RGBA{255, 200, 0, 255}
		} else {
			// 通常: 白
			textColor = color.RGBA{255, 255, 255, 255}
		}
	} else {
		textColor = color.RGBA{255, 255, 255, 255}
	}

	drawOutlinedText(screen, weightText, info.bodyFace, x, y, textColor)
}
