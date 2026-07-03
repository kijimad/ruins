package lifecycle

import (
	"errors"
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/engine/entities"
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// 定数定義
const (
	cameraNormalScale = 0.6 // カメラの通常スケール
	// AIVisionDistance はAIエンティティの視界距離（タイル単位）
	AIVisionDistance = 5
)

// エラー定義
var (
	ErrItemGeneration   = errors.New("アイテムの生成に失敗しました")
	ErrMemberGeneration = errors.New("メンバーの生成に失敗しました")
	ErrEnemyGeneration  = errors.New("敵の生成に失敗しました")
	ErrEffectGeneration = errors.New("エフェクトの生成に失敗しました")
)

// initialPatrolDir はPatrol移動の初期方向をランダムに決定する。X軸方向で+1か-1を返す
func initialPatrolDir() int {
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
		return consts.InvalidEntity, err
	}

	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, entitySpec)

	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return consts.InvalidEntity, err
	}
	if len(entitiesSlice) == 0 {
		return consts.InvalidEntity, fmt.Errorf("エンティティの生成に失敗しました")
	}
	return entitiesSlice[0], nil
}

// SpawnPlayer はプレイヤーキャラクターを生成する
func SpawnPlayer(world w.World, tileX int, tileY int, name string) (ecs.Entity, error) {
	componentList := entities.ComponentList[gc.EntitySpec]{}
	entitySpec, err := raw.NewPlayerSpec(world.Resources.RawMaster, name)
	if err != nil {
		return consts.InvalidEntity, fmt.Errorf("%w: %v", ErrMemberGeneration, err)
	}

	skills := gc.NewSkills()
	entitySpec.Skills = skills
	entitySpec.CharModifiers = gc.RecalculateCharModifiers(skills, nil, nil)

	entitySpec.GridElement = &gc.GridElement{X: consts.Tile(tileX), Y: consts.Tile(tileY)}
	tileSize := float64(consts.TileSize)
	initialX := float64(tileX)*tileSize + tileSize/2
	initialY := float64(tileY)*tileSize + tileSize/2
	entitySpec.Camera = &gc.Camera{
		Scale:   cameraNormalScale,
		ScaleTo: cameraNormalScale,
		X:       initialX,
		Y:       initialY,
		TargetX: initialX,
		TargetY: initialY,
	}
	entitySpec.Wallet = &gc.Wallet{Currency: 10000}
	entitySpec.HealthStatus = &gc.HealthStatus{}
	componentList.Entities = append(componentList.Entities, entitySpec)
	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return consts.InvalidEntity, err
	}
	if len(entitiesSlice) != 1 {
		return consts.InvalidEntity, fmt.Errorf("プレイヤーエンティティの生成に失敗しました: 予期しないエンティティ数=%d", len(entitiesSlice))
	}
	playerEntity := entitiesSlice[0]

	if err := FullRecover(world, playerEntity); err != nil {
		return consts.InvalidEntity, fmt.Errorf("プレイヤーの回復処理エラー: %w", err)
	}
	playerEntity.AddComponent(world.Components.WeightDirty, &gc.WeightDirty{})

	query.InvalidateSpatialIndex(world)
	return playerEntity, nil
}

