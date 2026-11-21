package hud

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/resources"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
)

// handSlotsConfig は手持ち欄表示の設定
type handSlotsConfig struct {
	SlotSize    int // 各スロットのサイズ（ピクセル）
	SlotSpacing int // スロット間の間隔（ピクセル）
	YOffset     int // 画面上端からのオフセット（ピクセル）
}

// HandSlots は手持ち欄表示ウィジェット
type HandSlots struct {
	face      text.Face
	chrome    Chrome
	gaugeFill *ebiten.Image // 残量のゲージの塗りのテクスチャ
}

// NewHandSlots は新しいHandSlotsを作成する
func NewHandSlots(face text.Face, gaugeFill *ebiten.Image, chrome Chrome) *HandSlots {
	return &HandSlots{
		face:      face,
		chrome:    chrome,
		gaugeFill: gaugeFill,
	}
}

// slotNumberPad はスロット番号を左上へ置くときの余白
const slotNumberPad = 4

// Draw は手持ち欄を画面上部中央へ横並びに描画する。中央寄せは Row の両端0幅列に委ね、
// 画面幅からの中央計算をしない。
func (hs *HandSlots) Draw(cv uicore.Canvas, data HandSlotsData, world w.World) {
	if len(data.Slots) == 0 {
		return
	}

	config := handSlotsConfig{SlotSize: 48, SlotSpacing: 8, YOffset: theme.Space4}
	sprites := world.Resources.Sprites

	// 両端0幅列で中央寄せし、スロットとスロット間隔を交互に並べる
	widths := make([]int, 0, len(data.Slots)*2+1)
	cells := make([]uicore.Widget, 0, len(data.Slots)*2+1)
	widths = append(widths, 0)
	cells = append(cells, uicore.NewGroup())
	for i, slot := range data.Slots {
		if i > 0 {
			widths = append(widths, config.SlotSpacing)
			cells = append(cells, uicore.NewGroup())
		}
		widths = append(widths, config.SlotSize)
		cells = append(cells, &slotWidget{
			chrome:    hs.chrome,
			face:      hs.face,
			slot:      slot,
			number:    i + 1,
			selected:  slot.SlotNumber == data.SelectedSlot,
			sprites:   sprites,
			gaugeFill: hs.gaugeFill,
		})
	}
	widths = append(widths, 0)
	cells = append(cells, uicore.NewGroup())

	row := uicore.Row(widths, cells...)
	items := []uicore.FlexItem{{W: row, Height: config.SlotSize}, {Grow: true}}
	inner := image.Rect(0, config.YOffset, data.ScreenDimensions.Width, data.ScreenDimensions.Height)
	uicore.FlexColumn(inner, items)
	drawFlexItems(cv, items)
}

// slotWidget は手持ち欄1枚。与えられた矩形へ背景枠を敷き、選択中なら枠線を重ね、武器スプライトを
// 中央に、番号を左上に描く。位置は Row/FlexColumn が確定し、内部は矩形基準で自己完結する。
type slotWidget struct {
	rect     image.Rectangle
	chrome   Chrome
	face     text.Face
	slot     HandSlotInfo
	number   int
	selected bool
	sprites  *resources.SpriteStore
	// gaugeFill は残量のゲージの塗りのテクスチャ
	gaugeFill *ebiten.Image
}

// Layout は uicore.Widget を満たす。
func (s *slotWidget) Layout(r image.Rectangle) { s.rect = r }

// Draw は uicore.Widget を満たす。背景・選択枠・武器スプライト・番号を描く。
func (s *slotWidget) Draw(cv uicore.Canvas) {
	s.chrome.Panel(cv, s.rect)
	if s.selected {
		// 選択中のスロットには明るい枠線を重ねる
		cv.StrokeRect(s.rect, 2, theme.HUDSlotSelectedBorder, uicore.RectOptions{Radius: theme.CornerRadius})
	}
	if s.slot.WeaponName != "" {
		if img := s.sprites.Image(&gc.SpriteRender{SpriteSheetName: s.slot.SpriteSheet, SpriteKey: s.slot.SpriteName}); img != nil {
			b := img.Bounds()
			cv.DrawImage(image.Pt(s.rect.Min.X+(s.rect.Dx()-b.Dx())/2, s.rect.Min.Y+(s.rect.Dy()-b.Dy())/2), img)
		}
	}
	if s.slot.HasCharge {
		s.drawGauge(cv)
	}
	numberText := fmt.Sprint(s.number)
	cv.DrawText(image.Pt(s.rect.Min.X+slotNumberPad, s.rect.Min.Y+slotNumberPad), numberText, s.face, theme.TextPrimary)
}

// 手持ち欄の残量のゲージの高さと角の丸み
const (
	slotGaugeHeight = 6
	slotGaugeRadius = 2
)

// drawGauge は欄の下端に残量のゲージを描く。HUD のほかのゲージと同じ部品で描く
func (s *slotWidget) drawGauge(cv uicore.Canvas) {
	inner := s.rect.Inset(slotNumberPad)
	g := &uicore.Gauge{Fill: s.gaugeFill, Ratio: s.slot.ChargeRatio, FillColor: theme.HealthGaugeColor(s.slot.ChargeRatio), Border: theme.HUDGaugeBorder, Radius: slotGaugeRadius}
	g.Layout(image.Rect(inner.Min.X, inner.Max.Y-slotGaugeHeight, inner.Max.X, inner.Max.Y))
	g.Draw(cv)
}

// Children は uicore.Widget を満たす。子は持たない。
func (s *slotWidget) Children() []uicore.Widget { return nil }
