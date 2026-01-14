package worldhelper

import (
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/lib/config"
	"github.com/kijimaD/ruins/lib/engine/entities"
	"github.com/kijimaD/ruins/lib/raw"
	"github.com/kijimaD/ruins/lib/turns"
	ecs "github.com/x-hgg-x/goecs/v2"

	gc "github.com/kijimaD/ruins/lib/components"
	w "github.com/kijimaD/ruins/lib/world"
)

// 定数定義
const (
	// カメラスケール
	cameraInitialScale = 0.1 // カメラの初期スケール（ズームアウト）
	cameraNormalScale  = 1.0 // カメラの通常スケール

	// AI設定
	aiVisionDistance = 160.0 // AIの視界距離（ピクセル）

	// ステータス計算係数
	hpBaseValue         = 30   // HP計算の基本値
	hpVitalityMultiply  = 8    // HP計算の体力係数
	spVitalityMultiply  = 2    // SP計算の体力係数
	epBaseValue         = 50   // EP計算の基本値
	epSensationMultiply = 3    // EP計算の感覚係数
	hpLevelGrowthRate   = 0.03 // HPのレベル成長率
	spLevelGrowthRate   = 0.02 // SPのレベル成長率
)

// エラー定義
var (
	ErrItemGeneration   = errors.New("アイテムの生成に失敗しました")
	ErrMemberGeneration = errors.New("メンバーの生成に失敗しました")
	ErrEnemyGeneration  = errors.New("敵の生成に失敗しました")
	ErrEffectGeneration = errors.New("エフェクトの生成に失敗しました")
)

// raw のNewSpec系と、worldhelperのSpawn系の使い分け
//
// raw: 共通なところに関してコンポーネント群を付与する
// worldhelper: 頻繁に変わる部分に関して引数を受け取れるようにする。エンティティを発行するところまでやる

// ================
// Field
// ================

// SpawnTile はタイルを生成する
// autoTileIndexが指定された場合、spriteKeyを動的に生成する（例: "wall_5"）
func SpawnTile(world w.World, tileName string, x gc.Tile, y gc.Tile, autoTileIndex *int) (ecs.Entity, error) {
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	entitySpec, err := rawMaster.NewTileSpec(tileName, x, y, autoTileIndex)
	if err != nil {
		return ecs.Entity(0), err
	}

	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, entitySpec)

	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return ecs.Entity(0), err
	}
	if len(entitiesSlice) == 0 {
		return ecs.Entity(0), fmt.Errorf("エンティティの生成に失敗しました")
	}
	return entitiesSlice[0], nil
}

// ================
// Characters
// ================

// SpawnPlayer はプレイヤーキャラクターを生成する
func SpawnPlayer(world w.World, tileX int, tileY int, name string) (ecs.Entity, error) {
	componentList := entities.ComponentList[gc.EntitySpec]{}
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	entitySpec, err := rawMaster.NewPlayerSpec(name)
	if err != nil {
		return ecs.Entity(0), fmt.Errorf("%w: %v", ErrMemberGeneration, err)
	}
	entitySpec.GridElement = &gc.GridElement{X: gc.Tile(tileX), Y: gc.Tile(tileY)}
	// カメラ
	{
		// config設定を確認
		cfg := config.Get()
		var scale, scaleTo float64
		if cfg.DisableAnimation {
			// アニメーション無効時は初期スケールを通常値に設定
			scale = cameraNormalScale
			scaleTo = cameraNormalScale
		} else {
			// アニメーション有効時はズームアウトアニメーション
			scale = cameraInitialScale
			scaleTo = cameraNormalScale
		}
		entitySpec.Camera = &gc.Camera{Scale: scale, ScaleTo: scaleTo}
	}
	entitySpec.Wallet = &gc.Wallet{Currency: 1000}
	componentList.Entities = append(componentList.Entities, entitySpec)
	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return ecs.Entity(0), err
	}
	if len(entitiesSlice) == 0 {
		return ecs.Entity(0), fmt.Errorf("プレイヤーエンティティの生成に失敗しました")
	}
	fullRecover(world, entitiesSlice[len(entitiesSlice)-1])

	return entitiesSlice[len(entitiesSlice)-1], nil
}

