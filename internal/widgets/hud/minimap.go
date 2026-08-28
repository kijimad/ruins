package hud

import (
	"image"

	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// playerMarkerR はプレイヤーの印の半径。地図の粒に対して小さすぎず大きすぎない大きさ
const playerMarkerR = 2

// Minimap はHUDのミニマップエリア
type Minimap struct {
	face    text.Face
	chrome  Chrome
	enabled bool
}

// NewMinimap は新しいHUDMinimapを作成する
func NewMinimap(face text.Face, chrome Chrome) *Minimap {
	return &Minimap{
		face:    face,
		chrome:  chrome,
		enabled: true,
	}
}

// Update はミニマップを更新する
func (minimap *Minimap) Update(_ w.World) {
	// 現在は更新処理なし
}

// Draw はミニマップを描画する
func (minimap *Minimap) Draw(cv ui.Canvas, data MinimapData) {
	if !minimap.enabled {
		return
	}

	// 探索済みタイルがない場合は空のミニマップを描画
	if len(data.ExploredTiles) == 0 {
		minimap.drawEmpty(cv, data)
		return
	}

	// ミニマップの設定
	minimapWidth := data.MinimapConfig.Width
	minimapHeight := data.MinimapConfig.Height
	minimapScale := data.MinimapConfig.Scale
	screenWidth := data.ScreenDimensions.Width
	minimapX := screenWidth - minimapWidth - theme.Space4
	minimapY := theme.Space4

	// ミニマップの背景を描画。メニュー枠と同じ共通 chrome に揃える
	if minimapWidth > 0 && minimapHeight > 0 {
		minimap.chrome.Panel(cv, image.Rect(minimapX, minimapY, minimapX+minimapWidth, minimapY+minimapHeight))
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
				// タイルの色は地形のデータから来るので、意匠のテクスチャでなく塗りで表す
				cv.FillRect(image.Rect(int(mapX), int(mapY), int(mapX)+minimapScale, int(mapY)+minimapScale), tileColor)
			}
		}
	}

	// プレイヤーの位置を印で示す。地図の粒と同じ大きさの四角にして、タイルの並びから浮かせない
	cv.FillRect(image.Rect(centerX-playerMarkerR, centerY-playerMarkerR, centerX+playerMarkerR, centerY+playerMarkerR),
		theme.HUDPlayerMarker)
}

// drawEmpty は空のミニマップを描画する
func (minimap *Minimap) drawEmpty(cv ui.Canvas, data MinimapData) {
	minimapWidth := data.MinimapConfig.Width
	minimapHeight := data.MinimapConfig.Height
	screenWidth := data.ScreenDimensions.Width
	minimapX := screenWidth - minimapWidth - theme.Space4
	minimapY := theme.Space4

	// ミニマップの背景を描画。メニュー枠と同じ共通 chrome に揃える
	if minimapWidth > 0 && minimapHeight > 0 {
		minimap.chrome.Panel(cv, image.Rect(minimapX, minimapY, minimapX+minimapWidth, minimapY+minimapHeight))
	}

	// 中央に"No Data"テキストを表示（枠線付き）
	textX := float64(minimapX + 50)
	textY := float64(minimapY + 70)
	noDataText := "No Data"

	drawOutlinedText(cv, noDataText, minimap.face, image.Pt(int(textX), int(textY)), theme.TextPrimary)
}
