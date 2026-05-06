package maptemplate

import (
	"fmt"
	"io"
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
// 地形とPropsとNPCの文字マッピングを提供する
type Palette struct {
	ID          string                  `toml:"id"`
	Description string                  `toml:"description"`
	Terrain     map[string]string       `toml:"terrain,omitempty"` // {文字: 地形名}
	Props       map[string]PaletteEntry `toml:"props,omitempty"`   // {文字: Prop定義}
	NPCs        map[string]PaletteEntry `toml:"npcs,omitempty"`    // {文字: NPC定義}
}

// PaletteFile はTOMLファイルのルート構造
type PaletteFile struct {
	Palette Palette `toml:"palette"`
}

// MarshalPaletteFile はPaletteFileをTOMLバイト列にエンコードする
func MarshalPaletteFile(file PaletteFile) ([]byte, error) {
	return toml.Marshal(file)
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
		return nil, fmt.Errorf("パレット読み込みエラー: %w", err)
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

// LoadFile はTOMLファイルからパレット定義を読み込む
func (l *PaletteLoader) LoadFile(path string) (*Palette, error) {
	f, err := assets.FS.Open(path)
	if err != nil {
		return nil, fmt.Errorf("パレットファイル読み込みエラー: %w", err)
	}
	defer func() { _ = f.Close() }()

	return l.Load(f)
}

// validate はパレット定義の妥当性を検証する
func (l *PaletteLoader) validate(p *Palette) error {
	if p.ID == "" {
		return fmt.Errorf("パレットIDが空です")
	}

	if len(p.Terrain) == 0 && len(p.Props) == 0 && len(p.NPCs) == 0 {
		return fmt.Errorf("地形、Props、またはNPCsの定義が必要です")
	}

	for char := range p.Terrain {
		if utf8.RuneCountInString(char) != 1 {
			return fmt.Errorf("地形のキーは1文字である必要があります: %q", char)
		}
	}

	for char, entry := range p.Props {
		if utf8.RuneCountInString(char) != 1 {
			return fmt.Errorf("propsのキーは1文字である必要があります: %q", char)
		}
		if entry.Tile == "" {
			return fmt.Errorf("propsのtileは必須です: %q", char)
		}
	}

	for char, entry := range p.NPCs {
		if utf8.RuneCountInString(char) != 1 {
			return fmt.Errorf("npcsのキーは1文字である必要があります: %q", char)
		}
		if entry.Tile == "" {
			return fmt.Errorf("npcsのtileは必須です: %q", char)
		}
	}

	// 文字の重複チェック
	used := make(map[string]string)
	for char := range p.Terrain {
		used[char] = "地形"
	}
	for char := range p.Props {
		if category, ok := used[char]; ok {
			return fmt.Errorf("文字 %q が%sとpropsで重複しています", char, category)
		}
		used[char] = "props"
	}
	for char := range p.NPCs {
		if category, ok := used[char]; ok {
			return fmt.Errorf("文字 %q が%sとnpcsで重複しています", char, category)
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
		Props:       make(map[string]PaletteEntry),
		NPCs:        make(map[string]PaletteEntry),
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

// GetTerrain は文字から地形名を取得する。
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