// SpawnNeutralNPC はフィールド上に中立NPCを生成する（会話可能なNPC用）
func SpawnNeutralNPC(world w.World, tileX int, tileY int, name string) (ecs.Entity, error) {
	componentList := entities.ComponentList[gc.EntitySpec]{}
	rawMaster := world.Resources.RawMaster.(*raw.Master)

	// NewMemberSpecでEntitySpecを生成
	entitySpec, err := rawMaster.NewMemberSpec(name)
	if err != nil {
		return ecs.Entity(0), fmt.Errorf("中立NPC生成エラー: %w", err)
	}

	// 中立派閥とDialog設定を確認
	if entitySpec.FactionType == nil || *entitySpec.FactionType != gc.FactionNeutral {
		return ecs.Entity(0), fmt.Errorf("'%s' は中立NPCではありません", name)
	}
	if entitySpec.Dialog == nil {
		return ecs.Entity(0), fmt.Errorf("'%s' には会話データがありません", name)
	}

	// フィールド用のコンポーネントを設定
	entitySpec.GridElement = &gc.GridElement{X: gc.Tile(tileX), Y: gc.Tile(tileY)}
	entitySpec.BlockPass = &gc.BlockPass{} // NPCは通行不可

	// 中立NPCにはAIを付けない（動かない）

	componentList.Entities = append(componentList.Entities, entitySpec)
	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return ecs.Entity(0), err
	}
	if len(entitiesSlice) == 0 {
		return ecs.Entity(0), fmt.Errorf("NPCエンティティの生成に失敗しました")
	}

	// 全回復
	npcEntity := entitiesSlice[len(entitiesSlice)-1]
	fullRecover(world, npcEntity)

	return npcEntity, nil
}

// SpawnEnemy はフィールド上に敵キャラクターを生成する
func SpawnEnemy(world w.World, tileX int, tileY int, name string) (ecs.Entity, error) {
	componentList := entities.ComponentList[gc.EntitySpec]{}
	rawMaster := world.Resources.RawMaster.(*raw.Master)

	// raw.Masterから敵データを取得
	entitySpec, err := rawMaster.NewEnemySpec(name)
	if err != nil {
		return ecs.Entity(0), fmt.Errorf("%w: %v", ErrEnemyGeneration, err)
	}

	// フィールド用のコンポーネントを設定
	entitySpec.GridElement = &gc.GridElement{X: gc.Tile(tileX), Y: gc.Tile(tileY)}
	entitySpec.BlockPass = &gc.BlockPass{}
	entitySpec.AIMoveFSM = &gc.AIMoveFSM{}
	entitySpec.AIRoaming = &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     1,                // 初期ターン
		DurationSubStateTurns: 2 + rand.IntN(3), // 2-4ターン待機
	}
	entitySpec.AIVision = &gc.AIVision{
		ViewDistance: gc.Pixel(aiVisionDistance),
	}
	entitySpec.Interactable = &gc.Interactable{
		Data: gc.MeleeInteraction{},
	}

	componentList.Entities = append(componentList.Entities, entitySpec)
	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return ecs.Entity(0), err
	}
	if len(entitiesSlice) == 0 {
		return ecs.Entity(0), fmt.Errorf("敵エンティティの生成に失敗しました")
	}

	// 全回復
	npcEntity := entitiesSlice[len(entitiesSlice)-1]
	fullRecover(world, npcEntity)

	// ActionPointsを初期化
	if world.Resources.TurnManager != nil {
		if turnManager, ok := world.Resources.TurnManager.(*turns.TurnManager); ok {
			if npcEntity.HasComponent(world.Components.TurnBased) {
				actionPoints := world.Components.TurnBased.Get(npcEntity).(*gc.TurnBased)
				maxAP, err := turnManager.CalculateMaxActionPoints(world, npcEntity)
				if err != nil {
					return ecs.Entity(0), fmt.Errorf("AP計算エラー: %w", err)
				}
				actionPoints.AP.Current = maxAP
				actionPoints.AP.Max = maxAP
			}
		}
	}

	return npcEntity, nil
}

// ================
// Items
// ================

