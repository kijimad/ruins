package hud

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/consts"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
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

// TemperatureArrow は体温の変化方向を示す矢印。体温ゲージの左に出す。
// 温まると赤の上向き、冷えると青の下向き、一定は黄の右向き。色の濃さが変化の速さ
type TemperatureArrow struct {
	Visible   bool
	Direction TempDirection
	Color     color.RGBA
}

// 体温矢印の寸法
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
func (info *GameInfo) Draw(cv uicore.Canvas, data GameInfoData) {
	if !info.enabled {
		return
	}

	info.drawTemperatureArrow(cv, data.TempArrow)

	// 体温ゲージ。矢印の隣となる最上段
	info.drawBodyTemperature(cv, data)

	// HPゲージ。体温ゲージの下段
	info.drawHealthBar(cv, data.PlayerHP, data.PlayerMaxHP)

	// 右下スタック。周囲気温・所持重量・所持通貨をメッセージエリアの上へ下から積む
	info.drawBottomRightStack(cv, data)

	// フロア情報（最後に描画して最前面に表示）
	info.drawFloorNumber(cv, data)
}

// drawFloorNumber は階層番号を描画する。オーバーワールドでは階層の概念が無いので描かない
func (info *GameInfo) drawFloorNumber(cv uicore.Canvas, data GameInfoData) {
	if !data.ShowFloor {
		return
	}
	floorText := fmt.Sprintf("%3dF", data.FloorNumber)

	// テキストの幅を測定
	textWidth := uicore.MeasureTextWidth(floorText, info.headingFace)

	// 右上に配置
	x := data.ScreenDimensions.Width - textWidth - theme.Space4
	drawOutlinedText(cv, floorText, info.headingFace, image.Pt(x, theme.Space4), theme.TextPrimary)
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

// drawTemperatureArrow は体温変化の矢印を体温ゲージの左に描く。
// 温まると上向き、冷えると下向き、一定は右向き。
//
// 三角形を組むのでなくアイコンフォントの字で描く。キーキャップの記号と同じ出どころにすれば、
// 矢印の意匠が字体の差し替えで揃い、描画も文字と同じ経路に乗る
func (info *GameInfo) drawTemperatureArrow(cv uicore.Canvas, arrow TemperatureArrow) {
	if !arrow.Visible {
		return
	}

	var glyph string
	switch arrow.Direction {
	case TempDirectionUp:
		glyph = consts.IconArrowUp
	case TempDirectionDown:
		glyph = consts.IconArrowDown
	case TempDirectionSteady:
		glyph = consts.IconArrowRight
	}

	// 体温ゲージの縦中心にそろえる
	_, gh := uicore.MeasureText(glyph, info.bodyFace)
	y := int(gaugeBaseY+gaugeHeight/2) - gh/2
	drawOutlinedText(cv, glyph, info.bodyFace, image.Pt(int(gaugeBaseX), y), arrow.Color)
}

// drawBodyTemperature は体温ゲージを最上段に描く。体温は片方向で、満タンが平熱、減って青くなるほど冷える
func (info *GameInfo) drawBodyTemperature(cv uicore.Canvas, data GameInfoData) {
	if !data.BodyTempVisible {
		return
	}
	x := gaugeBaseX + tempArrowSlotW
	y := gaugeBaseY
	info.drawGaugeBar(cv, x, y, gaugeWidth, data.BodyTempRatio, bodyTempFillColor(data.BodyTempRatio), theme.HUDGaugeBorder)
}

// bodyTempFillColor は体温ゲージの塗り色を返す。平熱の白から、冷えるほど青へ寄る片方向
func bodyTempFillColor(ratio float64) color.RGBA {
	// 体温は片方向。0が平熱かつ上限で寒さ方向へ負に動くので、ratio=1 が平熱、下がるほど冷えの色へ寄る
	return lerpColor(theme.HUDTempNeutral, theme.HUDTempCold, 1-ratio)
}

// lerpColor は2色を t (0..1) で線形補間する。ゲージの塗りを比率で連続に変えるのに使う
func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	lerp := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), 255}
}

// drawHealthBar はプレイヤーの体力ゲージを描画する
func (info *GameInfo) drawHealthBar(cv uicore.Canvas, currentHP, maxHP int) {
	// 矢印スロットぶん右へ寄せる
	x := gaugeBaseX + tempArrowSlotW
	y := gaugeBaseY + gaugeHeight + gaugeSpacing

	// HP比率を計算
	hpRatio := float64(0)
	if maxHP > 0 {
		hpRatio = float64(currentHP) / float64(maxHP)
		hpRatio = max(0, min(1, hpRatio))
	}

	// HP残量に応じた塗り色を決定する。半分を境に、緑から黄、黄から赤へ寄る
	var fillColor color.RGBA
	if hpRatio > 0.5 {
		fillColor = lerpColor(theme.HUDHealthFull, theme.HUDHealthHalf, (1.0-hpRatio)*2)
	} else {
		fillColor = lerpColor(theme.HUDHealthEmpty, theme.HUDHealthHalf, hpRatio*2)
	}

	info.drawGaugeBar(cv, x, y, gaugeWidth, hpRatio, fillColor, theme.HUDGaugeBorder)
}

