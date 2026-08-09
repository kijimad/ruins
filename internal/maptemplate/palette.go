package maptemplate

import (
	"fmt"
	"io"
	"maps"
	"unicode/utf8"

	"github.com/kijimaD/ruins/assets"
	"github.com/pelletier/go-toml/v2"
)

// PaletteEntry はパレットのProp/NPCマッピングの1エントリ
type PaletteEntry struct {
	ID   string `toml:"id"`
	Tile string `toml:"tile,omitempty"`
}

// Palette はマップ生成用のパレット定義
// terrainとPropsとNPCの文字マッピングを提供する
type Palette struct {
	ID          string                  `toml:"id"`
	Description string                  `toml:"description"`
	Terrain     map[string]string       `toml:"terrain,omitempty"` // {文字: terrain名}
	Props       map[string]PaletteEntry `toml:"props,omitempty"`   // {文字: Prop定義}
	NPCs        map[string]PaletteEntry `toml:"npcs,omitempty"`    // {文字: NPC定義}
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

// Load はio.Readerからパレット定義を読み込む
func (l *PaletteLoader) Load(r io.Reader) (*Palette, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read palette: %w", err)
	}

	var file PaletteFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse palette TOML: %w", err)
	}

	if err := l.validate(&file.Palette); err != nil {
		return nil, fmt.Errorf("failed to validate palette: %w", err)
	}

	return &file.Palette, nil
}

// LoadFile はTOMLファイルからパレット定義を読み込む
func (l *PaletteLoader) LoadFile(path string) (*Palette, error) {
	f, err := assets.FS.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read palette file: %w", err)
	}
	defer func() { _ = f.Close() }()

	return l.Load(f)
}

// validate はパレット定義の妥当性を検証する
func (l *PaletteLoader) validate(p *Palette) error {
	if p.ID == "" {
		return fmt.Errorf("palette ID is empty")
	}

	if len(p.Terrain) == 0 && len(p.Props) == 0 && len(p.NPCs) == 0 {
		return fmt.Errorf("terrain, Props, or NPCs definition is required")
	}

	for char := range p.Terrain {
		if utf8.RuneCountInString(char) != 1 {
			return fmt.Errorf("terrain key must be a single character: %q", char)
		}
	}

	for char, entry := range p.Props {
		if utf8.RuneCountInString(char) != 1 {
			return fmt.Errorf("props key must be a single character: %q", char)
		}
		if entry.Tile == "" {
			return fmt.Errorf("props tile is required: %q", char)
		}
	}

	for char, entry := range p.NPCs {
		if utf8.RuneCountInString(char) != 1 {
			return fmt.Errorf("npcs key must be a single character: %q", char)
		}
		if entry.Tile == "" {
			return fmt.Errorf("npcs tile is required: %q", char)
		}
	}

	// 文字の重複チェック
	used := make(map[string]string)
	for char := range p.Terrain {
		used[char] = "terrain"
	}
	for char := range p.Props {
		if category, ok := used[char]; ok {
			return fmt.Errorf("char %q is duplicated between %s and props", char, category)
		}
		used[char] = "props"
	}
	for char := range p.NPCs {
		if category, ok := used[char]; ok {
			return fmt.Errorf("char %q is duplicated between %s and npcs", char, category)
		}
	}

	return nil
}

// MergePalettes は複数のパレットをマージする
// 後のパレットが前のパレットを上書きする
func MergePalettes(palettes ...*Palette) *Palette {
	merged := &Palette{
		ID:          "merged",
		Description: "merged palette",
		Terrain:     make(map[string]string),
		Props:       make(map[string]PaletteEntry),
		NPCs:        make(map[string]PaletteEntry),
	}

	for _, p := range palettes {
		maps.Copy(merged.Terrain, p.Terrain)
		maps.Copy(merged.Props, p.Props)
		maps.Copy(merged.NPCs, p.NPCs)
	}

	return merged
}

// GetTerrain は文字からterrain名を取得する。
// Terrain定義を優先し、なければProps/NPCsのTileフィールドを参照する
func (p *Palette) GetTerrain(char string) (string, bool) {
	if terrain, ok := p.Terrain[char]; ok {
		return terrain, true
	}
	if entry, ok := p.Props[char]; ok && entry.Tile != "" {
		return entry.Tile, true
	}
	if entry, ok := p.NPCs[char]; ok && entry.Tile != "" {
		return entry.Tile, true
	}
	return "", false
}

// GetProp は文字からProp名を取得する
func (p *Palette) GetProp(char string) (string, bool) {
	entry, ok := p.Props[char]
	if !ok {
		return "", false
	}
	return entry.ID, true
}

// GetNPC は文字からNPC種別を取得する
func (p *Palette) GetNPC(char string) (string, bool) {
	entry, ok := p.NPCs[char]
	if !ok {
		return "", false
	}
	return entry.ID, true
}
