package loader

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/kijimaD/ruins/assets"
	"github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
)

// ResourceLoader はすべてのリソースの読み込みを統括するインターフェース
type ResourceLoader interface {
	// フォント関連
	LoadFonts() (map[string]resources.Font, error)
	// スプライトシート関連
	LoadSpriteSheets() (map[string]components.SpriteSheet, error)
	// Raw(エンティティ定義)関連
	LoadRaws() (*raw.Master, error)
}

// DefaultResourceLoader はResourceLoaderのデフォルト実装
type DefaultResourceLoader struct {
	// 設定ファイルのパス
	config ResourceConfig
	// キャッシュされたリソース
	cache *ResourceCache
}

// ResourceConfig はリソースファイルのパスを管理する設定
type ResourceConfig struct {
	FontsPath        string
	SpriteSheetsPath string
	RawsPath         string
}

// ResourceCache は読み込み済みのリソースをキャッシュする
type ResourceCache struct {
	Fonts        map[string]resources.Font
	SpriteSheets map[string]components.SpriteSheet
	RawMaster    *raw.Master
}

// NewResourceLoader はデフォルトのパス設定でResourceLoaderを作成する
func NewResourceLoader() ResourceLoader {
	return &DefaultResourceLoader{
		config: ResourceConfig{
			FontsPath:        "metadata/fonts/fonts.toml",
			SpriteSheetsPath: "metadata/spritesheets/spritesheets.toml",
			RawsPath:         "metadata/entities/raw/raw.toml",
		},
		cache: &ResourceCache{},
	}
}

// LoadFonts はフォントリソースを読み込む
func (rl *DefaultResourceLoader) LoadFonts() (map[string]resources.Font, error) {
	// キャッシュがあれば返す
	if rl.cache.Fonts != nil {
		return rl.cache.Fonts, nil
	}

	type fontMetadata struct {
		Fonts map[string]resources.Font `toml:"font"`
	}

	var metadata fontMetadata
	bs, err := assets.FS.ReadFile(rl.config.FontsPath)
	if err != nil {
		return nil, fmt.Errorf("フォントファイルの読み込みに失敗: %w", err)
	}

	metaData, err := toml.Decode(string(bs), &metadata)
	if err != nil {
		return nil, fmt.Errorf("フォントメタデータのデコードに失敗: %w", err)
	}

	// 未知のキーがあった場合はエラーにする
	undecoded := metaData.Undecoded()
	if len(undecoded) > 0 {
		return nil, fmt.Errorf("unknown keys found in fonts TOML: %v", undecoded)
	}

	rl.cache.Fonts = metadata.Fonts
	return metadata.Fonts, nil
}

// LoadSpriteSheets はスプライトシートリソースを読み込む
func (rl *DefaultResourceLoader) LoadSpriteSheets() (map[string]components.SpriteSheet, error) {
	// キャッシュがあれば返す
	if rl.cache.SpriteSheets != nil {
		return rl.cache.SpriteSheets, nil
	}

	// raw.tomlからSpriteSheet定義を読み込む
	rawMaster, err := rl.LoadRaws()
	if err != nil {
		return nil, fmt.Errorf("raw.tomlの読み込みに失敗: %w", err)
	}

	spriteSheets := make(map[string]components.SpriteSheet)

	// raw.tomlの定義に基づいてJSONファイルを読み込む
	for _, spriteSheetDef := range rawMaster.Raws.SpriteSheets {
		sheet, err := LoadSpriteSheetFromAseprite(spriteSheetDef.Path)
		if err != nil {
			return nil, fmt.Errorf("スプライトシート '%s' の読み込みに失敗: %w", spriteSheetDef.Name, err)
		}
		sheet.Name = spriteSheetDef.Name
		spriteSheets[spriteSheetDef.Name] = sheet
	}

	rl.cache.SpriteSheets = spriteSheets
	return spriteSheets, nil
}

// LoadRaws はRawデータを読み込む
func (rl *DefaultResourceLoader) LoadRaws() (*raw.Master, error) {
	// キャッシュがあれば返す
	if rl.cache.RawMaster != nil {
		return rl.cache.RawMaster, nil
	}

	rawMaster, err := raw.LoadFromFile(rl.config.RawsPath)
	if err != nil {
		return nil, err
	}
	rl.cache.RawMaster = &rawMaster

	return &rawMaster, nil
}
