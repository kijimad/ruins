package lifecycle

import (
	"errors"
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 定数定義
const (
	fieldSpriteSheet = "field" // オーバーワールドの地物・アイテムが使うスプライトシート名
)

// エラー定義
var (
	ErrItemGeneration   = errors.New("failed to generate item")
	ErrMemberGeneration = errors.New("failed to generate member")
	ErrEnemyGeneration  = errors.New("failed to generate enemy")
	ErrEffectGeneration = errors.New("failed to generate effect")
)

// initialPatrolDir はPatrol移動の初期方向をランダムに決定する。X軸方向で+1か-1を返す
func initialPatrolDir() consts.Tile {
	if rand.IntN(2) == 0 {
		return 1
	}
	return -1
}

// SpawnTile はタイルを生成する
// autoTileIndexが指定された場合、spriteKeyを動的に生成する（例: "wall_5"）
func SpawnTile(world w.World, tileName string, x consts.Tile, y consts.Tile, autoTileIndex *int) (ecs.Entity, error) {
	rawMaster := world.Resources.RawMaster
	entitySpec, err := raw.NewTileSpec(rawMaster, tileName, x, y, autoTileIndex)
	if err != nil {
		return gc.InvalidEntity, err
	}

	entity := world.Components.AddEntity(world.ECS, &entitySpec)
	return entity, nil
}

// SpawnPlayer はプレイヤーキャラクターを生成する
func SpawnPlayer(world w.World, pos consts.Coord[consts.Tile], name string) (ecs.Entity, error) {
	entitySpec, err := raw.NewPlayerSpec(world.Resources.RawMaster, name)
	if err != nil {
		return gc.InvalidEntity, fmt.Errorf("%w: %w", ErrMemberGeneration, err)
	}

	entitySpec.GridElement = &gc.GridElement{Coord: pos}
	center := consts.TileCenterToWorld(pos)
	entitySpec.Camera = &gc.Camera{
		Scale:   gc.CameraMinScale,
		ScaleTo: gc.CameraMinScale,
		Pos:     center,
		Target:  center,
		Pitch:   gc.CameraDefaultPitch,
		Dist:    gc.CameraDefaultDist,
	}
	entitySpec.Wallet = &gc.Wallet{Currency: 10000}
	playerEntity := world.Components.AddEntity(world.ECS, &entitySpec)

	if err := FullRecover(world, playerEntity); err != nil {
		return gc.InvalidEntity, fmt.Errorf("player recovery failed: %w", err)
	}
	world.Components.WeightDirty.Add(playerEntity, &gc.WeightDirty{})

	// 初期装備として松明を持たせて装備する。プレイヤーは内蔵光源を持たないので、
	// これが明かりになる。外すと暗くなる。StatsChangedSystem が装備の光源を owner へ転写する。
	// 所有者は生成中のプレイヤーと確定しているので、世界からの検索を挟まず直接持たせる
	torch, err := spawnItemBase(world, "torch")
	if err != nil {
		return gc.InvalidEntity, fmt.Errorf("failed to spawn starting torch: %w", err)
	}
	if err := MoveToBackpack(world, torch, playerEntity); err != nil {
		return gc.InvalidEntity, fmt.Errorf("failed to give starting torch: %w", err)
	}
	MoveToEquip(world, torch, playerEntity, gc.SlotWeapon1)

	query.InvalidateSpatialIndex(world)
	return playerEntity, nil
}

// SpawnNeutralNPC はフィールド上に中立NPCを生成する（会話可能なNPC用）
func SpawnNeutralNPC(world w.World, pos consts.Coord[consts.Tile], name string) (ecs.Entity, error) {
	entitySpec, err := raw.NewMemberSpec(world.Resources.RawMaster, name)
	if err != nil {
		return gc.InvalidEntity, fmt.Errorf("failed to generate neutral NPC: %w", err)
	}

	if entitySpec.FactionNeutral == nil {
		return gc.InvalidEntity, fmt.Errorf("'%s' is not a neutral NPC", name)
	}
	if entitySpec.Dialog == nil {
		return gc.InvalidEntity, fmt.Errorf("'%s' has no dialog data", name)
	}

	entitySpec.GridElement = &gc.GridElement{Coord: pos}

	if entitySpec.SoloAI != nil {
		solo := entitySpec.SoloAI
		solo.SubState = gc.AIStateWaiting
		solo.StartSubStateTurn = 1
		solo.DurationSubStateTurns = consts.Turn(2 + rand.IntN(3))
		solo.Origin = pos
		solo.PatrolDir.X = initialPatrolDir()
		solo.ViewDistance = consts.AIVisionDistance
	}

	npcEntity := world.Components.AddEntity(world.ECS, &entitySpec)
	if err := FullRecover(world, npcEntity); err != nil {
		return gc.InvalidEntity, fmt.Errorf("NPC recovery failed: %w", err)
	}

	// 商人は品揃えを在庫として持つ。生成経路に依らずここで積むことで、集落でも街マップの
	// マッププランナ経由でも同じ在庫を持たせる。売買と雇用はこの在庫を出し入れする
	if name == "merchant" {
		if err := PopulateMerchantStock(world, npcEntity, world.Resources.Config.RNG); err != nil {
			return gc.InvalidEntity, fmt.Errorf("failed to stock merchant: %w", err)
		}
	}

	query.InvalidateSpatialIndex(world)
	return npcEntity, nil
}

// SpawnEnemy はフィールド上に敵キャラクターを生成する
func SpawnEnemy(world w.World, pos consts.Coord[consts.Tile], name string) (ecs.Entity, error) {
	entitySpec, err := raw.NewEnemySpec(world.Resources.RawMaster, name)
	if err != nil {
		return gc.InvalidEntity, fmt.Errorf("%w: %w", ErrEnemyGeneration, err)
	}

	entitySpec.GridElement = &gc.GridElement{Coord: pos}
	if entitySpec.SoloAI == nil {
		return gc.InvalidEntity, fmt.Errorf("enemy entity has no AI specified: %s", entitySpec.Name)
	}
	solo := entitySpec.SoloAI
	solo.SubState = gc.AIStateWaiting
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = consts.Turn(2 + rand.IntN(3))
	solo.Origin = pos
	solo.PatrolDir.X = initialPatrolDir()
	solo.ViewDistance = consts.AIVisionDistance
	entitySpec.Interactable = &gc.Interactable{
		Interactions: []gc.InteractionKind{gc.InteractionMelee},
	}

	npcEntity := world.Components.AddEntity(world.ECS, &entitySpec)
	if err := FullRecover(world, npcEntity); err != nil {
		return gc.InvalidEntity, fmt.Errorf("enemy recovery failed: %w", err)
	}

	if world.Components.TurnBased.Has(npcEntity) {
		actionPoints := world.Components.TurnBased.Get(npcEntity)
		maxAP, err := query.CalculateMaxActionPoints(world, npcEntity)
		if err != nil {
			return gc.InvalidEntity, fmt.Errorf("AP calculation failed: %w", err)
		}
		actionPoints.AP.Current = maxAP
		actionPoints.AP.Max = maxAP
	}

	query.InvalidateSpatialIndex(world)
	return npcEntity, nil
}

// SpawnBackpackItem はバックパック内にアイテムを count 個生成する。
// 1個1エンティティなので count 回生成する。戻り値はスタック代表としての1個で、
// 同一スタックの個体はどれも等価。個数や全個体は代表から query 側で導出する。
func SpawnBackpackItem(world w.World, name string, count int) (ecs.Entity, error) {
	if count <= 0 {
		return gc.InvalidEntity, fmt.Errorf("count must be positive: %d", count)
	}

	// バックパックはプレイヤーの所有物なので、プレイヤー不在での生成は不変条件違反として返す。
	// 所有者なしのバックパック品は束ねの所有者一致に掛からず、silent な迷子になる
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return gc.InvalidEntity, fmt.Errorf("failed to spawn backpack item: %w", err)
	}

	last := gc.InvalidEntity
	for range count {
		item, err := spawnItemBase(world, name)
		if err != nil {
			return gc.InvalidEntity, err
		}
		if err := MoveToBackpack(world, item, playerEntity); err != nil {
			return item, fmt.Errorf("failed to move to backpack: %w", err)
		}
		last = item
	}
	return last, nil
}

