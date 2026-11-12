package maptemplate

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"math/rand/v2"
	"slices"
	"unicode/utf8"

	"github.com/kijimaD/ruins/internal/consts"

	"github.com/kijimaD/ruins/assets"
	"github.com/pelletier/go-toml/v2"
)

var (
	// errPaletteIDEmpty はパレットIDが空のときに返す
	errPaletteIDEmpty = errors.New("palette ID is empty")
	// errPaletteNoDefinition は文字の定義がどれもないときに返す
	errPaletteNoDefinition = errors.New("palette has no char definition")
	// errPaletteKeyNotSingleChar はパレットのキーが1文字でないときに返す
	errPaletteKeyNotSingleChar = errors.New("key must be a single character")
	// errPaletteTileRequired はPropsやNPCsのtileが未指定のときに返す
	errPaletteTileRequired = errors.New("tile is required")
	// errPaletteCharDuplicated は同じ文字が複数の分類で使われているときに返す
	errPaletteCharDuplicated = errors.New("char is duplicated")
	// errPalettePropIDAmbiguous は prop の id と候補の両方か、どちらも無いときに返す
	errPalettePropIDAmbiguous = errors.New("prop needs exactly one of id or candidates")
	// errPaletteItemGroupRequired は床の戦利品の itemGroup が未指定のときに返す
	errPaletteItemGroupRequired = errors.New("itemGroup is required")
)

// PaletteEntry はパレットのProp/NPCマッピングの1エントリ
type PaletteEntry struct {
	ID   string `toml:"id"`
	Tile string `toml:"tile,omitempty"`
	// Candidates は prop を重み付きで選ぶ候補。ID とどちらか一方だけを書く
	Candidates []WeightedID `toml:"candidates,omitempty"`
	// LootTable は収納の中身を引く itemTables の id。収納でなければ空
	LootTable string `toml:"lootTable,omitempty"`
}

// WeightedID は重み付きの候補の1つ
type WeightedID struct {
	ID     string `toml:"id"`
	Weight int    `toml:"weight"`
}

// ItemEntry は床に落ちる戦利品を置く文字の定義
type ItemEntry struct {
	Tile      string `toml:"tile"`
	ItemGroup string `toml:"itemGroup"`
}

// TileEntry は下に敷く地形だけを持つ文字の定義
type TileEntry struct {
	Tile string `toml:"tile"`
}

// Palette はマップ生成用のパレット定義
// terrainとPropsとNPCの文字マッピングを提供する
type Palette struct {
	ID          string                  `toml:"id"`
	Description string                  `toml:"description"`
	Terrain     map[string]string       `toml:"terrain,omitempty"` // {文字: terrain名}
	Props       map[string]PaletteEntry `toml:"props,omitempty"`   // {文字: Prop定義}
	NPCs        map[string]PaletteEntry `toml:"npcs,omitempty"`    // {文字: NPC定義}
	Items       map[string]ItemEntry    `toml:"items,omitempty"`   // {文字: 床の戦利品}
	Beings      map[string]TileEntry    `toml:"beings,omitempty"`  // {文字: 施設の敵の表から引く敵}
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
		return errPaletteIDEmpty
	}

	if len(p.Terrain)+len(p.Props)+len(p.NPCs)+len(p.Items)+len(p.Beings) == 0 {
		return errPaletteNoDefinition
	}

	for char := range p.Terrain {
		if utf8.RuneCountInString(char) != 1 {
			return fmt.Errorf("terrain %w: %q", errPaletteKeyNotSingleChar, char)
		}
	}
	for char, entry := range p.Props {
		if err := validateTileChar("props", char, entry.Tile); err != nil {
			return err
		}
		if (entry.ID == "") == (len(entry.Candidates) == 0) {
			return fmt.Errorf("props %w: %q", errPalettePropIDAmbiguous, char)
		}
		for _, c := range entry.Candidates {
			if c.Weight <= 0 {
				return fmt.Errorf("props %q candidate %q: %w: %d", char, c.ID, errNonPositiveWeight, c.Weight)
			}
		}
	}
	for char, entry := range p.NPCs {
		if err := validateTileChar("npcs", char, entry.Tile); err != nil {
			return err
		}
	}
	for char, entry := range p.Items {
		if err := validateTileChar("items", char, entry.Tile); err != nil {
			return err
		}
		if entry.ItemGroup == "" {
			return fmt.Errorf("items %w: %q", errPaletteItemGroupRequired, char)
		}
	}
	for char, entry := range p.Beings {
		if err := validateTileChar("beings", char, entry.Tile); err != nil {
			return err
		}
	}

	used := make(map[string]string)
	for _, c := range []struct {
		category string
		chars    []string
	}{
		{"terrain", keysOf(p.Terrain)},
		{"props", keysOf(p.Props)},
		{"npcs", keysOf(p.NPCs)},
		{"items", keysOf(p.Items)},
		{"beings", keysOf(p.Beings)},
	} {
		for _, char := range c.chars {
			if other, ok := used[char]; ok {
				return fmt.Errorf("%w: %q between %s and %s", errPaletteCharDuplicated, char, other, c.category)
			}
			used[char] = c.category
		}
	}

	return nil
}

// validateTileChar は文字が1文字で、下に敷く地形を持つことを検証する
func validateTileChar(category, char, tile string) error {
	if utf8.RuneCountInString(char) != 1 {
		return fmt.Errorf("%s %w: %q", category, errPaletteKeyNotSingleChar, char)
	}
	if tile == "" {
		return fmt.Errorf("%s %w: %q", category, errPaletteTileRequired, char)
	}
	return nil
}

// keysOf は map のキーを返す
func keysOf[V any](m map[string]V) []string {
	return slices.Collect(maps.Keys(m))
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
		Items:       make(map[string]ItemEntry),
		Beings:      make(map[string]TileEntry),
	}

	for _, p := range palettes {
		maps.Copy(merged.Terrain, p.Terrain)
		maps.Copy(merged.Props, p.Props)
		maps.Copy(merged.NPCs, p.NPCs)
		maps.Copy(merged.Items, p.Items)
		maps.Copy(merged.Beings, p.Beings)
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
	if entry, ok := p.Items[char]; ok {
		return entry.Tile, true
	}
	if entry, ok := p.Beings[char]; ok {
		return entry.Tile, true
	}
	return "", false
}

// GetProp は文字から prop の id と収納の中身の表を引く。候補を持つ文字は rng で1つ選ぶ
func (p *Palette) GetProp(char string, rng *rand.Rand) (id, lootTable string, ok bool) {
	entry, ok := p.Props[char]
	if !ok {
		return "", "", false
	}
	if len(entry.Candidates) == 0 {
		return entry.ID, entry.LootTable, true
	}
	i, _ := consts.Pick(rng, entry.Candidates, func(c WeightedID) float64 { return float64(c.Weight) })
	return entry.Candidates[i].ID, entry.LootTable, true
}

// GetNPC は文字からNPC種別を取得する
func (p *Palette) GetNPC(char string) (string, bool) {
	entry, ok := p.NPCs[char]
	if !ok {
		return "", false
	}
	return entry.ID, true
}
