package loader

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kijimaD/ruins/assets"
	"github.com/kijimaD/ruins/internal/components"
)

// LoadSpriteSheetFromAseprite は Aseprite JSON フォーマットからスプライトシートを読み込む
func LoadSpriteSheetFromAseprite(jsonPath string) (components.SpriteSheet, error) {
	// JSONファイルを読み込み
	bs, err := assets.FS.ReadFile(jsonPath)
	if err != nil {
		return components.SpriteSheet{}, fmt.Errorf("JSONファイルの読み込みに失敗: %w", err)
	}

	var aseData AsepriteJSON
	if err := json.Unmarshal(bs, &aseData); err != nil {
		return components.SpriteSheet{}, fmt.Errorf("JSONのパースに失敗: %w", err)
	}

	// 画像ファイルを読み込み
	imagePath := filepath.Join(filepath.Dir(jsonPath), aseData.Meta.Image)
	var texture components.Texture
	if err := texture.UnmarshalText([]byte(imagePath)); err != nil {
		return components.SpriteSheet{}, fmt.Errorf("画像の読み込みに失敗: %w", err)
	}

	// スプライト辞書を構築
	sprites := make(map[string]components.Sprite)

	for _, frame := range aseData.Frames {
		sprite := components.Sprite{
			X:      frame.Frame.X,
			Y:      frame.Frame.Y,
			Width:  frame.Frame.W,
			Height: frame.Frame.H,
		}

		if !strings.HasSuffix(frame.Filename, "_") {
			return components.SpriteSheet{}, fmt.Errorf("スプライトファイル名は'_'で終わる必要があります: %s", frame.Filename)
		}
		// キー名の生成（末尾のアンダースコアを削除）
		key := strings.TrimSuffix(frame.Filename, "_")

		// 重複チェック
		if _, exists := sprites[key]; exists {
			return components.SpriteSheet{}, fmt.Errorf("重複したスプライトキー: %s", key)
		}

		sprites[key] = sprite
	}

	return components.SpriteSheet{
		Name:    filepath.Base(jsonPath),
		Texture: texture,
		Sprites: sprites,
	}, nil
}
