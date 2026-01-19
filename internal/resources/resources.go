package resources

import (
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/components"
)

// Resources は具体的なリソース実装
// ゲーム固有のリソース管理を担当する
// engine/resources.ResourceProviderインターフェースを実装する
type Resources struct {
	ScreenDimensions   *ScreenDimensions
	SpriteSheets       *map[string]components.SpriteSheet
	Fonts              *map[string]Font
	Faces              *map[string]text.Face
	Dungeon            *Dungeon
	RawMaster          interface{}
	UIResources        *UIResources
	TurnManager        interface{}
	RNG                *rand.Rand // ゲーム全体で使う乱数生成器
	SelectedWeaponSlot int        // 選択中の武器スロット番号（1-5）
}

// ScreenDimensions contains current screen dimensions
type ScreenDimensions struct {
	Width  int
	Height int
}

// GetScreenDimensions はスクリーン寸法を取得する
func (r *Resources) GetScreenDimensions() (width, height int) {
	if r.ScreenDimensions == nil {
		return 0, 0
	}
	return r.ScreenDimensions.Width, r.ScreenDimensions.Height
}

// SetScreenDimensions はスクリーン寸法を設定する
func (r *Resources) SetScreenDimensions(width, height int) {
	if r.ScreenDimensions == nil {
		r.ScreenDimensions = &ScreenDimensions{}
	}
	r.ScreenDimensions.Width = width
	r.ScreenDimensions.Height = height
}

// InitializeResources は ResourceInitializer インターフェースを実装
func (r *Resources) InitializeResources() error {
	r.ScreenDimensions = &ScreenDimensions{}
	r.SpriteSheets = &map[string]components.SpriteSheet{}
	r.Fonts = &map[string]Font{}
	r.Faces = &map[string]text.Face{}
	r.UIResources = &UIResources{}
	r.RawMaster = nil
	r.TurnManager = nil
	return nil
}

// InitGameResources はゲームリソースを初期化する
func InitGameResources() *Resources {
	return &Resources{
		ScreenDimensions:   &ScreenDimensions{},
		SpriteSheets:       &map[string]components.SpriteSheet{},
		Fonts:              &map[string]Font{},
		Faces:              &map[string]text.Face{},
		UIResources:        &UIResources{},
		RNG:                nil,
		SelectedWeaponSlot: 1, // デフォルトで武器スロット1を選択
	}
}