// spawnItemBase はLocationなしでアイテムエンティティを1個生成する内部関数。
// 1個1エンティティなので個数は扱わない。N個はラッパがN回呼んで作る。
func spawnItemBase(world w.World, name string) (ecs.Entity, error) {
	entitySpec, err := raw.NewItemSpec(world.Resources.RawMaster, name)
	if err != nil {
		return gc.InvalidEntity, fmt.Errorf("%w: %w", ErrItemGeneration, err)
	}

	entity := world.Components.AddEntity(world.ECS, &entitySpec)
	// 腐敗の起点を今の総ターン数で刻む。GameTime は world 生成時から常在する
	if world.Components.Perishable.Has(entity) {
		world.Components.Perishable.Get(entity).RotUpdatedTurn = query.GetGameTime(world).TotalTurns
	}
	return entity, nil
}

// FullRecover はエンティティのHP/APを全回復する
func FullRecover(world w.World, entity ecs.Entity) error {
	if err := setMaxStats(world, entity); err != nil {
		return fmt.Errorf("failed to set max HP: %w", err)
	}

	hp := world.Components.HP.Get(entity)
	if hp == nil {
		return fmt.Errorf("HP component is missing")
	}
	hp.Current = hp.Max

	if world.Components.TurnBased.Has(entity) {
		maxAP, err := query.CalculateMaxActionPoints(world, entity)
		if err != nil {
			return fmt.Errorf("AP calculation failed: %w", err)
		}
		turnBased := world.Components.TurnBased.Get(entity)
		turnBased.AP.Current = maxAP
		turnBased.AP.Max = maxAP
	}

	return nil
}

