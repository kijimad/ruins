package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
)

// SquadHUD は隊員HP一覧を表示するHUDウィジェット
type SquadHUD struct {
	face text.Face
}

// NewSquadHUD は新しいSquadHUDを作成する
func NewSquadHUD(face text.Face) *SquadHUD {
	return &SquadHUD{face: face}
}

// Draw は隊員HP一覧を描画する。ミニマップの下に配置する
func (s *SquadHUD) Draw(screen *ebiten.Image, data SquadHUDData) {
	if len(data.Members) == 0 {
		return
	}

	lineHeight := 14
	barWidth := 60
	barHeight := 6
	padding := theme.Space2
	nameWidth := 50
	startX := data.ScreenDimensions.Width - theme.Space4 - nameWidth - barWidth
	startY := theme.Space4 + 160 // ミニマップの下

	for i, member := range data.Members {
		y := startY + i*(lineHeight+padding)

		// 名前
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(float64(startX), float64(y))
		nameOp.ColorScale.ScaleWithColor(theme.TextPrimary)
		text.Draw(screen, member.Name, s.face, nameOp)

		// HPバー
		barX := float32(startX + nameWidth)
		barY := float32(y + 2)

		// 背景バー
		vector.FillRect(screen, barX, barY, float32(barWidth), float32(barHeight), color.RGBA{40, 40, 40, 200}, false)

		// HPバー
		hpRatio := float32(0)
		if member.MaxHP > 0 {
			hpRatio = float32(member.CurrentHP) / float32(member.MaxHP)
		}
		barColor := color.RGBA{80, 200, 80, 255}
		if hpRatio < 0.25 {
			barColor = color.RGBA{200, 50, 50, 255}
		} else if hpRatio < 0.5 {
			barColor = color.RGBA{200, 200, 50, 255}
		}
		vector.FillRect(screen, barX, barY, float32(barWidth)*hpRatio, float32(barHeight), barColor, false)
	}
}
