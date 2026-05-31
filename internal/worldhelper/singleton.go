package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetSingleton はシングルトンエンティティからコンポーネントを取得する。未初期化の場合はnilを返す
func GetSingleton[T any](world w.World, comp *ecs.SliceComponent) *T {
	data := comp.Get(world.Resources.SingletonEntity)
	if data == nil {
		return nil
	}
	return data.(*T)
}

// GetTurnState はシングルトンエンティティからターン状態を取得する
func GetTurnState(world w.World) *gc.TurnState {
	return GetSingleton[gc.TurnState](world, world.Components.TurnState)
}

// GetGameProgress はシングルトンエンティティからGameProgressを取得する
func GetGameProgress(world w.World) *gc.GameProgress {
	return GetSingleton[gc.GameProgress](world, world.Components.GameProgress)
}

// GetDungeon はシングルトンエンティティからDungeonを取得する
// ダンジョン未開始の場合はnilを返す
func GetDungeon(world w.World) *gc.Dungeon {
	return GetSingleton[gc.Dungeon](world, world.Components.DungeonState)
}

// GetGameLog はシングルトンエンティティからGameLogストアを取得する
func GetGameLog(world w.World) *gamelog.SafeSlice {
	gl := GetSingleton[gc.GameLog](world, world.Components.GameLog)
	if gl == nil {
		return nil
	}
	return gl.Store
}

// SetDungeon はシングルトンエンティティにDungeonを設定する
func SetDungeon(world w.World, dungeon *gc.Dungeon) {
	world.Resources.SingletonEntity.AddComponent(world.Components.DungeonState, dungeon)
}
