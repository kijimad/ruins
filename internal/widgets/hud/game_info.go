package hud

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// TempDirection は体温変化の向き
type TempDirection string

const (
	// TempDirectionSteady は変化がほぼ無い状態
	TempDirectionSteady TempDirection = "steady"
	// TempDirectionUp は温まっている状態
	TempDirectionUp TempDirection = "up"
	// TempDirectionDown は冷えている状態
	TempDirectionDown TempDirection = "down"
)

// TemperatureArrow は体温の変化方向を示す矢印。HP バーの左に出す。
// 温まると赤の上向き、冷えると青の下向き、一定は灰の右向き。色の濃さが変化の速さ
type TemperatureArrow struct {
	Visible   bool
	Direction TempDirection
	Color     color.RGBA
}

// 体温矢印の寸法。フォントグリフではなくベクター三角形で描くので大きさを自由に決められる
const (
	tempArrowSlotW = 26.0 // HP バーの左に確保するスロット幅
	tempArrowW     = 16.0 // 三角形の幅
	tempArrowH     = 16.0 // 三角形の高さ
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

	// 体温変化の矢印（HP バーの左）
	info.drawTemperatureArrow(screen, data.TempArrow)

	// HP情報
	info.drawHealthBar(screen, data.PlayerHP, data.PlayerMaxHP)

	// 所持重量表示（右下）
	info.drawWeightDisplay(screen, data)

	// フロア情報（最後に描画して最前面に表示）
	info.drawFloorNumber(screen, data)
}

// drawFloorNumber は階層番号を描画する
func (info *GameInfo) drawFloorNumber(screen *ebiten.Image, data GameInfoData) {
	floorText := fmt.Sprintf("%3dF", data.FloorNumber)

	// テキストの幅を測定
	textWidth, _ := text.Measure(floorText, info.headingFace, 0)

	// 右上に配置
	x := float64(data.ScreenDimensions.Width) - textWidth - theme.Space4F
	y := theme.Space4F

	drawOutlinedText(screen, floorText, info.headingFace, x, y, theme.TextPrimary)
}

// ゲージ共通のレイアウト定数
const (
	gaugeBaseX      = theme.Space4F                  // 左マージン
	gaugeBaseY      = theme.Space4F                  // 最初のゲージの上マージン
	gaugeWidth      = 180.0                          // ゲージの幅
	gaugeBorderH    = 2.0                            // 白枠線の合計（上1 + 下1）
	gaugeFillHeight = 12.0                           // ゲージ塗り部分の高さ
	gaugeHeight     = gaugeBorderH + gaugeFillHeight // 白枠 + 塗り
	gaugeSpacing    = 4.0                            // ゲージ間の間隔
)

// arrowOutlineOffsets は三角形の縁取りを描く8方向のオフセット。OutlinedText と同じ考え方
var arrowOutlineOffsets = [8][2]float32{
	{-1, -1}, {0, -1}, {1, -1},
	{-1, 0}, {1, 0},
	{-1, 1}, {0, 1}, {1, 1},
}

// drawTemperatureArrow は体温変化の矢印を HP バーの左にベクター三角形で描く。
// 温まると上向き、冷えると下向き、一定は右向き。多様な背景でも読めるよう縁取りを重ねる
func (info *GameInfo) drawTemperatureArrow(screen *ebiten.Image, arrow TemperatureArrow) {
	if !arrow.Visible {
		return
	}

	// HP ゲージの縦中心にそろえる
	left := float32(gaugeBaseX)
	cy := float32(gaugeBaseY) + float32(gaugeHeight)/2
	top := cy - tempArrowH/2
	bottom := cy + tempArrowH/2
	midX := left + tempArrowW/2
	right := left + tempArrowW

	var pts [3][2]float32
	switch arrow.Direction {
	case TempDirectionUp:
		pts = [3][2]float32{{midX, top}, {left, bottom}, {right, bottom}}
	case TempDirectionDown:
		pts = [3][2]float32{{midX, bottom}, {left, top}, {right, top}}
	default:
		// 一定は右向き
		pts = [3][2]float32{{right, cy}, {left, top}, {left, bottom}}
	}

	// 縁取りを先に8方向へずらして描き、上に本体色を重ねる
	for _, o := range arrowOutlineOffsets {
		fillTriangle(screen, offsetTriangle(pts, o[0], o[1]), theme.HUDTextOutline)
	}
	fillTriangle(screen, pts, arrow.Color)
}

// offsetTriangle は三角形の全頂点を平行移動した新しい三角形を返す
func offsetTriangle(pts [3][2]float32, dx, dy float32) [3][2]float32 {
	for i := range pts {
		pts[i][0] += dx
		pts[i][1] += dy
	}
	return pts
}

// fillTriangle は3頂点の塗り三角形を描く
func fillTriangle(screen *ebiten.Image, pts [3][2]float32, clr color.Color) {
	var p vector.Path
	p.MoveTo(pts[0][0], pts[0][1])
	p.LineTo(pts[1][0], pts[1][1])
	p.LineTo(pts[2][0], pts[2][1])
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(screen, &p, nil, op)
}

// drawHealthBar はプレイヤーの体力ゲージを描画する

func (info *GameInfo) drawHealthBar(screen *ebiten.Image, currentHP, maxHP int) {
	// 矢印スロットぶん右へ寄せる
	x := gaugeBaseX + tempArrowSlotW
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
	// 所持重量テキストを作成する
	weightText := fmt.Sprintf("%s / %s", data.PlayerWeight.KgString(), data.PlayerMaxWeight.KgString())

	// テキストの幅を測定
	textWidth, textHeight := text.Measure(weightText, info.bodyFace, 0)

	// メッセージエリアの高さを取得
	messageAreaHeight := float64(data.MessageAreaHeight)

	// 画面右下に配置（通貨表示の上に重ならないように2行分上げる）
	screenWidth := float64(data.ScreenDimensions.Width)
	screenHeight := float64(data.ScreenDimensions.Height)
	x := screenWidth - textWidth - theme.Space4F
	y := screenHeight - messageAreaHeight - theme.Space4F - textHeight*2 - theme.Space2F

	// 重量比率を計算して色を決定
	var textColor color.RGBA
	if data.PlayerMaxWeight > 0 {
		ratio := float64(data.PlayerWeight) / float64(data.PlayerMaxWeight)
		switch {
		case ratio > 1.0:
			// 超過: 赤
			textColor = theme.HUDWeightDanger
		case ratio > 0.8:
			// 80%以上: 黄色
			textColor = theme.HUDWeightWarning
		default:
			// 通常: 白
			textColor = theme.TextPrimary
		}
	} else {
		textColor = theme.TextPrimary
	}

	drawOutlinedText(screen, weightText, info.bodyFace, x, y, textColor)
}
