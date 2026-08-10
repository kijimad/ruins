package hud

import (
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

	y := startY
	for _, member := range data.Members {
		// 名前
		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(float64(startX), float64(y))
		nameOp.ColorScale.ScaleWithColor(theme.TextPrimary)
		text.Draw(screen, member.Name, s.face, nameOp)

		// HPバー
		barX := float32(startX + nameWidth)
		barY := float32(y + 2)

		// 背景バー
		vector.FillRect(screen, barX, barY, float32(barWidth), float32(barHeight), theme.HUDSquadBarBg, false)

		// HPバー
		hpRatio := float32(0)
		if member.MaxHP > 0 {
			hpRatio = float32(member.CurrentHP) / float32(member.MaxHP)
		}
		barColor := theme.HUDSquadHPHigh
		if hpRatio < 0.25 {
			barColor = theme.HUDSquadHPLow
		} else if hpRatio < 0.5 {
			barColor = theme.HUDSquadHPMid
		}
		vector.FillRect(screen, barX, barY, float32(barWidth)*hpRatio, float32(barHeight), barColor, false)

		// 空腹表示。空腹以上のときだけHPバーの下に出す。出した分だけ行を高くして次の隊員と重ならないようにする
		rowHeight := lineHeight + padding
		if member.HungerLevel != "" {
			hungerOp := &text.DrawOptions{}
			hungerOp.GeoM.Translate(float64(startX+nameWidth), float64(y+barHeight+2))
			hungerOp.ColorScale.ScaleWithColor(theme.HUDSquadHunger)
			text.Draw(screen, member.HungerLevel, s.face, hungerOp)
			rowHeight = barHeight + 2 + lineHeight + padding
		}
		y += rowHeight
	}
}