// setMaxStats はエンティティの最大HPを設定する
func setMaxStats(world w.World, entity ecs.Entity) error {
	if !world.Components.HP.Has(entity) || !world.Components.Abilities.Has(entity) {
		return fmt.Errorf("entity %v does not have required components (HP or Abilities)", entity)
	}

	hp := world.Components.HP.Get(entity)
	abils := world.Components.Abilities.Get(entity)

	if abils.Vitality.Total == 0 {
		abils.Vitality.Total = abils.Vitality.Base
	}
	if abils.Strength.Total == 0 {
		abils.Strength.Total = abils.Strength.Base
	}
	if abils.Sensation.Total == 0 {
		abils.Sensation.Total = abils.Sensation.Base
	}
	if abils.Dexterity.Total == 0 {
		abils.Dexterity.Total = abils.Dexterity.Base
	}
	if abils.Agility.Total == 0 {
		abils.Agility.Total = abils.Agility.Base
	}
	if abils.Defense.Total == 0 {
		abils.Defense.Total = abils.Defense.Base
	}

	hp.Max = formula.CalcHP(abils.Vitality.Total, abils.Strength.Total, abils.Sensation.Total)
	hp.Current = hp.Max

	return nil
}

// SpawnStorageItem は収納内にアイテムを count 個生成する。戻り値はスタック代表としての1個。
func SpawnStorageItem(world w.World, itemName string, count int, storage ecs.Entity) (ecs.Entity, error) {
	if count <= 0 {
		return gc.InvalidEntity, fmt.Errorf("count must be positive: %d", count)
	}
	last := gc.InvalidEntity
	for range count {
		item, err := spawnItemBase(world, itemName)
		if err != nil {
			return gc.InvalidEntity, err
		}
		if err := MoveToStorage(world, item, storage); err != nil {
			return item, fmt.Errorf("failed to move to storage: %w", err)
		}
		last = item
	}
	return last, nil
}

// SpawnFieldItem はフィールド上に同じ位置へアイテムを count 個生成する。戻り値はスタック代表としての1個。
func SpawnFieldItem(world w.World, itemName string, x consts.Tile, y consts.Tile, count int) (ecs.Entity, error) {
	if count <= 0 {
		return gc.InvalidEntity, fmt.Errorf("count must be positive: %d", count)
	}
	last := gc.InvalidEntity
	for range count {
		item, err := spawnItemBase(world, itemName)
		if err != nil {
			return gc.InvalidEntity, err
		}
		MoveToField(world, item, nil)
		world.Components.GridElement.Add(item, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
		last = item
	}
	return last, nil
}

// SpawnVisualEffect はエンティティの位置にエフェクト専用エンティティを生成する
func SpawnVisualEffect(target ecs.Entity, effect gc.VisualEffect, world w.World) {
	if !world.Components.GridElement.Has(target) {
		return
	}

	gridElement := world.Components.GridElement.Get(target)

	effectEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(effectEntity, &gc.GridElement{Coord: gridElement.Coord})
	world.Components.VisualEffects.Add(effectEntity, &gc.VisualEffects{
		Effects: []gc.VisualEffect{effect},
	})
}
