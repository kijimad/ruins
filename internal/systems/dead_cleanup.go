package systems

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// DeadCleanupSystem はDeadコンポーネントを持つ敵エンティティを削除する
// 削除前にドロップテーブルがあればアイテムをドロップする
type DeadCleanupSystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys DeadCleanupSystem) String() string {
	return "DeadCleanupSystem"
}

// Update はDeadコンポーネントを持つ敵エンティティを削除する
// w.Updater interfaceを実装
func (sys *DeadCleanupSystem) Update(world w.World) error {
	logger := logger.New(logger.CategorySystem)

	// Deadコンポーネントを持つエンティティを検索
	var toDelete []ecs.Entity
	deadQuery := ecs.NewFilter1[gc.Dead](world.ECS).
		Without(ecs.C[gc.Player]()).Query()
	for deadQuery.Next() {
		entity := deadQuery.Entity()
		toDelete = append(toDelete, entity)
	}

	// 死亡エンティティのアクティビティをキャンセルする
	for _, entity := range toDelete {
		if query.GetActivity(world, entity) != nil {
			activity.CancelActivity(entity, "死亡", world)
		}
	}

	// ドロップアイテム生成
	rawMaster := world.Resources.RawMaster

	for _, entity := range toDelete {
		// ドロップに必要なコンポーネントをチェック
		if !world.Components.DropTable.Has(entity) {
			continue
		}
		if !world.Components.GridElement.Has(entity) {
			continue
		}

		dropTableComp := world.Components.DropTable.Get(entity)
		gridElement := world.Components.GridElement.Get(entity)

		dropTable, err := raw.GetDropTable(rawMaster, dropTableComp.Name)
		if err != nil {
			logger.Debug("ドロップテーブル取得失敗", "error", err, "table_name", dropTableComp.Name)
			continue
		}

		// アイテム選択
		materialName, err := raw.SelectDropByWeight(dropTable, world.Config.RNG)
		if err != nil {
			return err
		}
		// ドロップしない
		if materialName == "" {
			continue
		}

		// フィールドにアイテムをスポーン
		_, err = lifecycle.SpawnFieldItem(world, materialName, gridElement.X, gridElement.Y, 1)
		if err != nil {
			logger.Debug("ドロップアイテム生成失敗", "error", err, "material", materialName)
		} else {
			logger.Debug("ドロップアイテム生成", "material", materialName, "x", gridElement.X, "y", gridElement.Y)
		}
	}

	// ボス撃破時の処理: 扉アンロック + クリアフラグ
	for _, entity := range toDelete {
		if world.Components.Boss.Has(entity) {
			// 全扉をアンロックして開く
			if lifecycle.UnlockAllDoors(world) > 0 {
				gamelog.New(query.GetGameLog(world)).
					Append("どこかで扉が開いたようだ。").
					Log()
			}

			// DoorLockTriggerエンティティを削除する。ボス撃破後はトリガー不要
			lifecycle.DeleteDoorLockTriggers(world)

			// ダンジョンクリアフラグを立てる
			dungeonName := query.GetDungeon(world).DefinitionName
			query.GetGameProgress(world).MarkDungeonCleared(dungeonName)

			logger.Debug("ボス撃破: 扉アンロック+クリアフラグ", "dungeon", dungeonName)
		}
	}

	// 死亡エンティティのバックパック内アイテムをフィールドにドロップする
	for _, entity := range toDelete {
		if !world.Components.GridElement.Has(entity) {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		gridX, gridY := grid.X, grid.Y
		owner := entity
		var items []ecs.Entity
		backpackQuery := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
		for backpackQuery.Next() {
			item := backpackQuery.Entity()
			if world.Components.LocationInBackpack.Get(item).Owner == owner {
				items = append(items, item)
			}
		}
		for _, item := range items {
			world.Components.GridElement.Add(item, &gc.GridElement{X: gridX, Y: gridY})
			lifecycle.MoveToField(world, item, &owner)
		}
	}

	// エンティティを削除する
	for _, entity := range toDelete {
		// スプライトフェードアウトエフェクトを生成
		if world.Components.SpriteRender.Has(entity) && world.Components.GridElement.Has(entity) {
			spriteRender := world.Components.SpriteRender.Get(entity)
			gridElement := world.Components.GridElement.Get(entity)

			effect := gc.NewSpriteFadeoutEffect(spriteRender.SpriteSheetName, spriteRender.SpriteKey)
			effectEntity := world.ECS.NewEntity()
			world.Components.GridElement.Add(effectEntity, &gc.GridElement{
				X: gridElement.X,
				Y: gridElement.Y,
			})
			world.Components.VisualEffect.Add(effectEntity, &gc.VisualEffects{
				Effects: []gc.VisualEffect{effect},
			})
		}

		world.ECS.RemoveEntity(entity)
	}

	if len(toDelete) > 0 {
		logger.Debug("Dead cleanup completed", "deleted_count", len(toDelete))
	}

	return nil
}
