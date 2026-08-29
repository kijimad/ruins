package hud

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
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

// Draw は武器スロットを画面上部中央に描画する
func (ws *WeaponSlots) Draw(cv ui.Canvas, data WeaponSlotsData, world w.World) {
	if len(data.Slots) == 0 {
		return
	}

	config := defaultWeaponSlotsConfig

	// 全体の幅を計算
	totalWidth := len(data.Slots)*config.SlotSize + (len(data.Slots)-1)*config.SlotSpacing

	// 画面中央に配置するためのX座標
	startX := (data.ScreenDimensions.Width - totalWidth) / 2

	// 画面上部に配置するためのY座標
	startY := config.YOffset

	sprites := world.Resources.Sprites

	// 各スロットを描画
	for i, slot := range data.Slots {
		x := startX + i*(config.SlotSize+config.SlotSpacing)
		y := startY

		// 選択中のスロットかどうか
		isSelected := i == data.SelectedSlot

		// スロットの背景を描画
		ws.drawSlotBackground(cv, x, y, config.SlotSize, isSelected)

		// 武器スプライトを描画
		drawWeaponSprite(cv, x, y, config.SlotSize, slot, sprites)

		// スロット番号を描画
		drawSlotNumber(cv, ws.face, x, y, config.SlotSize, i+1)
	}
}

// drawSlotBackground はスロット背景をNineSlice描画する
func (ws *WeaponSlots) drawSlotBackground(cv ui.Canvas, x, y, size int, selected bool) {
	r := image.Rect(x, y, x+size, y+size)
	ws.chrome.Panel(cv, r)

	// 選択中のスロットには明るい枠線を重ねる
	if selected {
		cv.StrokeRect(r, 2, theme.HUDSlotSelectedBorder)
	}
}

// drawSlotNumber はスロット番号を左上に描画
func drawSlotNumber(cv ui.Canvas, face text.Face, x, y, _ int, number int) {
	numberText := string(rune('0' + number))
	cv.DrawText(image.Pt(x+slotNumberPad, y+slotNumberPad), numberText, face, theme.TextPrimary)
}

// drawWeaponSprite は武器スプライトを中央に描画
func drawWeaponSprite(cv ui.Canvas, x, y, slotSize int, slot WeaponSlotInfo, sprites *resources.SpriteStore) {
	// 武器が装備されていない場合は何も描画しない
	if slot.WeaponName == "" {
		return
	}

	// スプライトを解決する。シートやキーが無ければ描画しない
	img := sprites.Image(&gc.SpriteRender{SpriteSheetName: slot.SpriteSheet, SpriteKey: slot.SpriteName})
	if img == nil {
		return
	}

	// スプライトをスロットの中央に置く
	b := img.Bounds()
	cv.DrawImage(image.Pt(x+(slotSize-b.Dx())/2, y+(slotSize-b.Dy())/2), img)
}
