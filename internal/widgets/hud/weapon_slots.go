package hud

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
)

// WeaponSlotsConfig は武器スロット表示の設定
type WeaponSlotsConfig struct {
	SlotSize    int // 各スロットのサイズ（ピクセル）
	SlotSpacing int // スロット間の間隔（ピクセル）
	BorderWidth int // 枠線の幅（ピクセル）
	YOffset     int // 画面上端からのオフセット（ピクセル）
}

// DefaultWeaponSlotsConfig はデフォルトの武器スロット設定
var DefaultWeaponSlotsConfig = WeaponSlotsConfig{
	SlotSize:    48,
	SlotSpacing: 8,
	BorderWidth: 2,
	YOffset:     20,
}

// WeaponSlots は武器スロット表示ウィジェット
type WeaponSlots struct {
	face text.Face
}

// NewWeaponSlots は新しいWeaponSlotsを作成する
func NewWeaponSlots(face text.Face) *WeaponSlots {
	return &WeaponSlots{
		face: face,
	}
}

// Draw は武器スロットを画面上部中央に描画する
func (ws *WeaponSlots) Draw(screen *ebiten.Image, data WeaponSlotsData, world w.World) {
	if len(data.Slots) == 0 {
		return
	}

	config := DefaultWeaponSlotsConfig

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
		drawSlotBackground(screen, x, y, config.SlotSize, isSelected, config.BorderWidth)

		// 武器スプライトを描画
		drawWeaponSprite(screen, x, y, config.SlotSize, slot, spriteSheets)

		// スロット番号を描画
		drawSlotNumber(screen, ws.face, x, y, config.SlotSize, i+1)
	}
}

// drawSlotBackground はスロットの背景と枠線を描画
func drawSlotBackground(screen *ebiten.Image, x, y, size int, selected bool, borderWidth int) {
	fx := float32(x)
	fy := float32(y)
	fsize := float32(size)
	fborder := float32(borderWidth)

	// 背景色（半透明の黒）
	bgColor := color.RGBA{0, 0, 0, 180}
	vector.FillRect(screen, fx, fy, fsize, fsize, bgColor, false)

	// 枠線色（選択中は明るい黄色、非選択時は白）
	var borderColor color.RGBA
	if selected {
		borderColor = color.RGBA{255, 255, 100, 255} // 明るい黄色
	} else {
		borderColor = color.RGBA{200, 200, 200, 255} // グレー
	}

	// 上辺
	vector.FillRect(screen, fx, fy, fsize, fborder, borderColor, false)
	// 下辺
	vector.FillRect(screen, fx, fy+fsize-fborder, fsize, fborder, borderColor, false)
	// 左辺
	vector.FillRect(screen, fx, fy, fborder, fsize, borderColor, false)
	// 右辺
	vector.FillRect(screen, fx+fsize-fborder, fy, fborder, fsize, borderColor, false)
}

// drawSlotNumber はスロット番号を左上に描画
func drawSlotNumber(screen *ebiten.Image, face text.Face, x, y, _ int, number int) {
	numberText := string(rune('0' + number))
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x+4), float64(y+4))
	op.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255})
	text.Draw(screen, numberText, face, op)
}

// drawWeaponSprite は武器スプライトを中央に描画
func drawWeaponSprite(screen *ebiten.Image, x, y, slotSize int, slot WeaponSlotInfo, spriteSheets *map[string]gc.SpriteSheet) {
	// 武器が装備されていない場合は何も描画しない
	if slot.WeaponName == "" {
		return
	}

	if spriteSheets == nil {
		return
	}

	// スプライトシートを取得
	sheet, ok := (*spriteSheets)[slot.SpriteSheet]
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