// SpawnItem はアイテムを生成する（Stackableコンポーネントは付与しない）
func SpawnItem(world w.World, name string, locationType gc.ItemLocationType) (ecs.Entity, error) {
	componentList := entities.ComponentList[gc.EntitySpec]{}
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	entitySpec, err := rawMaster.NewItemSpec(name)
	if err != nil {
		return ecs.Entity(0), fmt.Errorf("%w: %v", ErrItemGeneration, err)
	}
	entitySpec.ItemLocationType = &locationType
	componentList.Entities = append(componentList.Entities, entitySpec)
	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return ecs.Entity(0), err
	}
	if len(entitiesSlice) == 0 {
		return ecs.Entity(0), fmt.Errorf("アイテムエンティティの生成に失敗しました")
	}

	return entitiesSlice[len(entitiesSlice)-1], nil
}

// SpawnStackable はStackableアイテムを生成する
// countは1以上である必要がある（0以下の場合はエラー）
func SpawnStackable(world w.World, name string, count int, location gc.ItemLocationType) (ecs.Entity, error) {
	if count <= 0 {
		return 0, fmt.Errorf("count must be positive: %d", count)
	}

	rawMaster := world.Resources.RawMaster.(*raw.Master)

	itemIdx, ok := rawMaster.ItemIndex[name]
	if !ok {
		return 0, fmt.Errorf("item not found: %s", name)
	}
	itemDef := rawMaster.Raws.Items[itemIdx]
	if itemDef.Stackable == nil || !*itemDef.Stackable {
		return 0, fmt.Errorf("item %s is not stackable", name)
	}

	componentList := entities.ComponentList[gc.EntitySpec]{}
	entitySpec, err := rawMaster.NewItemSpec(name)
	if err != nil {
		return 0, fmt.Errorf("failed to spawn stackable item: %w", err)
	}
	entitySpec.ItemLocationType = &location
	entitySpec.Item.Count = count

	componentList.Entities = append(componentList.Entities, entitySpec)
	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return ecs.Entity(0), err
	}
	if len(entitiesSlice) == 0 {
		return ecs.Entity(0), fmt.Errorf("Stackableアイテムエンティティの生成に失敗しました")
	}

	return entitiesSlice[len(entitiesSlice)-1], nil
}

// 完全回復させる
func fullRecover(world w.World, entity ecs.Entity) {
	// 新しく生成されたエンティティの最大HP/SPを設定
	_ = setMaxHPSP(world, entity) // エラーが発生した場合もリカバリーは続行する

	// Poolsコンポーネントを取得
	poolsComponent := world.Components.Pools.Get(entity)
	if poolsComponent == nil {
		return // Poolsがない場合は何もしない
	}

	pools := poolsComponent.(*gc.Pools)

	// HP全回復
	pools.HP.Current = pools.HP.Max

	// SP全回復
	pools.SP.Current = pools.SP.Max

	// EP全回復
	pools.EP.Current = pools.EP.Max

	// ActionPointsコンポーネントがある場合は最大APに設定
	if entity.HasComponent(world.Components.TurnBased) {
		if world.Resources.TurnManager != nil {
			if turnManager, ok := world.Resources.TurnManager.(*turns.TurnManager); ok {
				actionPoints := world.Components.TurnBased.Get(entity).(*gc.TurnBased)
				maxAP, err := turnManager.CalculateMaxActionPoints(world, entity)
				if err != nil {
					// FullRecoverはエラーを返さないので、ログに記録してデフォルト値を設定
					fmt.Printf("AP計算エラー: %v\n", err)
					maxAP = 100 // デフォルト値
				}
				actionPoints.AP.Current = maxAP
				actionPoints.AP.Max = maxAP
			}
		}
	}
}

