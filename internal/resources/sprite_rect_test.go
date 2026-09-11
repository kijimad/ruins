package resources

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
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
		wantErr error
	}{
		{
			name:    "srがnil",
			sheets:  map[string]components.SpriteSheet{},
			sr:      nil,
			wantErr: ErrSpriteRenderNil,
		},
		{
			name:    "シートが無い",
			sheets:  map[string]components.SpriteSheet{},
			sr:      &components.SpriteRender{SpriteSheetName: "x"},
			wantErr: ErrSpriteSheetNotFound,
		},
		{
			name:    "キーがシートに無い",
			sheets:  map[string]components.SpriteSheet{"s": {Name: "s", Sprites: map[string]components.Sprite{}}},
			sr:      &components.SpriteRender{SpriteSheetName: "s", SpriteKey: "k"},
			wantErr: ErrSpriteKeyNotFound,
		},
		{
			name:    "テクスチャ画像が無い",
			sheets:  map[string]components.SpriteSheet{"s": sheetNoTexture},
			sr:      &components.SpriteRender{SpriteSheetName: "s", SpriteKey: "k"},
			wantErr: ErrSpriteNoTexture,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := spriteRect(tt.sheets, tt.sr)
			// 内部関数のエラーは sentinel で同定する。文字列照合は規約で禁止
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestSetSpriteSheets_シートを保持しストアへ渡す は、設定したシートが Resources に
// 保持され、解決キャッシュ側の SpriteStore にも渡ることを固定する。
func TestSetSpriteSheets_シートを保持しストアへ渡す(t *testing.T) {
	t.Parallel()

	r := &Resources{Sprites: NewSpriteStore()}
	sheets := map[string]components.SpriteSheet{
		"s": {
			Name:    "s",
			Texture: components.Texture{Image: ebiten.NewImage(16, 16)},
			Sprites: map[string]components.Sprite{"k": {Width: 8, Height: 8}},
		},
	}

	r.SetSpriteSheets(sheets)

	// Resources 側にシートが保持される
	assert.Equal(t, sheets, r.SpriteSheets)

	// もう一つの責務。SpriteStore へも渡り、渡したシートで解決できる。
	// 委譲が壊れるとストアは空のままで ok=false になる
	_, _, ok := r.Sprites.Rect(&components.SpriteRender{SpriteSheetName: "s", SpriteKey: "k"})
	assert.True(t, ok, "SpriteStore にもシートが渡り解決できる")
}
