package hud

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	w "github.com/kijimaD/ruins/internal/world"
)

// DebugOverlay はAI情報のデバッグ表示エリア
type DebugOverlay struct {
	face    text.Face
	enabled bool
}

// NewDebugOverlay は新しいHUDDebugOverlayを作成する
func NewDebugOverlay(face text.Face) *DebugOverlay {
	return &DebugOverlay{
		face:    face,
		enabled: true,
	}
}

// Update はデバッグオーバーレイを更新する
func (overlay *DebugOverlay) Update(_ w.World) {
	// 現在は更新処理なし
}

// Draw はデバッグオーバーレイを描画する
func (overlay *DebugOverlay) Draw(screen *ebiten.Image, data DebugOverlayData) {
	if !overlay.enabled || !data.Enabled {
		return
	}

	// AI状態を描画
	for _, aiState := range data.AIStates {
		textOffsetY := 30.0
		drawOutlinedText(screen, aiState.StateText, overlay.face, float64(int(aiState.ScreenX)-20), aiState.ScreenY-textOffsetY, color.White)
	}

	// 視界範囲を描画
	for _, visionRange := range data.VisionRanges {
		overlay.drawVisionCircle(screen, float32(visionRange.ScreenX), float32(visionRange.ScreenY), visionRange.ScaledRadius)
	}

	// HP情報を描画
	for _, hpDisplay := range data.HPDisplays {
		hpText := fmt.Sprintf("%d/%d", hpDisplay.CurrentHP, hpDisplay.MaxHP)
		textOffsetY := 15.0 // AI状態テキスト（30.0）より上に表示して重複を避ける
		drawOutlinedText(screen, hpText, overlay.face, float64(int(hpDisplay.ScreenX)-15), hpDisplay.ScreenY-textOffsetY, color.White)
	}
}

// drawVisionCircle は指定した位置と半径で視界円を描画する
func (overlay *DebugOverlay) drawVisionCircle(screen *ebiten.Image, centerX, centerY, radius float32) {
	// 円周上の点数
	circlePoints := 32
	vertices := []ebiten.Vertex{}
	indices := []uint16{}

	// 中心点
	vertices = append(vertices, ebiten.Vertex{
		DstX:   centerX,
		DstY:   centerY,
		SrcX:   0,
		SrcY:   0,
		ColorR: 0.0,
		ColorG: 1.0,
		ColorB: 0.0,
		ColorA: 0.3, // 半透明
	})

	// 円周上の点
	for i := 0; i < circlePoints; i++ {
		angle := 2 * math.Pi * float64(i) / float64(circlePoints)
		x := centerX + radius*float32(math.Cos(angle))
		y := centerY + radius*float32(math.Sin(angle))

		vertices = append(vertices, ebiten.Vertex{
			DstX:   x,
			DstY:   y,
			SrcX:   0,
			SrcY:   0,
			ColorR: 0.0,
			ColorG: 1.0,
			ColorB: 0.0,
			ColorA: 0.3,
		})

		// 三角形のインデックス
		if i < circlePoints {
			indices = append(indices, 0, uint16(i+1), uint16((i+1)%circlePoints+1))
		}
	}

	// 円を描画
	opt := &ebiten.DrawTrianglesOptions{}
	// 1x1ピクセルの白い画像を作成
	whiteImg := ebiten.NewImage(1, 1)
	whiteImg.Fill(color.White)
	screen.DrawTriangles(vertices, indices, whiteImg, opt)
}