// SpawnNeutralNPC はフィールド上に中立NPCを生成する（会話可能なNPC用）
func SpawnNeutralNPC(world w.World, tileX int, tileY int, name string) (ecs.Entity, error) {
	componentList := entities.ComponentList[gc.EntitySpec]{}
	entitySpec, err := raw.NewMemberSpec(world.Resources.RawMaster, name)
	if err != nil {
		return consts.InvalidEntity, fmt.Errorf("中立NPC生成エラー: %w", err)
	}

	if entitySpec.FactionType == nil || *entitySpec.FactionType != gc.FactionNeutral {
		return consts.InvalidEntity, fmt.Errorf("'%s' は中立NPCではありません", name)
	}
	if entitySpec.Dialog == nil {
		return consts.InvalidEntity, fmt.Errorf("'%s' には会話データがありません", name)
	}

	entitySpec.GridElement = &gc.GridElement{X: consts.Tile(tileX), Y: consts.Tile(tileY)}

	if entitySpec.AIPolicy != nil {
		if err := validateAIPolicy(entitySpec.AIPolicy, entitySpec.SquadMember != nil); err != nil {
			return consts.InvalidEntity, fmt.Errorf("AIPolicy検証エラー (%s): %w", name, err)
		}
		entitySpec.AIMoveFSM = &gc.AIMoveFSM{}
		entitySpec.AIState = &gc.AIState{
			SubState:              gc.AIStateWaiting,
			StartSubStateTurn:     1,
			DurationSubStateTurns: 2 + rand.IntN(3),
			SpawnX:                tileX,
			SpawnY:                tileY,
			PatrolDirX:            initialPatrolDir(),
		}
		entitySpec.AIVision = &gc.AIVision{
			ViewDistance: AIVisionDistance,
		}
	}

	componentList.Entities = append(componentList.Entities, entitySpec)
	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return consts.InvalidEntity, err
	}
	if len(entitiesSlice) == 0 {
		return consts.InvalidEntity, fmt.Errorf("NPCエンティティの生成に失敗しました")
	}

	npcEntity := entitiesSlice[len(entitiesSlice)-1]
	if err := FullRecover(world, npcEntity); err != nil {
		return consts.InvalidEntity, fmt.Errorf("NPCの回復処理エラー: %w", err)
	}

	query.InvalidateSpatialIndex(world)
	return npcEntity, nil
}

// SpawnEnemyOption はSpawnEnemyの振る舞いを変更する関数オプション
type SpawnEnemyOption func(ecs.Entity, w.World)

// WithBoss はボスコンポーネントを付与するオプション
func WithBoss() SpawnEnemyOption {
	return func(entity ecs.Entity, world w.World) {
		entity.AddComponent(world.Components.Boss, &ecs.NullComponent{})
	}
}

// SpawnEnemy はフィールド上に敵キャラクターを生成する
func SpawnEnemy(world w.World, tileX int, tileY int, name string, opts ...SpawnEnemyOption) (ecs.Entity, error) {
	componentList := entities.ComponentList[gc.EntitySpec]{}
	entitySpec, err := raw.NewEnemySpec(world.Resources.RawMaster, name)
	if err != nil {
		return consts.InvalidEntity, fmt.Errorf("%w: %v", ErrEnemyGeneration, err)
	}

	entitySpec.GridElement = &gc.GridElement{X: consts.Tile(tileX), Y: consts.Tile(tileY)}
	entitySpec.AIMoveFSM = &gc.AIMoveFSM{}
	entitySpec.AIState = &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2 + rand.IntN(3),
		SpawnX:                tileX,
		SpawnY:                tileY,
		PatrolDirX:            initialPatrolDir(),
	}
	entitySpec.AIVision = &gc.AIVision{
		ViewDistance: AIVisionDistance,
	}
	entitySpec.Interactable = &gc.Interactable{
		Interactions: []gc.InteractionData{gc.MeleeInteraction{}},
	}
	if entitySpec.AIPolicy == nil {
		return consts.InvalidEntity, fmt.Errorf("敵エンティティにAIPolicyが指定されていません: %s", entitySpec.Name)
	}
	if err := validateAIPolicy(entitySpec.AIPolicy, entitySpec.SquadMember != nil); err != nil {
		return consts.InvalidEntity, fmt.Errorf("AIPolicy検証エラー (%s): %w", entitySpec.Name, err)
	}

	componentList.Entities = append(componentList.Entities, entitySpec)
	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return consts.InvalidEntity, err
	}
	if len(entitiesSlice) == 0 {
		return consts.InvalidEntity, fmt.Errorf("敵エンティティの生成に失敗しました")
	}

	npcEntity := entitiesSlice[len(entitiesSlice)-1]
	if err := FullRecover(world, npcEntity); err != nil {
		return consts.InvalidEntity, fmt.Errorf("敵の回復処理エラー: %w", err)
	}

	if npcEntity.HasComponent(world.Components.TurnBased) {
		actionPoints := world.Components.TurnBased.Get(npcEntity).(*gc.TurnBased)
		maxAP, err := query.CalculateMaxActionPoints(world, npcEntity)
		if err != nil {
			return consts.InvalidEntity, fmt.Errorf("AP計算エラー: %w", err)
		}
		actionPoints.AP.Current = maxAP
		actionPoints.AP.Max = maxAP
	}

	for _, opt := range opts {
		opt(npcEntity, world)
	}

	query.InvalidateSpatialIndex(world)
	return npcEntity, nil
}

