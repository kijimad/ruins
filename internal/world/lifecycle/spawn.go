package lifecycle

import (
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// エラー定義
var (
	ErrItemGeneration   = errors.New("failed to generate item")
	ErrMemberGeneration = errors.New("failed to generate member")
	ErrEnemyGeneration  = errors.New("failed to generate enemy")
	ErrEffectGeneration = errors.New("failed to generate effect")

	// errNotNeutralNPC は中立でないメンバーを中立NPCとして生成しようとしたときに返す
	errNotNeutralNPC = errors.New("not a neutral NPC")
	// errNonPositiveCount は生成個数が正でないときに返す
	errNonPositiveCount = errors.New("count must be positive")
	// errMissingStatsComponents はHPかAbilitiesを持たないエンティティの最大HPを設定しようとしたときに返す
	errMissingStatsComponents = errors.New("entity does not have required components HP or Abilities")
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

	if err := giveStartingGear(world, playerEntity, name); err != nil {
		return gc.InvalidEntity, err
	}

	query.InvalidateSpatialIndex(world)
	return playerEntity, nil
}

// giveStartingGear は member の行の初期装備を装備させ、初期アイテムをバックパックへ持たせる。
// 所有者は生成中の member と確定しているので、世界からの検索を挟まず直接持たせる
func giveStartingGear(world w.World, owner ecs.Entity, name string) error {
	member, err := raw.FindMember(world.Resources.RawMaster, name)
	if err != nil {
		return err
	}
	for _, equip := range raw.PtrSlice(member.StartingEquips) {
		item, err := spawnItemBase(world, equip.Name)
		if err != nil {
			return fmt.Errorf("failed to spawn starting %s: %w", equip.Name, err)
		}
		if err := MoveToBackpack(world, item, owner); err != nil {
			return fmt.Errorf("failed to give starting %s: %w", equip.Name, err)
		}
		slot, ok := gc.ParseEquipmentSlot(equip.Slot)
		if !ok {
			return fmt.Errorf("invalid equipment slot %q for %s", equip.Slot, equip.Name)
		}
		MoveToEquip(world, item, owner, slot)
	}
	for _, it := range raw.PtrSlice(member.StartingItems) {
		for range it.Count {
			item, err := spawnItemBase(world, it.Name)
			if err != nil {
				return fmt.Errorf("failed to spawn starting %s: %w", it.Name, err)
			}
			if err := MoveToBackpack(world, item, owner); err != nil {
				return fmt.Errorf("failed to give starting %s: %w", it.Name, err)
			}
		}
	}
	return nil
}

// SpawnNeutralNPC はフィールド上に中立NPCを生成する（会話可能なNPC用）
func SpawnNeutralNPC(world w.World, pos consts.Coord[consts.Tile], name string) (ecs.Entity, error) {
	entitySpec, err := raw.NewMemberSpec(world.Resources.RawMaster, name)
	if err != nil {
		return gc.InvalidEntity, fmt.Errorf("failed to generate neutral NPC: %w", err)
	}

	if entitySpec.FactionNeutral == nil {
		return gc.InvalidEntity, fmt.Errorf("%w: %s", errNotNeutralNPC, name)
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

	// 在庫を持つ member は店を開く。生成経路に依らずここで積むことで、集落でも街マップの
	// マッププランナ経由でも同じ在庫を持たせる。売買と雇用はこの在庫を出し入れする
	if err := PopulateStock(world, npcEntity, name); err != nil {
		return gc.InvalidEntity, err
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
		maxAP := query.CalculateMaxActionPoints(world, npcEntity)
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
		return gc.InvalidEntity, fmt.Errorf("%w: %d", errNonPositiveCount, count)
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
		maxAP := query.CalculateMaxActionPoints(world, entity)
		turnBased := world.Components.TurnBased.Get(entity)
		turnBased.AP.Current = maxAP
		turnBased.AP.Max = maxAP
	}

	return nil
}

// setMaxStats はエンティティの最大HPを設定する
func setMaxStats(world w.World, entity ecs.Entity) error {
	if !world.Components.HP.Has(entity) || !world.Components.Abilities.Has(entity) {
		return fmt.Errorf("%w: %v", errMissingStatsComponents, entity)
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
		return gc.InvalidEntity, fmt.Errorf("%w: %d", errNonPositiveCount, count)
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
		return gc.InvalidEntity, fmt.Errorf("%w: %d", errNonPositiveCount, count)
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

// SpawnSplashText は画面中央にテキストをスプラッシュ表示するエフェクト専用エンティティを生成する。
// 場面の節目を大書きで知らせる演出に使う。face はフォント未ロードの環境では nil のまま作り、
// 描画しない環境でも生成できるようにする
func SpawnSplashText(world w.World, textStr string) {
	screenW, screenH := world.Resources.GetScreenDimensions()
	var face text.Face
	if tr := world.Resources.UIResources.Text; tr != nil {
		face = tr.SplashFontFace
	}
	effect := gc.NewSplashTextEffect(textStr, face, screenW, screenH)
	effectEntity := world.ECS.NewEntity()
	world.Components.VisualEffects.Add(effectEntity, &gc.VisualEffects{
		Effects: []gc.VisualEffect{effect},
	})
}
