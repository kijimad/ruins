package loader

import (
	"fmt"

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

// LoadFonts はフォントリソースを読み込む
func LoadFonts() (map[string]resources.Font, error) {
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
		return nil, fmt.Errorf("フォントファイルの読み込みに失敗: %w", err)
	}

	metaData, err := toml.Decode(string(bs), &metadata)
	if err != nil {
		return nil, fmt.Errorf("フォントメタデータのデコードに失敗: %w", err)
	}

	undecoded := metaData.Undecoded()
	if len(undecoded) > 0 {
		return nil, fmt.Errorf("unknown keys found in fonts TOML: %v", undecoded)
	}

	fonts := make(map[string]resources.Font, len(metadata.Fonts))
	for name, entry := range metadata.Fonts {
		font, err := resources.NewFont(entry.Font)
		if err != nil {
			return nil, fmt.Errorf("フォント %q の読み込みに失敗: %w", name, err)
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
			return nil, fmt.Errorf("スプライトシート '%s' の読み込みに失敗: %w", spriteSheetDef.Name, err)
		}
		sheet.Name = spriteSheetDef.Name
		spriteSheets[spriteSheetDef.Name] = sheet
	}

	return spriteSheets, nil
}

// LoadUIResources はフォントマップからUIリソースを初期化する
func LoadUIResources(fonts map[string]resources.Font) (resources.UIResources, error) {
	fontSources := []*text.GoTextFaceSource{
		fonts["dougenzaka"].FaceSource,
		fonts["nerd"].FaceSource,
	}

	return resources.NewUIResources(fontSources)
}

// BuildFaces はフォントマップからFaceマップを構築する
func BuildFaces(fonts map[string]resources.Font) map[string]text.Face {
	return map[string]text.Face{
		"dougenzaka": &text.GoTextFace{
			Source: fonts["dougenzaka"].FaceSource,
			Size:   16,
		},
	}
}

// LoadRaws はRawデータを読み込む
func LoadRaws() (oapi.Raws, error) {
	return raw.LoadFromFile(rawsPath)
}
