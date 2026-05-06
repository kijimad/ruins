package systems

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
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
	world.Manager.Join(
		world.Components.Dead,
		world.Components.Player.Not(),
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		toDelete = append(toDelete, entity)
	}))

	// 死亡エンティティのアクティビティをキャンセルする
	for _, entity := range toDelete {
		if worldhelper.GetActivity(world, entity) != nil {
			activity.CancelActivity(entity, "死亡", world)
		}
	}

	// ドロップアイテム生成
	rawMaster := world.Resources.RawMaster

	for _, entity := range toDelete {
		// ドロップに必要なコンポーネントをチェック
		if !entity.HasComponent(world.Components.DropTable) {
			continue
		}
		if !entity.HasComponent(world.Components.GridElement) {
			continue
		}

		dropTableComp := world.Components.DropTable.Get(entity).(*gc.DropTable)
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		dropTable, err := rawMaster.GetDropTable(dropTableComp.Name)
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
		_, err = worldhelper.SpawnFieldItem(world, materialName, gridElement.X, gridElement.Y)
		if err != nil {
			logger.Debug("ドロップアイテム生成失敗", "error", err, "material", materialName)
		} else {
			logger.Debug("ドロップアイテム生成", "material", materialName, "x", gridElement.X, "y", gridElement.Y)
		}
	}

	// ボス撃破時の処理: 扉アンロック + クリアフラグ
	for _, entity := range toDelete {
		if entity.HasComponent(world.Components.Boss) {
			// 全扉をアンロックして開く
			if worldhelper.UnlockAllDoors(world) > 0 {
				gamelog.New(gamelog.FieldLog).
					Append("どこかで扉が開いたようだ。").
					Log()
			}

			// DoorLockTriggerエンティティを削除する。ボス撃破後はトリガー不要
			worldhelper.DeleteDoorLockTriggers(world)

			// ダンジョンクリアフラグを立てる
			dungeonName := world.Resources.Dungeon.DefinitionName
			world.Resources.GameProgress.MarkDungeonCleared(dungeonName)

			logger.Debug("ボス撃破: 扉アンロック+クリアフラグ", "dungeon", dungeonName)
		}
	}

	// エンティティを削除する
	for _, entity := range toDelete {
		// スプライトフェードアウトエフェクトを生成
		if entity.HasComponent(world.Components.SpriteRender) && entity.HasComponent(world.Components.GridElement) {
			spriteRender := world.Components.SpriteRender.Get(entity).(*gc.SpriteRender)
			gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

			effect := gc.NewSpriteFadeoutEffect(spriteRender.SpriteSheetName, spriteRender.SpriteKey)
			effectEntity := world.Manager.NewEntity()
			effectEntity.AddComponent(world.Components.GridElement, &gc.GridElement{
				X: gridElement.X,
				Y: gridElement.Y,
			})
			effectEntity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
				Effects: []gc.VisualEffect{effect},
			})
		}

		world.Manager.DeleteEntity(entity)
	}

	if len(toDelete) > 0 {
		logger.Debug("Dead cleanup completed", "deleted_count", len(toDelete))
	}

	return nil
}
