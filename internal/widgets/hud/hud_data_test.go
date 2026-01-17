package hud

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

// TestHUDData はHUDDataの統合データ構造をテスト
func TestHUDData(t *testing.T) {
	t.Parallel()
	hudData := Data{
		GameInfo: GameInfoData{
			FloorNumber: 3,
			TurnNumber:  25,
			PlayerMoves: 75,
		},
		MinimapData: MinimapData{
			PlayerTileX:      5,
			PlayerTileY:      7,
			ExploredTiles:    map[gc.GridElement]bool{gc.GridElement{X: 5, Y: 7}: true},
			TileColors:       map[gc.GridElement]TileColorInfo{gc.GridElement{X: 5, Y: 7}: {R: 255, G: 255, B: 255, A: 255}},
			MinimapConfig:    MinimapConfig{Width: 150, Height: 150, Scale: 3},
			ScreenDimensions: ScreenDimensions{Width: 1024, Height: 768},
		},
		DebugOverlay: DebugOverlayData{
			Enabled: false,
		},
		MessageData: MessageData{
			Messages:         []string{"Hello", "World"},
			ScreenDimensions: ScreenDimensions{Width: 1024, Height: 768},
			Config:           DefaultMessageAreaConfig,
		},
	}

	// データ構造が正しく作成されることを確認
	assert.Equal(t, 3, hudData.GameInfo.FloorNumber)
	assert.Equal(t, 25, hudData.GameInfo.TurnNumber)
	assert.Equal(t, 75, hudData.GameInfo.PlayerMoves)
	assert.Equal(t, 5, hudData.MinimapData.PlayerTileX)
	assert.Len(t, hudData.MessageData.Messages, 2)
}

// TestTileColorInfo はタイル色情報の構造をテスト
func TestTileColorInfo(t *testing.T) {
	t.Parallel()
	colorInfo := TileColorInfo{
		R: 128,
		G: 64,
		B: 32,
		A: 255,
	}

	assert.Equal(t, uint8(128), colorInfo.R)
	assert.Equal(t, uint8(255), colorInfo.A)
}

// TestScreenDimensions は画面サイズ情報をテスト
func TestScreenDimensions(t *testing.T) {
	t.Parallel()
	dimensions := ScreenDimensions{
		Width:  1920,
		Height: 1080,
	}

	assert.Equal(t, 1920, dimensions.Width)
	assert.Equal(t, 1080, dimensions.Height)
}
