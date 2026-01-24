package maptemplate

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Palette はマップ生成用のパレット定義
// 地形と家具の文字マッピングを提供する
type Palette struct {
	ID          string            `toml:"id"`
	Description string            `toml:"description"`
	Terrain     map[string]string `toml:"terrain"` // {文字: 地形名}
	// TODO(kijima): 家具に限らない。Prop とかのほうがいいかも
	Furniture map[string]string `toml:"furniture"` // {文字: 家具名（ドアも含む）}
}

// PaletteFile はTOMLファイルのルート構造
type PaletteFile struct {
	Palette Palette `toml:"palette"`
}

// PaletteLoader はパレット定義の読み込みを担当する
type PaletteLoader struct{}

// NewPaletteLoader はPaletteLoaderを生成する
func NewPaletteLoader() *PaletteLoader {
	return &PaletteLoader{}
}

// LoadFromFile はTOMLファイルからパレット定義を読み込む
func (l *PaletteLoader) LoadFromFile(path string) (*Palette, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("パレットファイル読み込みエラー: %w", err)
	}

	var file PaletteFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("パレットTOMLパースエラー: %w", err)
	}

	if err := l.validate(&file.Palette); err != nil {
		return nil, fmt.Errorf("パレット検証エラー: %w", err)
	}

	return &file.Palette, nil
}

// validate はパレット定義の妥当性を検証する
func (l *PaletteLoader) validate(p *Palette) error {
	if p.ID == "" {
		return fmt.Errorf("パレットIDが空です")
	}

	if len(p.Terrain) == 0 && len(p.Furniture) == 0 {
		return fmt.Errorf("地形または家具の定義が必要です")
	}

	// 文字の重複チェック（地形と家具で同じ文字を使うのはOK）
	for char := range p.Terrain {
		if len(char) != 1 {
			return fmt.Errorf("地形のキーは1文字である必要があります: %q", char)
		}
	}

	for char := range p.Furniture {
		if len(char) != 1 {
			return fmt.Errorf("家具のキーは1文字である必要があります: %q", char)
		}
	}

	return nil
}

// MergePalettes は複数のパレットをマージする
// 後のパレットが前のパレットを上書きする
func MergePalettes(palettes ...*Palette) *Palette {
	merged := &Palette{
		ID:          "merged",
		Description: "マージされたパレット",
		Terrain:     make(map[string]string),
		Furniture:   make(map[string]string),
	}

	for _, p := range palettes {
		for k, v := range p.Terrain {
			merged.Terrain[k] = v
		}
		for k, v := range p.Furniture {
			merged.Furniture[k] = v
		}
	}

	return merged
}

// GetTerrain は文字から地形名を取得する
func (p *Palette) GetTerrain(char string) (string, bool) {
	terrain, ok := p.Terrain[char]
	return terrain, ok
}

// GetFurniture は文字から家具名を取得する
func (p *Palette) GetFurniture(char string) (string, bool) {
	furniture, ok := p.Furniture[char]
	return furniture, ok
}
