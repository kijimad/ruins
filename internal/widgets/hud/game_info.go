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

// tempArrowSlotW は体温ゲージの左に確保する矢印スロットの幅
const tempArrowSlotW = 26

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

	// 左上ゲージ。体温矢印・体温ゲージ・HPゲージを縦に積む
	info.drawGauges(cv, data)

	// 右下スタック。周囲気温・所持重量・所持通貨をメッセージエリアの上へ下から積む
	info.drawBottomRightStack(cv, data)

	// フロア情報（最後に描画して最前面に表示）
	info.drawFloorNumber(cv, data)
}

// drawFloorNumber は階層番号を右上へ描画する。オーバーワールドでは階層の概念が無いので描かない。
// 右端揃えは Text の Align に委ね、画面幅からの逆算をしない。
func (info *GameInfo) drawFloorNumber(cv uicore.Canvas, data GameInfoData) {
	if !data.ShowFloor {
		return
	}
	floorText := fmt.Sprintf("%3dF", data.FloorNumber)
	// 高さ0の矩形は VCenter や高さ参照で壊れるのでテキスト高を持たせる
	_, h := uicore.MeasureText(floorText, info.headingFace)
	t := &uicore.Text{Value: floorText, Face: info.headingFace, Color: theme.TextPrimary, OutlineColor: theme.HUDTextOutline, Align: uicore.AlignRight}
	t.Layout(image.Rect(0, theme.Space4, data.ScreenDimensions.Width-theme.Space4, theme.Space4+h))
	t.Draw(cv)
}

// ゲージ共通のレイアウト定数
const (
	gaugeBaseX      = theme.Space4                   // 左マージン
	gaugeBaseY      = theme.Space4                   // 最初のゲージの上マージン
	gaugeWidth      = 180                            // ゲージ塗りの幅
	gaugeBorderH    = 2                              // 白枠線の合計（上1 + 下1）
	gaugeFillHeight = 12                             // ゲージ塗り部分の高さ
	gaugeHeight     = gaugeBorderH + gaugeFillHeight // 白枠 + 塗り
	gaugeSpacing    = 4                              // ゲージ間の間隔
)

// drawGauges は体温矢印・体温ゲージ・HPゲージを左上へ縦に積む。矢印スロットとゲージを横に並べ、
// 体温の下へ間隔を空けて HP を置く。位置決めは FlexColumn/Row に委ね、段や右寄せの手計算をしない。
func (info *GameInfo) drawGauges(cv uicore.Canvas, data GameInfoData) {
	tempRow := uicore.Row([]int{tempArrowSlotW, gaugeWidth},
		info.arrowWidget(data.TempArrow),
		info.bodyTempGauge(data),
	)
	hpRow := uicore.Row([]int{tempArrowSlotW, gaugeWidth},
		uicore.NewGroup(),
		info.healthGauge(data),
	)
	items := []uicore.FlexItem{
		{W: tempRow, Height: gaugeHeight},
		{Height: gaugeSpacing},
		{W: hpRow, Height: gaugeHeight},
		{Grow: true},
	}
	inner := image.Rect(gaugeBaseX, gaugeBaseY, gaugeBaseX+tempArrowSlotW+gaugeWidth, data.ScreenDimensions.Height)
	uicore.FlexColumn(inner, items)
	drawFlexItems(cv, items)
}

// arrowWidget は体温変化の矢印を返す。温まると上向き、冷えると下向き、一定は右向き。三角形でなく
// アイコンフォントの字で描き、キーキャップの記号と意匠も描画経路もそろえる。非表示なら空を返す。
func (info *GameInfo) arrowWidget(arrow TemperatureArrow) uicore.Widget {
	if !arrow.Visible {
		return uicore.NewGroup()
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
	// VCenter で体温ゲージの縦中心にそろえる
	return &uicore.Text{Value: glyph, Face: info.bodyFace, Color: arrow.Color, OutlineColor: theme.HUDTextOutline, VCenter: true}
}

// bodyTempGauge は体温ゲージを返す。満タンが平熱、減って青くなるほど冷える片方向。非表示なら空を返す。
func (info *GameInfo) bodyTempGauge(data GameInfoData) uicore.Widget {
	if !data.BodyTempVisible {
		return uicore.NewGroup()
	}
	return &gaugeWidget{fill: info.gaugeFill, ratio: data.BodyTempRatio, fillColor: bodyTempFillColor(data.BodyTempRatio), border: theme.HUDGaugeBorder}
}

// healthGauge は HP ゲージを返す。半分を境に緑から黄、黄から赤へ寄る。
func (info *GameInfo) healthGauge(data GameInfoData) uicore.Widget {
	ratio := 0.0
	if data.PlayerMaxHP > 0 {
		ratio = max(0, min(1, float64(data.PlayerHP)/float64(data.PlayerMaxHP)))
	}
	var fill color.RGBA
	if ratio > 0.5 {
		fill = theme.LerpColor(theme.HUDHealthFull, theme.HUDHealthHalf, (1.0-ratio)*2)
	} else {
		fill = theme.LerpColor(theme.HUDHealthEmpty, theme.HUDHealthHalf, ratio*2)
	}
	return &gaugeWidget{fill: info.gaugeFill, ratio: ratio, fillColor: fill, border: theme.HUDGaugeBorder}
}

// bodyTempFillColor は体温ゲージの塗り色を返す。平熱の白から、冷えるほど青へ寄る片方向
func bodyTempFillColor(ratio float64) color.RGBA {
	// 体温は片方向。0が平熱かつ上限で寒さ方向へ負に動くので、ratio=1 が平熱、下がるほど冷えの色へ寄る
	return theme.LerpColor(theme.HUDTempNeutral, theme.HUDTempCold, 1-ratio)
}

// gaugeOverhang はセパレーターライン・枠線がゲージ塗りから左右にはみ出す量
const gaugeOverhang = 6

// gaugeWidget は1本のゲージ。与えられた矩形を塗り部分とし、上下の白枠を左右へ overhang ぶんはみ出して
// 引き、比率ぶんの塗りをグラデーションのテクスチャで色掛けする。上が明るく下が暗い光沢になる。
type gaugeWidget struct {
	rect      image.Rectangle
	fill      *ebiten.Image
	ratio     float64
	fillColor color.RGBA
	border    color.RGBA
}

// Layout は uicore.Widget を満たす。
func (g *gaugeWidget) Layout(r image.Rectangle) { g.rect = r }

// Draw は uicore.Widget を満たす。上下の白枠を左右へはみ出して引き、比率ぶんの塗りを重ねる。
func (g *gaugeWidget) Draw(cv uicore.Canvas) {
	top := g.rect.Min.Y
	left := g.rect.Min.X - gaugeOverhang
	right := g.rect.Max.X + gaugeOverhang
	cv.FillRect(image.Rect(left, top, right, top+1), g.border)
	cv.FillRect(image.Rect(left, g.rect.Max.Y-1, right, g.rect.Max.Y), g.border)
	if g.ratio > 0 && g.fill != nil {
		fillW := int(float64(g.rect.Dx()) * g.ratio)
		dst := image.Rect(g.rect.Min.X, top+1, g.rect.Min.X+fillW, top+1+gaugeFillHeight)
		cv.DrawImageTintedRect(dst, g.fill, color.NRGBA(g.fillColor))
	}
}

// Children は uicore.Widget を満たす。子は持たない。
func (g *gaugeWidget) Children() []uicore.Widget { return nil }

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
		labelText := data.WeatherName + " " + data.AmbientShelterLabel + " "
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
