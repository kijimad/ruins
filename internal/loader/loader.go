package loader

import (
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/assets"
	"github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
)

const (
	fontsPath = "metadata/fonts/fonts.toml"
	rawsPath  = "metadata/entities/raw/raw.toml"
)

// fontMu はフォントと UI リソースの構築を直列化する。font source の構築は共有状態を持ち
// 並行安全でない。呼び出し側がどこから並行に読み込んでも安全なよう、守りはこの package が持つ。
// 構築後のフェイスは呼び出しごとに独立するので、利用側はロック無しで並列に描ける
var fontMu sync.Mutex

// LoadFonts はフォントリソースを読み込む
func LoadFonts() (map[string]resources.Font, error) {
	fontMu.Lock()
	defer fontMu.Unlock()
	// TOML にはフォントパスのみが入る。ロード済みの resources.Font はパスから構築する
	type fontEntry struct {
		Font string `toml:"font"`
	}
	type fontMetadata struct {
		Fonts map[string]fontEntry `toml:"font"`
	}

	var metadata fontMetadata
	bs, err := assets.FS.ReadFile(fontsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load font file: %w", err)
	}

	metaData, err := toml.Decode(string(bs), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode font metadata: %w", err)
	}

	undecoded := metaData.Undecoded()
	if len(undecoded) > 0 {
		return nil, fmt.Errorf("unknown keys found in fonts TOML: %v", undecoded)
	}

	fonts := make(map[string]resources.Font, len(metadata.Fonts))
	for name, entry := range metadata.Fonts {
		font, err := resources.NewFont(entry.Font)
		if err != nil {
			return nil, fmt.Errorf("failed to load font %q: %w", name, err)
		}
		fonts[name] = font
	}

	return fonts, nil
}

// LoadSpriteSheets はoapi.RawsのSpriteSheet定義に基づいてスプライトシートを読み込む
func LoadSpriteSheets(raws oapi.Raws) (map[string]components.SpriteSheet, error) {
	spriteSheets := make(map[string]components.SpriteSheet)

	for _, spriteSheetDef := range raw.PtrSlice(raws.SpriteSheets) {
		sheet, err := LoadSpriteSheetFromAseprite(spriteSheetDef.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to load sprite sheet '%s': %w", spriteSheetDef.Name, err)
		}
		sheet.Name = spriteSheetDef.Name
		spriteSheets[spriteSheetDef.Name] = sheet
	}

	return spriteSheets, nil
}

// LoadUIResources はフォントマップからUIリソースを初期化する
func LoadUIResources(fonts map[string]resources.Font) (resources.UIResources, error) {
	fontMu.Lock()
	defer fontMu.Unlock()
	fontSources := []*text.GoTextFaceSource{
		fonts["dougenzaka"].FaceSource,
		fonts["nerd"].FaceSource,
	}

	return resources.NewUIResources(fontSources)
}

// LoadRaws はRawデータを読み込む
func LoadRaws() (oapi.Raws, error) {
	return raw.LoadFromFile(rawsPath)
}
