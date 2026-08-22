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
	Updaters   map[string]Updater
	Renderers  map[string]Renderer
}

// InitWorld は初期化する。config はシングルトンの初期化で読むので、構築前に渡す。
func InitWorld(c *gc.Components, cfg *config.Config) (World, error) {
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
	world.Resources.Config = cfg

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
	world.Components.Dungeon.Add(singleton, gc.NewDungeon())
	world.Components.TurnState.Add(singleton, gc.NewTurnState())
	world.Components.SpatialIndex.Add(singleton, gc.NewSpatialIndex())
	world.Components.WeaponSelection.Add(singleton, &gc.WeaponSelection{Slot: 1})
	world.Components.GameTime.Add(singleton, &gc.GameTime{})
	world.Components.VisionState.Add(singleton, gc.NewVisionState())
	// config は構築時に渡されているので、設定言語をそのまま種にする。
	world.Components.UserSettings.Add(singleton, gc.NewUserSettings(world.Resources.Config.User.Language))
	world.Components.AuctionHistory.Add(singleton, gc.NewAuctionHistory())
	world.Resources.SingletonEntity = singleton
}

// ResetForNewGame は前のゲームの全実体を消し、シングルトンを作り直す。
// 新しいゲームを始める経路が呼ぶ。ロードは save 側の ECS.Reset が同じ役目を担う
func (world World) ResetForNewGame() {
	var clearEntities []ecs.Entity
	clearQuery := ecs.NewUnsafeFilter(world.ECS).Query()
	for clearQuery.Next() {
		clearEntities = append(clearEntities, clearQuery.Entity())
	}
	for _, e := range clearEntities {
		world.ECS.RemoveEntity(e)
	}
	world.InitSingleton()
}

// GetWorld は entities.World インターフェースを満たすためのメソッド
func (world World) GetWorld() *ecs.World {
	return world.ECS
}