// セパレーターライン・枠線がゲージ塗りから左右にはみ出す量
const gaugeOverhang = 6.0

// drawGaugeBar はゲージバーを描画する。
// 上下にグラデーションセパレーターライン、その間に白枠線で囲まれたグラデーション塗りを描画する。
// セパレーターラインと枠線はゲージ塗りより左右に少しはみ出す
func (info *GameInfo) drawGaugeBar(cv uicore.Canvas, x, y, width, ratio float64, fillColor, borderColor color.RGBA) {
	top := int(y)
	frameX := int(x - gaugeOverhang)
	frameW := int(width + gaugeOverhang*2)
	fillAreaH := int(gaugeBorderH + gaugeFillHeight)

	// 白い枠線は上辺と下辺だけ引く
	cv.FillRect(image.Rect(frameX, top, frameX+frameW, top+1), borderColor)
	cv.FillRect(image.Rect(frameX, top+fillAreaH-1, frameX+frameW, top+fillAreaH), borderColor)

	// 塗りは縦のグラデーションのテクスチャを伸ばして色を掛ける。上が明るく下が暗い光沢になる
	if ratio > 0 && info.gaugeFill != nil {
		fillW := int(width * ratio)
		dst := image.Rect(int(x), top+1, int(x)+fillW, top+1+int(gaugeFillHeight))
		cv.DrawImageTintedRect(dst, info.gaugeFill, color.NRGBA(fillColor))
	}
}

// drawBottomRightStack は周囲気温・所持重量・所持通貨を右下、メッセージエリアの上へ下から積んで描く。
// 画面端からの逆算や段の手積みをせず、下寄せの FlexColumn へ流して各行を右端に揃える。先頭の Grow
// スペーサが上の余りを吸収して全体を下端へ押し、末尾スペーサがメッセージエリアの高さを確保する。
// 温度は場所依存なので、屋内へ入る・火に近づく効果がその場で読める。囲われラベルは白、温度は快適帯の
// 内外の色、重量は積載率で色を変える。
func (info *GameInfo) drawBottomRightStack(cv uicore.Canvas, data GameInfoData) {
	face := info.bodyFace
	outline := theme.HUDTextOutline

	items := []uicore.FlexItem{{Grow: true}}

	if data.AmbientTempVisible {
		labelText := data.AmbientShelterLabel + " "
		tempText := fmt.Sprintf("%d%s", data.AmbientTemp, consts.IconDegree)
		labelW, h := uicore.MeasureText(labelText, face)
		tempW, _ := uicore.MeasureText(tempText, face)
		// 先頭0幅列が余り幅を吸収し、ラベルと温度を右端へ寄せる
		row := uicore.Row([]int{0, labelW, tempW},
			uicore.NewGroup(),
			&uicore.Text{Value: labelText, Face: face, Color: theme.TextPrimary, OutlineColor: outline},
			&uicore.Text{Value: tempText, Face: face, Color: data.AmbientTempColor, OutlineColor: outline},
		)
		items = append(items, uicore.FlexItem{W: row, Height: h}, uicore.FlexItem{Height: theme.Space2})
	}

	weightText := fmt.Sprintf("%s / %s", data.PlayerWeight.KgString(), data.PlayerMaxWeight.KgString())
	_, wh := uicore.MeasureText(weightText, face)
	items = append(items,
		uicore.FlexItem{W: &uicore.Text{Value: weightText, Face: face, Color: weightColor(data), OutlineColor: outline, Align: uicore.AlignRight}, Height: wh},
		uicore.FlexItem{Height: theme.Space2},
	)

	currencyText := data.Currency.String()
	_, ch := uicore.MeasureText(currencyText, face)
	items = append(items, uicore.FlexItem{W: &uicore.Text{Value: currencyText, Face: face, Color: theme.TextPrimary, OutlineColor: outline, Align: uicore.AlignRight}, Height: ch})

	// スタックの下端をメッセージログ枠の上端の1余白ぶん上へ置く
	items = append(items, uicore.FlexItem{Height: data.MessageAreaHeight + theme.Space4})

	inner := image.Rect(0, 0, data.ScreenDimensions.Width-theme.Space4, data.ScreenDimensions.Height)
	uicore.FlexColumn(inner, items)
	drawFlexItems(cv, items)
}

// weightColor は所持重量の文字色を返す。積載超過は赤、8割超は黄、通常は白。
func weightColor(data GameInfoData) color.RGBA {
	if data.PlayerMaxWeight <= 0 {
		return theme.TextPrimary
	}
	switch ratio := float64(data.PlayerWeight) / float64(data.PlayerMaxWeight); {
	case ratio > 1.0:
		return theme.HUDWeightDanger
	case ratio > 0.8:
		return theme.HUDWeightWarning
	default:
		return theme.TextPrimary
	}
}

// drawFlexItems は FlexColumn/Row で矩形が確定済みの各行 Widget を描く。W が nil のスペーサ行は飛ばす。
func drawFlexItems(cv uicore.Canvas, items []uicore.FlexItem) {
	for _, it := range items {
		if it.W != nil {
			it.W.Draw(cv)
		}
	}
}
