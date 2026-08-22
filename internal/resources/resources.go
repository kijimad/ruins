package resources

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/mlange-42/ark/ecs"
)

// Resources はゲーム固有のリソース管理を担当する
// engine/resources.ResourceProviderインターフェースを実装する
// 初期化時のみセットされ、あとから変更はされない
type Resources struct {
	ScreenDimensions ScreenDimensions
	SpriteSheets     map[string]components.SpriteSheet
	Fonts            map[string]Font
	Faces            map[string]text.Face
	UIResources      UIResources
	RawMaster        oapi.Raws
	I18N             i18n.Catalog   // 国際化のマスタ。全言語の訳を持つ読み取り専用データ。現在言語は UserSettings が持ち query.T が引く
	Config           *config.Config // 実行設定。起動時に注入し、ResetForNewGame を跨いで持続する。現在言語のミラー UserSettings とは別
	SingletonEntity  ecs.Entity     // シングルトンエンティティIDキャッシュ

	// InputSource は Action の入力供給源。nil なら本番どおりキーボードから変換する。
	// 再生ドライバだけが Action 列を返す供給源を差し、キー入力を経由せず本番フローを駆動する。
	// world 単位で持つことでグローバル可変状態を作らず、押し込んだ先の state にも同じ源が効く
	InputSource inputmapper.Source
}

// ScreenDimensions contains current screen dimensions
type ScreenDimensions struct {
	Width  int
	Height int
}

// GetScreenDimensions はスクリーン寸法を取得する
func (r *Resources) GetScreenDimensions() (width, height int) {
	return r.ScreenDimensions.Width, r.ScreenDimensions.Height
}

// SetScreenDimensions はスクリーン寸法を設定する
func (r *Resources) SetScreenDimensions(width, height int) {
	r.ScreenDimensions.Width = width
	r.ScreenDimensions.Height = height
}

// InitializeResources は engine/resources.ResourceInitializer インターフェースを実装する
func (r *Resources) InitializeResources() error {
	*r = *InitGameResources()
	return nil
}

// InitGameResources はゲームリソースを初期化する
func InitGameResources() *Resources {
	return &Resources{
		SpriteSheets: map[string]components.SpriteSheet{},
		Fonts:        map[string]Font{},
		Faces:        map[string]text.Face{},
		UIResources:  UIResources{},
		// 全言語の訳を持つ不変マスタを常に持たせる。どのワールドでも res.I18N が非 nil になる。
		I18N: i18n.NewCatalog(),
	}
}
