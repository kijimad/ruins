package resources

import (
	"testing"

	"github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpriteRect_エラー経路 は spriteRect の各エラー分岐を固定する。
// いずれも画像を要さず、nil・未登録・テクスチャ欠落を純粋なマップ操作で検証する。
func TestSpriteRect_エラー経路(t *testing.T) {
	t.Parallel()

	sheetNoTexture := components.SpriteSheet{
		Name:    "s",
		Sprites: map[string]components.Sprite{"k": {X: 0, Y: 0, Width: 8, Height: 8}},
		// Texture.Image は nil のまま
	}

	tests := []struct {
		name    string
		sheets  map[string]components.SpriteSheet
		sr      *components.SpriteRender
		wantErr string
	}{
		{
			name:    "srがnil",
			sheets:  map[string]components.SpriteSheet{},
			sr:      nil,
			wantErr: "sprite render is nil",
		},
		{
			name:    "シートが無い",
			sheets:  map[string]components.SpriteSheet{},
			sr:      &components.SpriteRender{SpriteSheetName: "x"},
			wantErr: "sprite sheet",
		},
		{
			name:    "キーがシートに無い",
			sheets:  map[string]components.SpriteSheet{"s": {Name: "s", Sprites: map[string]components.Sprite{}}},
			sr:      &components.SpriteRender{SpriteSheetName: "s", SpriteKey: "k"},
			wantErr: "not found in sheet",
		},
		{
			name:    "テクスチャ画像が無い",
			sheets:  map[string]components.SpriteSheet{"s": sheetNoTexture},
			sr:      &components.SpriteRender{SpriteSheetName: "s", SpriteKey: "k"},
			wantErr: "no texture image",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := spriteRect(tt.sheets, tt.sr)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestSetSpriteSheets_シートを保持しストアへ渡す は、設定したシートが Resources に
// 保持され、解決キャッシュ側の SpriteStore にも渡ることを固定する。
func TestSetSpriteSheets_シートを保持しストアへ渡す(t *testing.T) {
	t.Parallel()

	r := &Resources{Sprites: NewSpriteStore()}
	sheets := map[string]components.SpriteSheet{"s": {Name: "s"}}

	r.SetSpriteSheets(sheets)

	assert.Equal(t, sheets, r.SpriteSheets)
}