// 指定したエンティティの最大HP/SPを設定する
func setMaxHPSP(world w.World, entity ecs.Entity) error {

	if !entity.HasComponent(world.Components.Pools) || !entity.HasComponent(world.Components.Attributes) {
		return fmt.Errorf("entity %v does not have required components (Pools or Attributes)", entity)
	}

	pools := world.Components.Pools.Get(entity).(*gc.Pools)
	attrs := world.Components.Attributes.Get(entity).(*gc.Attributes)

	// Totalが設定されていない場合はBaseから初期化
	if attrs.Vitality.Total == 0 {
		attrs.Vitality.Total = attrs.Vitality.Base
	}
	if attrs.Strength.Total == 0 {
		attrs.Strength.Total = attrs.Strength.Base
	}
	if attrs.Sensation.Total == 0 {
		attrs.Sensation.Total = attrs.Sensation.Base
	}
	if attrs.Dexterity.Total == 0 {
		attrs.Dexterity.Total = attrs.Dexterity.Base
	}
	if attrs.Agility.Total == 0 {
		attrs.Agility.Total = attrs.Agility.Base
	}
	if attrs.Defense.Total == 0 {
		attrs.Defense.Total = attrs.Defense.Base
	}

	// 最大HP計算: base+(体力*multiplyV+力+感覚)
	pools.HP.Max = int(hpBaseValue) + attrs.Vitality.Total*hpVitalityMultiply + attrs.Strength.Total + attrs.Sensation.Total
	pools.HP.Current = pools.HP.Max

	// 最大SP計算: (体力*multiplyV+器用さ+素早さ)
	pools.SP.Max = attrs.Vitality.Total*spVitalityMultiply + attrs.Dexterity.Total + attrs.Agility.Total
	pools.SP.Current = pools.SP.Max

	// 最大EP計算: base+(感覚*multiplyS+器用さ)
	pools.EP.Max = int(epBaseValue) + attrs.Sensation.Total*epSensationMultiply + attrs.Dexterity.Total
	pools.EP.Current = pools.EP.Max

	return nil
}

// SpawnFieldItem はフィールド上にアイテムを生成する
func SpawnFieldItem(world w.World, itemName string, x gc.Tile, y gc.Tile) (ecs.Entity, error) {
	// TOMLの定義を取得してStackable対応かどうかを確認
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	itemIdx, ok := rawMaster.ItemIndex[itemName]
	if !ok {
		return ecs.Entity(0), fmt.Errorf("item not found: %s", itemName)
	}
	itemDef := rawMaster.Raws.Items[itemIdx]
	isStackable := itemDef.Stackable != nil && *itemDef.Stackable

	var item ecs.Entity
	var err error
	if isStackable {
		// Stackable対応アイテムはCount=1で生成
		item, err = SpawnStackable(world, itemName, 1, gc.ItemLocationOnField)
		if err != nil {
			return ecs.Entity(0), err
		}
	} else {
		// 通常アイテムは通常通り生成
		item, err = SpawnItem(world, itemName, gc.ItemLocationOnField)
		if err != nil {
			return ecs.Entity(0), err
		}
	}

	// フィールド表示用のコンポーネントを追加
	item.AddComponent(world.Components.GridElement, &gc.GridElement{X: x, Y: y})

	return item, nil
}

// MovePlayerToPosition は既存のプレイヤーエンティティを指定位置に移動させる
// GridElement、SpriteRender、Cameraコンポーネントがない場合は追加する（ロード時に対応）
func MovePlayerToPosition(world w.World, tileX int, tileY int) error {
	// 既存のプレイヤーエンティティを検索
	var playerEntity ecs.Entity
	var found bool

	world.Manager.Join(world.Components.Player).Visit(ecs.Visit(func(entity ecs.Entity) {
		if !found {
			playerEntity = entity
			found = true
		}
	}))

	if !found {
		return errors.New("プレイヤーエンティティが見つかりません")
	}

	// GridElementがない場合は追加
	if !playerEntity.HasComponent(world.Components.GridElement) {
		playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{})
	}

	// プレイヤーの位置を更新
	gridElement := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)
	gridElement.X = gc.Tile(tileX)
	gridElement.Y = gc.Tile(tileY)

	// SpriteRenderがない場合は追加
	if !playerEntity.HasComponent(world.Components.SpriteRender) {
		// プレイヤー名から正しいスプライト情報を取得
		rawMaster := world.Resources.RawMaster.(*raw.Master)
		nameComp := world.Components.Name.Get(playerEntity).(*gc.Name)
		playerSpec, err := rawMaster.NewPlayerSpec(nameComp.Name)
		if err != nil {
			return fmt.Errorf("プレイヤーのスプライト情報取得に失敗: %w", err)
		}

		playerEntity.AddComponent(world.Components.SpriteRender, playerSpec.SpriteRender)
	}

	// Cameraがない場合は追加（通常スケールで初期化）
	if !playerEntity.HasComponent(world.Components.Camera) {
		playerEntity.AddComponent(world.Components.Camera, &gc.Camera{
			Scale:   cameraNormalScale,
			ScaleTo: cameraNormalScale,
		})
	}

	return nil
}
