package resources

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// Resources はゲーム固有のリソース管理を担当する
// 初期化時のみセットされ、あとから変更はされない
type Resources struct {
	// 静的
	ScreenDimensions ScreenDimensions
	SpriteSheets     map[string]components.SpriteSheet
	Fonts            map[string]Font
	Faces            map[string]text.Face
	UIResources      UIResources
	RawMaster        oapi.Raws
	SingletonEntity  ecs.Entity // シングルトンエンティティIDキャッシュ
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

// InitGameResources はゲームリソースを初期化する
func InitGameResources() *Resources {
	return &Resources{
		SpriteSheets: map[string]components.SpriteSheet{},
		Fonts:        map[string]Font{},
		Faces:        map[string]text.Face{},
		UIResources:  UIResources{},
	}
}
