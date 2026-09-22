package hud

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/resources"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
)

// weaponSlotsConfig は武器スロット表示の設定
type weaponSlotsConfig struct {
	SlotSize    int // 各スロットのサイズ（ピクセル）
	SlotSpacing int // スロット間の間隔（ピクセル）
	YOffset     int // 画面上端からのオフセット（ピクセル）
}

// defaultWeaponSlotsConfig はデフォルトの武器スロット設定
var defaultWeaponSlotsConfig = weaponSlotsConfig{
	SlotSize:    48,
	SlotSpacing: 8,
	YOffset:     theme.Space4,
}

// WeaponSlots は武器スロット表示ウィジェット
type WeaponSlots struct {
	face   text.Face
	chrome Chrome
}

// NewWeaponSlots は新しいWeaponSlotsを作成する
func NewWeaponSlots(face text.Face, chrome Chrome) *WeaponSlots {
	return &WeaponSlots{
		face:   face,
		chrome: chrome,
	}
}

// slotNumberPad はスロット番号を左上へ置くときの余白
const slotNumberPad = 4

// Draw は武器スロットを画面上部中央へ横並びに描画する。中央寄せは Row の両端0幅列に委ね、
// 画面幅からの中央計算をしない。
func (ws *WeaponSlots) Draw(cv uicore.Canvas, data WeaponSlotsData, world w.World) {
	if len(data.Slots) == 0 {
		return
	}

	config := defaultWeaponSlotsConfig
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
			chrome:   ws.chrome,
			face:     ws.face,
			slot:     slot,
			number:   i + 1,
			selected: i == data.SelectedSlot,
			sprites:  sprites,
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

// slotWidget は武器スロット1枚。与えられた矩形へ背景枠を敷き、選択中なら枠線を重ね、武器スプライトを
// 中央に、番号を左上に描く。位置は Row/FlexColumn が確定し、内部は矩形基準で自己完結する。
type slotWidget struct {
	rect     image.Rectangle
	chrome   Chrome
	face     text.Face
	slot     WeaponSlotInfo
	number   int
	selected bool
	sprites  *resources.SpriteStore
}

// Layout は uicore.Widget を満たす。
func (s *slotWidget) Layout(r image.Rectangle) { s.rect = r }

// Draw は uicore.Widget を満たす。背景・選択枠・武器スプライト・番号を描く。
func (s *slotWidget) Draw(cv uicore.Canvas) {
	s.chrome.Panel(cv, s.rect)
	if s.selected {
		// 選択中のスロットには明るい枠線を重ねる
		cv.StrokeRect(s.rect, 2, theme.HUDSlotSelectedBorder)
	}
	if s.slot.WeaponName != "" {
		if img := s.sprites.Image(&gc.SpriteRender{SpriteSheetName: s.slot.SpriteSheet, SpriteKey: s.slot.SpriteName}); img != nil {
			b := img.Bounds()
			cv.DrawImage(image.Pt(s.rect.Min.X+(s.rect.Dx()-b.Dx())/2, s.rect.Min.Y+(s.rect.Dy()-b.Dy())/2), img)
		}
	}
	numberText := string(rune('0' + s.number))
	cv.DrawText(image.Pt(s.rect.Min.X+slotNumberPad, s.rect.Min.Y+slotNumberPad), numberText, s.face, theme.TextPrimary)
}

// Children は uicore.Widget を満たす。子は持たない。
func (s *slotWidget) Children() []uicore.Widget { return nil }
