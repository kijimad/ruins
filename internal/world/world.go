// Package world はゲームワールドの実装を提供する。
package world

import (
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/resources"

	ecs "github.com/x-hgg-x/goecs/v2"
)

// Updater はロジック更新を行うシステム
// Systemを再利用して状態を保持するのに使う
type Updater interface {
	// String はシステム名を返す
	String() string

	// Update はゲームロジックの更新処理を行う
	Update(world World) error
}

// Renderer は描画を行うシステム
type Renderer interface {
	// String はシステム名を返す
	String() string

	// Draw は描画処理を行う
	Draw(world World, screen *ebiten.Image) error
}

// World はゲーム全体に必要な情報を保持する
type World struct {
	Manager    *ecs.Manager
	Components *gc.Components
	Resources  *resources.Resources
	Updaters   map[string]Updater
	Renderers  map[string]Renderer
}

// InitWorld は初期化する
func InitWorld(c *gc.Components) (World, error) {
	manager := ecs.NewManager()
	err := c.InitializeComponents(manager)
	if err != nil {
		return World{}, err
	}
	return World{
		Manager:    manager,
		Components: c,
		Resources:  resources.InitGameResources(),
		Updaters:   make(map[string]Updater),
		Renderers:  make(map[string]Renderer),
	}, nil
}

// GetManager は World interfaceを満たすためのメソッド
func (w World) GetManager() *ecs.Manager {
	return w.Manager
}

// GetComponents は World interfaceを満たすためのメソッド
func (w World) GetComponents() interface{} {
	return w.Components
}