// SpawnSquadMember は隊員エンティティを生成する。
// リーダーの隣接空きタイルに配置され、ポリシーに基づいて自律行動する
func SpawnSquadMember(world w.World, leader ecs.Entity, name string, abilities gc.Abilities, spriteKey string) (ecs.Entity, error) {
	if !leader.HasComponent(world.Components.GridElement) {
		return consts.InvalidEntity, fmt.Errorf("リーダーにGridElementがありません")
	}
	leaderGrid := world.Components.GridElement.Get(leader).(*gc.GridElement)

	// リーダーの隣接空きタイルを探す
	spawnX, spawnY, err := findAdjacentEmptyTile(world, int(leaderGrid.X), int(leaderGrid.Y), nil)
	if err != nil {
		return consts.InvalidEntity, fmt.Errorf("隊員のスポーン位置が見つかりません: %w", err)
	}

	skills := gc.NewSkills()
	charMods := gc.RecalculateCharModifiers(skills, &abilities, nil)

	entitySpec := gc.EntitySpec{
		Name:           &gc.Name{Name: name},
		Abilities:      &abilities,
		HP:             &gc.HP{},
		TurnBased:      &gc.TurnBased{AP: gc.IntPool{Current: 100, Max: 100}},
		Skills:         skills,
		CharModifiers:  charMods,
		WeightCapacity: &gc.WeightCapacity{},
		HealthStatus:   &gc.HealthStatus{},
		FactionType:    &gc.FactionAlly,
		AIPolicy: &gc.AIPolicy{
			Planner:       gc.PlannerSquad,
			CombatDefault: gc.CombatAttack,
			CombatCurrent: gc.CombatAttack,
			Movement:      gc.MovementEscort,
			ItemPickup:    gc.PolicyPickup,
			ItemHandling:  gc.PolicyDistribute,
		},
		AIState: &gc.AIState{
			SubState:              gc.AIStateWaiting,
			DurationSubStateTurns: 2,
		},
		GridElement: &gc.GridElement{X: consts.Tile(spawnX), Y: consts.Tile(spawnY)},
		SpriteRender: &gc.SpriteRender{
			SpriteSheetName: "field",
			SpriteKey:       spriteKey,
			Depth:           gc.DepthNumPlayer,
		},
		AIMoveFSM:    &gc.AIMoveFSM{},
		CommandTable: &gc.CommandTable{Name: "素手"},
		AIVision: &gc.AIVision{
			ViewDistance: AIVisionDistance,
		},
		SquadMember: &gc.SquadMember{},
	}

	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, entitySpec)
	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return consts.InvalidEntity, fmt.Errorf("%w: %v", ErrMemberGeneration, err)
	}
	if len(entitiesSlice) != 1 {
		return consts.InvalidEntity, fmt.Errorf("隊員エンティティの生成に失敗しました: エンティティ数=%d", len(entitiesSlice))
	}

	memberEntity := entitiesSlice[0]
	if err := FullRecover(world, memberEntity); err != nil {
		return consts.InvalidEntity, fmt.Errorf("隊員の回復処理エラー: %w", err)
	}

	query.InvalidateSpatialIndex(world)
	return memberEntity, nil
}

