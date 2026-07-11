// Package world はゲームワールドの実装を提供する。
package world

import (
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/mlange-42/ark/ecs"
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
	ECS        *ecs.World
	Components *gc.Components
	Resources  *resources.Resources
	Config     *config.Config
	Updaters   map[string]Updater
	Renderers  map[string]Renderer
}

// InitWorld は初期化する
func InitWorld(c *gc.Components) (World, error) {
	arkWorld := ecs.NewWorld()
	if err := c.InitializeComponents(arkWorld); err != nil {
		return World{}, err
	}

	world := World{
		ECS:        arkWorld,
		Components: c,
		Resources:  resources.InitGameResources(),
		Updaters:   make(map[string]Updater),
		Renderers:  make(map[string]Renderer),
	}

	world.InitSingleton()

	return world, nil
}

// InitSingleton はシングルトンエンティティを新規作成してIDを保存する
func (world World) InitSingleton() {
	singleton := world.ECS.NewEntity()
	world.Components.GameLog.Add(singleton, &gc.GameLog{
		Store: gamelog.NewSafeSlice(gamelog.GameLogMaxSize),
	})
	world.Components.GameProgress.Add(singleton, gc.NewGameProgress())
	world.Components.DungeonState.Add(singleton, gc.NewDungeon())
	world.Components.TurnState.Add(singleton, gc.NewTurnState())
	world.Components.SpatialIndex.Add(singleton, gc.NewSpatialIndex())
	world.Resources.SingletonEntity = singleton
}

// GetWorld は entities.World インターフェースを満たすためのメソッド
func (world World) GetWorld() *ecs.World {
	return world.ECS
}
