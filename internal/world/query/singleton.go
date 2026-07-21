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

// GetWeaponSelection はシングルトンから選択中の武器スロットを取得する
func GetWeaponSelection(world w.World) *gc.WeaponSelection {
	return GetSingleton[gc.WeaponSelection](world, world.Components.WeaponSelection)
}

// GetGameTime はシングルトンからゲーム内時間を取得する
func GetGameTime(world w.World) *gc.GameTime {
	return GetSingleton[gc.GameTime](world, world.Components.GameTime)
}

// GetVisionState はシングルトンから視界計算の一時状態を取得する
func GetVisionState(world w.World) *gc.VisionState {
	return GetSingleton[gc.VisionState](world, world.Components.VisionState)
}

// stageMetaEntity は key に束縛された StageMeta エンティティを返す。
// クエリ反復は途中 return するとワールドロックが残るため、最後まで回してから返す。
func stageMetaEntity(world w.World, key gc.StageKey) (ecs.Entity, bool) {
	var found ecs.Entity
	ok := false
	q := ecs.NewFilter2[gc.StageMeta, gc.StageBound](world.ECS).Query()
	for q.Next() {
		if !ok && world.Components.StageBound.Get(q.Entity()).Key == key {
			found = q.Entity()
			ok = true
		}
	}
	return found, ok
}

// currentStageEntity は現ステージ CurrentStage のメタエンティティを返す。
func currentStageEntity(world w.World) (ecs.Entity, bool) {
	return stageMetaEntity(world, GetDungeon(world).CurrentStage)
}

// ensureStageMetaEntity は key に束縛された StageMeta エンティティを確保して返す。
// 無ければ生成する。返り値は必ず生存エンティティなので、呼び出し側は Has/Get を安全に使える。
func ensureStageMetaEntity(world w.World, key gc.StageKey) ecs.Entity {
	if e, ok := stageMetaEntity(world, key); ok {
		return e
	}
	e := world.ECS.NewEntity()
	world.Components.StageBound.Add(e, &gc.StageBound{Key: key})
	world.Components.StageMeta.Add(e, gc.NewStageMeta())
	return e
}

// EnsureStageMeta は key に束縛された StageMeta を確保して返す。無ければ生成する。
// ステージ生成時に呼び、そのステージのフィールド寸法などを書き込む。
// StageBound を直接付けるため、生成物を束縛する stage.Bind の対象からは外れる。
func EnsureStageMeta(world w.World, key gc.StageKey) *gc.StageMeta {
	return world.Components.StageMeta.Get(ensureStageMetaEntity(world, key))
}

// GetCurrentStageMeta は現ステージのメタを返す。未生成なら nil。
func GetCurrentStageMeta(world w.World) *gc.StageMeta {
	e, ok := currentStageEntity(world)
	if !ok {
		return nil
	}
	return world.Components.StageMeta.Get(e)
}

// GetSeamlessBand は現ステージが持つ帯の永続状態を返す。持たなければ nil。
// nil はオーバーワールドでないことを意味する。
func GetSeamlessBand(world w.World) *gc.SeamlessBand {
	e, ok := currentStageEntity(world)
	if !ok || !world.Components.SeamlessBand.Has(e) {
		return nil
	}
	return world.Components.SeamlessBand.Get(e)
}

// EnsureSeamlessBand は現ステージのメタに帯の永続状態を確保して返す。
// オーバーワールドを現ステージに確定してから呼ぶ。以後この帯データの有無が
// オーバーワールド判定を兼ねる。
func EnsureSeamlessBand(world w.World) *gc.SeamlessBand {
	e := ensureStageMetaEntity(world, GetDungeon(world).CurrentStage)
	if !world.Components.SeamlessBand.Has(e) {
		world.Components.SeamlessBand.Add(e, &gc.SeamlessBand{})
	}
	return world.Components.SeamlessBand.Get(e)
}

// IsOnOverworld は現在地がオーバーワールドかを返す。オーバーワールド固有の振る舞い
// (霜・寒波前線の気温/移動効果・帯シフト)の gate に使い、場所判定をこの1関数へ集約する。
//
// 種別・深度・名前で判定せず、現ステージのメタが帯データ SeamlessBand を持つかで判定する。
// オーバーワールドもダンジョン階も同じ「ステージ」で、違いは保有データだけという設計に従う。
// 帯データは遺跡進入で退避され現ステージから外れるため、遺跡内へ効果が漏れない。
func IsOnOverworld(world w.World) bool {
	e, ok := currentStageEntity(world)
	return ok && world.Components.SeamlessBand.Has(e)
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
	meta := GetCurrentStageMeta(world)
	if dungeon == nil || meta == nil {
		return
	}
	si.MapWidth = meta.Level.TileWidth
	si.MapHeight = meta.Level.TileHeight
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
