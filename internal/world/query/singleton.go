package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// GetSingleton はシングルトンエンティティからコンポーネントを取得する。未初期化の場合はnilを返す
func GetSingleton[T any](world w.World, comp *ecs.Map[T]) *T {
	if !comp.Has(world.Resources.SingletonEntity) {
		return nil
	}
	return comp.Get(world.Resources.SingletonEntity)
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
	return GetSingleton[gc.Dungeon](world, world.Components.Dungeon)
}

// IsOnOverworld は現在地がオーバーワールドかを返す。オーバーワールド固有の振る舞い
// (霜・寒波前線の気温/移動効果・帯シフト)の gate に使い、場所判定をこの1関数へ集約する。
//
// 共存方式では遺跡へ入っても SeamlessBand は退避されたまま Active に残るため、Front.Active や
// SeamlessBand.Active を「オーバーワールドにいる」の代理に使うと遺跡内へ効果が漏れる。現ステージが
// オーバーワールド帯ステージかで判定する。
//
// interim: 恒久形(設計 65.md Phase H3)ではオーバーワールド固有データを現ステージのメタが持つかで
// 判定する。それまでは種別判定の乱立を防ぐためここに集約し、置換点を1箇所に閉じ込める。
func IsOnOverworld(world w.World) bool {
	return GetDungeon(world).CurrentStage == gc.NewOverworldStage()
}

// GetGameLog はシングルトンエンティティからGameLogストアを取得する
func GetGameLog(world w.World) *gamelog.SafeSlice {
	gl := GetSingleton[gc.GameLog](world, world.Components.GameLog)
	if gl == nil {
		return nil
	}
	return gl.Store
}

// SetDungeon はシングルトンエンティティにDungeonを設定する。
// nilを渡すとダンジョン未開始として扱い、コンポーネントを取り除く
func SetDungeon(world w.World, dungeon *gc.Dungeon) {
	entity := world.Resources.SingletonEntity
	comp := world.Components.Dungeon
	if dungeon == nil {
		if comp.Has(entity) {
			comp.Remove(entity)
		}
		return
	}
	if comp.Has(entity) {
		comp.Set(entity, dungeon)
	} else {
		comp.Add(entity, dungeon)
	}
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

// UpdateCharacterPositionInIndex は移動に伴い空間インデックスのキャラ位置を増分更新する。
// 無効化→全再構築のチャーンを避けるための入口。
// インデックスが未構築なら何もしないため、
// 意図的に GetSpatialIndex ではなく GetSingleton を使う。
func UpdateCharacterPositionInIndex(world w.World, entity ecs.Entity, from, to consts.Coord[consts.Tile]) {
	si := GetSingleton[gc.SpatialIndex](world, world.Components.SpatialIndex)
	if si == nil {
		return
	}
	si.MoveCharacter(from, to, entity)
}

// buildSpatialIndex は壁・キャラクター・プレイヤーの位置をスキャンしてインデックスを構築する
func buildSpatialIndex(world w.World, si *gc.SpatialIndex) {
	dungeon := GetDungeon(world)
	if dungeon == nil {
		return
	}
	si.MapWidth = dungeon.Level.TileWidth
	si.MapHeight = dungeon.Level.TileHeight
	si.BlockPass = make(map[gc.GridElement]bool)
	si.Characters = make(map[gc.GridElement]ecs.Entity)
	si.PlayerEntity = nil

	// 静的障害物のインデックス構築。退避中ステージのタイルは現ステージの座標索引に混ぜない
	blockPassQuery := ActiveFilter2[gc.GridElement, gc.BlockPass](world).Query()
	for blockPassQuery.Next() {
		entity := blockPassQuery.Entity()
		if world.Components.Dead.Has(entity) {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		si.BlockPass[*grid] = true
	}

	// キャラクター位置のインデックス構築。退避中ステージのキャラクターは現ステージに混ぜない
	characterQuery := ActiveFilter1[gc.GridElement](world).Query()
	for characterQuery.Next() {
		entity := characterQuery.Entity()
		if world.Components.Dead.Has(entity) {
			continue
		}
		isCharacter := world.Components.Player.Has(entity) ||
			world.Components.SoloAI.Has(entity) ||
			world.Components.SquadAI.Has(entity) ||
			world.Components.SquadMember.Has(entity)
		if !isCharacter {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		si.Characters[*grid] = entity
		if world.Components.Player.Has(entity) {
			e := entity
			si.PlayerEntity = &e
		}
	}

	si.Built = true
	si.BuildCount++ // 再構築チャーン検知用の累積カウンタ
}
