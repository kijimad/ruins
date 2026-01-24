package maptemplate

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Palette はマップ生成用のパレット定義
// 地形とPropsとNPCの文字マッピングを提供する
type Palette struct {
	ID          string            `toml:"id"`
	Description string            `toml:"description"`
	Terrain     map[string]string `toml:"terrain"` // {文字: 地形名}
	Props       map[string]string `toml:"props"`   // {文字: Prop名（ドア、テーブルなど）}
	NPCs        map[string]string `toml:"npcs"`    // {文字: NPC種別}
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

	if len(p.Terrain) == 0 && len(p.Props) == 0 && len(p.NPCs) == 0 {
		return fmt.Errorf("地形、Props、またはNPCsの定義が必要です")
	}

	// 文字の重複チェック
	for char := range p.Terrain {
		if len(char) != 1 {
			return fmt.Errorf("地形のキーは1文字である必要があります: %q", char)
		}
	}

	for char := range p.Props {
		if len(char) != 1 {
			return fmt.Errorf("Propsのキーは1文字である必要があります: %q", char)
		}
	}

	for char := range p.NPCs {
		if len(char) != 1 {
			return fmt.Errorf("NPCsのキーは1文字である必要があります: %q", char)
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
		Props:       make(map[string]string),
		NPCs:        make(map[string]string),
	}

	for _, p := range palettes {
		for k, v := range p.Terrain {
			merged.Terrain[k] = v
		}
		for k, v := range p.Props {
			merged.Props[k] = v
		}
		for k, v := range p.NPCs {
			merged.NPCs[k] = v
		}
	}

	return merged
}

// GetTerrain は文字から地形名を取得する
func (p *Palette) GetTerrain(char string) (string, bool) {
	terrain, ok := p.Terrain[char]
	return terrain, ok
}

// GetProp は文字からProp名を取得する
func (p *Palette) GetProp(char string) (string, bool) {
	prop, ok := p.Props[char]
	return prop, ok
}

// GetNPC は文字からNPC種別を取得する
func (p *Palette) GetNPC(char string) (string, bool) {
	npc, ok := p.NPCs[char]
	return npc, ok
}
