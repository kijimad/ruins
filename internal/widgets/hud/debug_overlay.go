package hud

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/ui"
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

// Draw はデバッグオーバーレイを描画する。
//
// これは意匠の chrome ではなく、世界に重ねる図形。敵の頭上の文字と視界の円で、
// どちらも世界の座標に従う。円は多角形の帯で塗るので Canvas の語彙では表せず、
// 描画先を直に受け取る。文字は Canvas 越しに描き、縁取りを HUD の他と揃える
func (overlay *DebugOverlay) Draw(screen *ebiten.Image, data DebugOverlayData) {
	if !overlay.enabled || !data.Enabled {
		return
	}
	cv := ui.NewEbitenCanvas(screen)

	// AI状態を描画
	for _, aiState := range data.AIStates {
		const textOffsetY = 30
		drawOutlinedText(cv, aiState.StateText, overlay.face, image.Pt(int(aiState.Screen.X)-20, int(aiState.Screen.Y)-textOffsetY), theme.TextPrimary)
	}

	// 視界範囲を描画
	for _, visionRange := range data.VisionRanges {
		overlay.drawVisionCircle(screen, float32(visionRange.Screen.X), float32(visionRange.Screen.Y), visionRange.ScaledRadius)
	}

	// HP情報を描画
	for _, hpDisplay := range data.HPDisplays {
		hpText := fmt.Sprintf("%d/%d", hpDisplay.CurrentHP, hpDisplay.MaxHP)
		const textOffsetY = 15 // AI状態の文字より上に置いて重ならないようにする
		drawOutlinedText(cv, hpText, overlay.face, image.Pt(int(hpDisplay.Screen.X)-15, int(hpDisplay.Screen.Y)-textOffsetY), theme.TextPrimary)
	}
}

// drawVisionCircle は指定した位置と半径で視界円を描画する
func (overlay *DebugOverlay) drawVisionCircle(screen *ebiten.Image, centerX, centerY, radius float32) {
	// 円周上の点数
	circlePoints := 32
	vertices := make([]ebiten.Vertex, 0, 1+circlePoints)
	indices := make([]uint16, 0, circlePoints*3)

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
	for i := range circlePoints {
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