// SpawnDefaultSquadMember はゲーム開始時のデフォルト隊員を生成する
func SpawnDefaultSquadMember(world w.World, leader ecs.Entity) (ecs.Entity, error) {
	abilities := gc.Abilities{
		Vitality:  gc.Ability{Base: 8},
		Strength:  gc.Ability{Base: 7},
		Sensation: gc.Ability{Base: 6},
		Dexterity: gc.Ability{Base: 6},
		Agility:   gc.Ability{Base: 7},
		Defense:   gc.Ability{Base: 5},
	}
	return SpawnSquadMember(world, leader, "Jim", abilities, "general")
}

// SpawnBackpackItem はバックパック内にアイテムを生成する
func SpawnBackpackItem(world w.World, name string, count int) (ecs.Entity, error) {
	item, err := spawnItemBase(world, name, count)
	if err != nil {
		return consts.InvalidEntity, err
	}

	var playerEntity ecs.Entity
	var found bool
	world.Manager.Join(world.Components.Player).Visit(ecs.Visit(func(e ecs.Entity) {
		playerEntity = e
		found = true
	}))
	if !found {
		item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{})
		return item, nil
	}
	if err := MoveToBackpack(world, item, playerEntity); err != nil {
		return item, fmt.Errorf("バックパックへの移動に失敗: %w", err)
	}

	return item, nil
}

// spawnItemBase はLocationなしでアイテムエンティティを生成する内部関数
func spawnItemBase(world w.World, name string, count int) (ecs.Entity, error) {
	if count <= 0 {
		return consts.InvalidEntity, fmt.Errorf("count must be positive: %d", count)
	}

	{
		itemDef, err := raw.FindItem(world.Resources.RawMaster, name)
		if err != nil {
			return consts.InvalidEntity, fmt.Errorf("item not found: %s", name)
		}
		isStackable := itemDef.Stackable != nil && *itemDef.Stackable

		if !isStackable && count > 1 {
			return consts.InvalidEntity, fmt.Errorf("item %s is not stackable, count must be 1 (got %d)", name, count)
		}
	}

	componentList := entities.ComponentList[gc.EntitySpec]{}
	entitySpec, err := raw.NewItemSpec(world.Resources.RawMaster, name)
	if err != nil {
		return consts.InvalidEntity, fmt.Errorf("%w: %v", ErrItemGeneration, err)
	}
	if entitySpec.Stackable != nil {
		entitySpec.Stackable.Count = count
	}
	componentList.Entities = append(componentList.Entities, entitySpec)
	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return consts.InvalidEntity, err
	}
	if len(entitiesSlice) == 0 {
		return consts.InvalidEntity, fmt.Errorf("アイテムエンティティの生成に失敗しました")
	}

	return entitiesSlice[len(entitiesSlice)-1], nil
}

// FullRecover はエンティティのHP/APを全回復する
func FullRecover(world w.World, entity ecs.Entity) error {
	if err := setMaxStats(world, entity); err != nil {
		return fmt.Errorf("最大HP設定エラー: %w", err)
	}

	hpComponent := world.Components.HP.Get(entity)
	if hpComponent == nil {
		return fmt.Errorf("HPコンポーネントがありません")
	}
	hp := hpComponent.(*gc.HP)
	hp.Current = hp.Max

	if entity.HasComponent(world.Components.TurnBased) {
		maxAP, err := query.CalculateMaxActionPoints(world, entity)
		if err != nil {
			return fmt.Errorf("AP計算エラー: %w", err)
		}
		turnBased := world.Components.TurnBased.Get(entity).(*gc.TurnBased)
		turnBased.AP.Current = maxAP
		turnBased.AP.Max = maxAP
	}

	return nil
}

