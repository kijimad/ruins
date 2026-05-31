package loader

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/kijimaD/ruins/assets"
	"github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
)

const (
	fontsPath = "metadata/fonts/fonts.toml"
	rawsPath  = "metadata/entities/raw/raw.toml"
)

// LoadFonts はフォントリソースを読み込む
func LoadFonts() (map[string]resources.Font, error) {
	type fontMetadata struct {
		Fonts map[string]resources.Font `toml:"font"`
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

	return metadata.Fonts, nil
}

// LoadSpriteSheets はraw.MasterのSpriteSheet定義に基づいてスプライトシートを読み込む
func LoadSpriteSheets(rawMaster raw.Master) (map[string]components.SpriteSheet, error) {
	spriteSheets := make(map[string]components.SpriteSheet)

	for _, spriteSheetDef := range rawMaster.Raws.SpriteSheets {
		sheet, err := LoadSpriteSheetFromAseprite(spriteSheetDef.Path)
		if err != nil {
			return nil, fmt.Errorf("スプライトシート '%s' の読み込みに失敗: %w", spriteSheetDef.Name, err)
		}
		sheet.Name = spriteSheetDef.Name
		spriteSheets[spriteSheetDef.Name] = sheet
	}

	return spriteSheets, nil
}

// LoadRaws はRawデータを読み込む
func LoadRaws() (raw.Master, error) {
	return raw.LoadFromFile(rawsPath)
}
