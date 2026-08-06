package resources

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/mlange-42/ark/ecs"
)

// Resources はゲーム固有のリソース管理を担当する
// engine/resources.ResourceProviderインターフェースを実装する
// 初期化時のみセットされ、あとから変更はされない
type Resources struct {
	// 静的
	ScreenDimensions ScreenDimensions
	SpriteSheets     map[string]components.SpriteSheet
	Fonts            map[string]Font
	Faces            map[string]text.Face
	UIResources      UIResources
	RawMaster        oapi.Raws
	SingletonEntity  ecs.Entity       // シングルトンエンティティIDキャッシュ
	I18N             *i18n.Translator // 国際化の翻訳器。呼び出し側は res.I18N.T で訳を引く
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
		// 源泉言語 en の翻訳器を常に持たせ、どのワールドでも res.I18N を非 nil にする。表示言語は起動後に config から反映する。
		I18N: i18n.NewDefault(),
	}
}