// setMaxStats はエンティティの最大HPを設定する
func setMaxStats(world w.World, entity ecs.Entity) error {
	if !entity.HasComponent(world.Components.HP) || !entity.HasComponent(world.Components.Abilities) {
		return fmt.Errorf("entity %v does not have required components (HP or Abilities)", entity)
	}

	hp := world.Components.HP.Get(entity).(*gc.HP)
	abils := world.Components.Abilities.Get(entity).(*gc.Abilities)

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

// SpawnStorageItem は収納内にアイテムを生成する
func SpawnStorageItem(world w.World, itemName string, count int, storage ecs.Entity) (ecs.Entity, error) {
	item, err := spawnItemBase(world, itemName, count)
	if err != nil {
		return consts.InvalidEntity, err
	}

	if err := MoveToStorage(world, item, storage); err != nil {
		return item, fmt.Errorf("収納への移動に失敗: %w", err)
	}

	return item, nil
}

// SpawnFieldItem はフィールド上にアイテムを生成する。countで個数を指定する
func SpawnFieldItem(world w.World, itemName string, x consts.Tile, y consts.Tile, count int) (ecs.Entity, error) {
	item, err := spawnItemBase(world, itemName, count)
	if err != nil {
		return consts.InvalidEntity, err
	}

	MoveToField(world, item, nil)
	item.AddComponent(world.Components.GridElement, &gc.GridElement{X: x, Y: y})

	return item, nil
}

// SpawnVisualEffect はエンティティの位置にエフェクト専用エンティティを生成する
func SpawnVisualEffect(target ecs.Entity, effect gc.VisualEffect, world w.World) {
	if !target.HasComponent(world.Components.GridElement) {
		return
	}

	gridElement := world.Components.GridElement.Get(target).(*gc.GridElement)

	effectEntity := world.Manager.NewEntity()
	effectEntity.AddComponent(world.Components.GridElement, &gc.GridElement{
		X: gridElement.X,
		Y: gridElement.Y,
	})
	effectEntity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{effect},
	})
}

// validateAIPolicy はAIPolicyのPlannerとMovementの組み合わせが有効かを検証する
func validateAIPolicy(policy *gc.AIPolicy, hasSquadMember bool) error {
	if policy.Planner == gc.PlannerSquad && !hasSquadMember {
		return fmt.Errorf("PlannerSquad には SquadMember コンポーネントが必要です")
	}
	if policy.Planner != gc.PlannerSquad && hasSquadMember {
		return fmt.Errorf("SquadMember には PlannerSquad が必要です")
	}

	switch policy.Planner {
	case gc.PlannerRoaming:
		switch policy.Movement {
		case gc.MovementRandom, gc.MovementPatrol, gc.MovementWallHug,
			gc.MovementStationary, gc.MovementWander, gc.MovementTerritorial,
			gc.MovementSwarm:
		case gc.MovementEscort, gc.MovementVanguard, gc.MovementRetreat:
			return fmt.Errorf("PlannerRoaming に隊員用の MovementPolicy %q は使用できません", policy.Movement)
		default:
			return fmt.Errorf("未知の MovementPolicy %q です", policy.Movement)
		}
	case gc.PlannerSquad:
		switch policy.Movement {
		case gc.MovementEscort, gc.MovementVanguard, gc.MovementRetreat,
			gc.MovementPatrol, gc.MovementStationary:
		case gc.MovementRandom, gc.MovementWallHug, gc.MovementWander,
			gc.MovementTerritorial, gc.MovementSwarm:
			return fmt.Errorf("PlannerSquad に敵用の MovementPolicy %q は使用できません", policy.Movement)
		default:
			return fmt.Errorf("未知の MovementPolicy %q です", policy.Movement)
		}
	default:
		return fmt.Errorf("未知の PlannerType %d です", policy.Planner)
	}

	return nil
}
