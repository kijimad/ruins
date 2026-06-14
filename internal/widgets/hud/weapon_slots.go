package hud

import (
	"image"

	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	gc "github.com/kijimaD/ruins/internal/components"
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
	face      text.Face
	slotImage *euiimage.NineSlice // スロット背景のNineSlice画像。PanelResourcesと共有する
}

// NewWeaponSlots は新しいWeaponSlotsを作成する
func NewWeaponSlots(face text.Face, slotImage *euiimage.NineSlice) *WeaponSlots {
	return &WeaponSlots{
		face:      face,
		slotImage: slotImage,
	}
}

// Draw は武器スロットを画面上部中央に描画する
func (ws *WeaponSlots) Draw(screen *ebiten.Image, data WeaponSlotsData, world w.World) {
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

	// スプライトシートを取得
	spriteSheets := world.Resources.SpriteSheets

	// 各スロットを描画
	for i, slot := range data.Slots {
		x := startX + i*(config.SlotSize+config.SlotSpacing)
		y := startY

		// 選択中のスロットかどうか
		isSelected := i == data.SelectedSlot

		// スロットの背景を描画
		ws.drawSlotBackground(screen, x, y, config.SlotSize, isSelected)

		// 武器スプライトを描画
		drawWeaponSprite(screen, x, y, config.SlotSize, slot, spriteSheets)

		// スロット番号を描画
		drawSlotNumber(screen, ws.face, x, y, config.SlotSize, i+1)
	}
}

// drawSlotBackground はスロット背景をNineSlice描画する
func (ws *WeaponSlots) drawSlotBackground(screen *ebiten.Image, x, y, size int, selected bool) {
	if ws.slotImage == nil {
		return
	}

	ws.slotImage.Draw(screen, size, size, func(opts *ebiten.DrawImageOptions) {
		opts.GeoM.Translate(float64(x), float64(y))
	})

	// 選択中のスロットには明るい枠線を重ねて描画する
	if selected {
		vector.StrokeRect(screen, float32(x), float32(y), float32(size), float32(size), 2, theme.HUDSlotSelectedBorder, false)
	}
}

// drawSlotNumber はスロット番号を左上に描画
func drawSlotNumber(screen *ebiten.Image, face text.Face, x, y, _ int, number int) {
	numberText := string(rune('0' + number))
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x+4), float64(y+4))
	op.ColorScale.ScaleWithColor(theme.TextPrimary)
	text.Draw(screen, numberText, face, op)
}

// drawWeaponSprite は武器スプライトを中央に描画
func drawWeaponSprite(screen *ebiten.Image, x, y, slotSize int, slot WeaponSlotInfo, spriteSheets map[string]gc.SpriteSheet) {
	// 武器が装備されていない場合は何も描画しない
	if slot.WeaponName == "" {
		return
	}

	// スプライトシートを取得
	sheet, ok := spriteSheets[slot.SpriteSheet]
	if !ok {
		return
	}

	// スプライトシートの画像を取得
	spriteImage := sheet.Texture.Image
	if spriteImage == nil {
		return
	}

	// スプライトデータを取得
	spriteData, ok := sheet.Sprites[slot.SpriteName]
	if !ok {
		return
	}

	// スプライトの矩形を作成
	spriteRect := image.Rect(
		spriteData.X,
		spriteData.Y,
		spriteData.X+spriteData.Width,
		spriteData.Y+spriteData.Height,
	)

	// スプライトをスロットの中央に配置
	offsetX := float64(x) + (float64(slotSize)-float64(spriteData.Width))/2
	offsetY := float64(y) + (float64(slotSize)-float64(spriteData.Height))/2

	// 描画オプション
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(offsetX, offsetY)

	// スプライトをそのまま描画
	screen.DrawImage(spriteImage.SubImage(spriteRect).(*ebiten.Image), op)
}
