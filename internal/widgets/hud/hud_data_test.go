package hud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHUDData はHUDDataの統合データ構造をテスト
func TestHUDData(t *testing.T) {
	t.Parallel()
	hudData := Data{
		GameInfo: GameInfoData{
			FloorNumber: 3,
		},
		MacroMap: MacroMapData{
			HasBand: true,
			Config:  MacroMapConfig{Width: 150, Height: 150, MinGlyphPx: 10},
			Screen:  ScreenDimensions{Width: 1024, Height: 768},
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
	assert.True(t, hudData.MacroMap.HasBand)
	assert.Len(t, hudData.MessageData.Messages, 2)
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
