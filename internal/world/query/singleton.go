package query

import (
	"fmt"
	"reflect"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetSingleton はシングルトンエンティティからコンポーネントを取得する。未初期化の場合はnilを返す。
// comp Component[T] で受けることで、型引数 T とコンポーネントの取り違えをコンパイラが検出する。
func GetSingleton[T any](world w.World, comp gc.Component[T]) *T {
	data := comp.Get(world.Resources.SingletonEntity)
	if data == nil {
		return nil
	}
	v, ok := data.(*T)
	if !ok {
		panic(fmt.Sprintf("GetSingleton: シングルトンが %s を保持していない", reflect.TypeFor[T]()))
	}
	return v
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
	gc.AddComponent(world.Resources.SingletonEntity, world.Components.DungeonState, dungeon)
}

// GetSpatialIndex はシングルトンから空間インデックスを取得する
// 未構築の場合は構築する
func GetSpatialIndex(world w.World) *gc.SpatialIndex {
	si := GetSingleton[gc.SpatialIndex](world, world.Components.SpatialIndex)
	if si == nil {
		return nil
	}
	if !si.Built {
		buildSpatialIndex(world, si)
	}
	if si.MapWidth == 0 || si.MapHeight == 0 {
		return nil
	}
	return si
}

// InvalidateSpatialIndex は空間インデックスを無効化する。次回アクセス時に再構築される
func InvalidateSpatialIndex(world w.World) {
	si := GetSingleton[gc.SpatialIndex](world, world.Components.SpatialIndex)
	if si != nil {
		si.Invalidate()
	}
}

// buildSpatialIndex は壁・キャラクター・プレイヤーの位置をスキャンしてインデックスを構築する
func buildSpatialIndex(world w.World, si *gc.SpatialIndex) {
	dungeon := GetDungeon(world)
	if dungeon == nil {
		return
	}
	si.MapWidth = int(dungeon.Level.TileWidth)
	si.MapHeight = int(dungeon.Level.TileHeight)
	si.BlockPass = make(map[gc.GridElement]bool)
	si.Characters = make(map[gc.GridElement]ecs.Entity)
	si.PlayerEntity = nil

	// 静的障害物のインデックス構築
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.BlockPass,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if entity.HasComponent(world.Components.Dead) {
			return
		}
		grid := world.Components.GridElement.MustGet(entity)
		si.BlockPass[*grid] = true
	}))

	// キャラクター位置のインデックス構築
	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if entity.HasComponent(world.Components.Dead) {
			return
		}
		isCharacter := entity.HasComponent(world.Components.Player) ||
			entity.HasComponent(world.Components.AI) ||
			entity.HasComponent(world.Components.SquadMember)
		if !isCharacter {
			return
		}
		grid := world.Components.GridElement.MustGet(entity)
		si.Characters[*grid] = entity
		if entity.HasComponent(world.Components.Player) {
			e := entity
			si.PlayerEntity = &e
		}
	}))

	si.Built = true
}
