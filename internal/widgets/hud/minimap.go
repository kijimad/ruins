package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// Minimap はHUDのミニマップエリア
type Minimap struct {
	face    text.Face
	enabled bool
}

// NewMinimap は新しいHUDMinimapを作成する
func NewMinimap(face text.Face) *Minimap {
	return &Minimap{
		face:    face,
		enabled: true,
	}
}

// Update はミニマップを更新する
func (minimap *Minimap) Update(_ w.World) {
	// 現在は更新処理なし
}

// Draw はミニマップを描画する
func (minimap *Minimap) Draw(screen *ebiten.Image, data MinimapData) {
	if !minimap.enabled {
		return
	}

	// 探索済みタイルがない場合は空のミニマップを描画
	if len(data.ExploredTiles) == 0 {
		minimap.drawEmpty(screen, data)
		return
	}

	// ミニマップの設定
	minimapWidth := data.MinimapConfig.Width
	minimapHeight := data.MinimapConfig.Height
	minimapScale := data.MinimapConfig.Scale
	screenWidth := data.ScreenDimensions.Width
	minimapX := screenWidth - minimapWidth - theme.Space4
	minimapY := theme.Space4

	// ミニマップの背景を描画
	if minimapWidth > 0 && minimapHeight > 0 {
		minimapBg := ebiten.NewImage(minimapWidth, minimapHeight)
		minimapBg.Fill(theme.HUDMinimapBg)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(minimapX), float64(minimapY))
		screen.DrawImage(minimapBg, op)
	}

	// ミニマップの中心をプレイヤー位置に合わせる
	centerX := minimapX + minimapWidth/2
	centerY := minimapY + minimapHeight/2

	// 探索済みタイルを描画
	for gridElement := range data.ExploredTiles {
		tileX := int(gridElement.X)
		tileY := int(gridElement.Y)

		// プレイヤー位置からの相対位置を計算
		relativeX := tileX - int(data.PlayerTile.X)
		relativeY := tileY - int(data.PlayerTile.Y)

		// ミニマップ上の座標を計算（回転なし、素直な座標変換）
		// X軸: 右方向が正、Y軸: 下方向が正
		mapX := float32(centerX + relativeX*minimapScale)
		mapY := float32(centerY + relativeY*minimapScale)

		// ミニマップの範囲内かチェック
		if mapX >= float32(minimapX) && mapX <= float32(minimapX+minimapWidth-minimapScale) &&
			mapY >= float32(minimapY) && mapY <= float32(minimapY+minimapHeight-minimapScale) {

			// タイル色情報を取得
			if colorInfo, exists := data.TileColors[gridElement]; exists {
				tileColor := color.RGBA{colorInfo.R, colorInfo.G, colorInfo.B, colorInfo.A}
				vector.FillRect(screen, mapX, mapY, float32(minimapScale), float32(minimapScale), tileColor, false)
			}
		}
	}

	// 隊員の位置を青い点で表示
	squadColor := color.RGBA{80, 140, 255, 255}
	for _, pos := range data.SquadPositions {
		relX := int(pos.Tile.X - data.PlayerTile.X)
		relY := int(pos.Tile.Y - data.PlayerTile.Y)
		mx := float32(centerX + relX*minimapScale)
		my := float32(centerY + relY*minimapScale)
		if mx >= float32(minimapX) && mx <= float32(minimapX+minimapWidth) &&
			my >= float32(minimapY) && my <= float32(minimapY+minimapHeight) {
			vector.FillCircle(screen, mx, my, 2, squadColor, false)
		}
	}

	// プレイヤーの位置を赤い点で表示
	playerMapX := float32(centerX)
	playerMapY := float32(centerY)
	vector.FillCircle(screen, playerMapX, playerMapY, 2, theme.HUDPlayerMarker, false)
}

// drawEmpty は空のミニマップを描画する
func (minimap *Minimap) drawEmpty(screen *ebiten.Image, data MinimapData) {
	minimapWidth := data.MinimapConfig.Width
	minimapHeight := data.MinimapConfig.Height
	screenWidth := data.ScreenDimensions.Width
	minimapX := screenWidth - minimapWidth - theme.Space4
	minimapY := theme.Space4

	// ミニマップの背景を描画（半透明の黒い四角）
	if minimapWidth > 0 && minimapHeight > 0 {
		minimapBg := ebiten.NewImage(minimapWidth, minimapHeight)
		minimapBg.Fill(theme.HUDMinimapBg)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(minimapX), float64(minimapY))
		screen.DrawImage(minimapBg, op)
	}

	// 中央に"No Data"テキストを表示（枠線付き）
	textX := float64(minimapX + 50)
	textY := float64(minimapY + 70)
	noDataText := "No Data"

	drawOutlinedText(screen, noDataText, minimap.face, textX, textY, theme.TextPrimary)
}
