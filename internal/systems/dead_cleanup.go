package systems

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
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
	deadQuery := query.ActiveFilter1[gc.Dead](world).Without(ecs.C[gc.Player]()).Query()
	for deadQuery.Next() {
		entity := deadQuery.Entity()
		toDelete = append(toDelete, entity)
	}

	// 死亡エンティティのアクティビティをキャンセルする
	for _, entity := range toDelete {
		if query.GetActivity(world, entity) != nil {
			activity.CancelActivity(entity, "death", world)
		}
	}

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

		// テーブル名はロード時に参照検証済みで、ここで失敗するのは整合性バグ
		dropTable, err := raw.GetDropTable(rawMaster, dropTableComp.Name)
		if err != nil {
			return fmt.Errorf("failed to get drop table: %w", err)
		}

		// アイテム選択
		materialName, err := raw.SelectDropByWeight(dropTable, world.Resources.Config.RNG)
		if err != nil {
			return err
		}
		// ドロップしない
		if materialName == "" {
			continue
		}

		// フィールドにアイテムをスポーン。素材名はロード時に参照検証済み
		_, err = lifecycle.SpawnFieldItem(world, materialName, gridElement.X, gridElement.Y, 1)
		if err != nil {
			return fmt.Errorf("failed to spawn drop item: %w", err)
		}
		logger.Debug("spawn drop item", "material", materialName, "x", gridElement.X, "y", gridElement.Y)
	}

	if err := dropPropYields(world, toDelete); err != nil {
		return err
	}

	// 撃破統計: 除去する敵を run 統計へ加算する
	if stats := query.GetRunStats(world); stats != nil {
		for _, entity := range toDelete {
			if world.Components.FactionEnemy.Has(entity) {
				stats.EnemiesKilled++
			}
		}
	}

	// 死亡エンティティのバックパック内アイテムをフィールドにドロップする。
	// クエリ走査中の構造変更を避けるため先に集め、フィールドへの一括移動は移動口に委ねる
	for _, entity := range toDelete {
		if !world.Components.GridElement.Has(entity) {
			continue
		}
		coord := world.Components.GridElement.Get(entity).Coord
		// 収納 prop を壊した時に中身を足元へ出し、失わせない
		if world.Components.WeightCapacity.Has(entity) {
			lifecycle.SpillStorageItems(world, entity, coord.X, coord.Y)
		}
		owner := entity
		var items []ecs.Entity
		backpackQuery := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
		for backpackQuery.Next() {
			item := backpackQuery.Entity()
			if world.Components.LocationInBackpack.Get(item).Owner == owner {
				items = append(items, item)
			}
		}
		lifecycle.MoveMembersToField(world, items, coord, owner)
	}

	// エンティティを削除する
	for _, entity := range toDelete {
		// スプライトフェードアウトエフェクトを生成
		if world.Components.SpriteRender.Has(entity) && world.Components.GridElement.Has(entity) {
			spriteRender := world.Components.SpriteRender.Get(entity)
			gridElement := world.Components.GridElement.Get(entity)

			effect := gc.NewSpriteFadeoutEffect(spriteRender.SpriteSheetName, spriteRender.SpriteKey)
			effectEntity := world.ECS.NewEntity()
			world.Components.GridElement.Add(effectEntity, &gc.GridElement{Coord: gridElement.Coord})
			world.Components.VisualEffects.Add(effectEntity, &gc.VisualEffects{
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

// dropPropYields は壊れた prop の産出を足元へ落とす。分解定義を持つ prop は確定枠を低確率で回収でき、
// 工具がない序盤でも素材が少しは手に入る。伐採定義を持つ prop は倒れると産出を落とす
func dropPropYields(world w.World, entities []ecs.Entity) error {
	rawMaster := world.Resources.RawMaster
	rng := world.Resources.Config.RNG
	for _, entity := range entities {
		if !world.Components.Prop.Has(entity) || !world.Components.GridElement.Has(entity) {
			continue
		}
		id := query.GetEntityID(entity, world)
		var stacks []lifecycle.YieldStack
		// 産出名・個数表記はロード時に検証済みで、ここで失敗するのは整合性バグ。
		// 握りつぶすと産出が静かに消えるため loud に返す
		if def, ok := raw.FindDisassembly(rawMaster, id); ok {
			salvage, err := lifecycle.RollDisassemblyYields(rng, def, 0, 0, false)
			if err != nil {
				return fmt.Errorf("failed to draw salvage item: %w", err)
			}
			stacks = append(stacks, salvage...)
		}
		if def, ok := raw.FindFelling(rawMaster, id); ok {
			felled, err := lifecycle.RollFellingYields(rng, def)
			if err != nil {
				return fmt.Errorf("failed to draw felling yield: %w", err)
			}
			stacks = append(stacks, felled...)
		}
		grid := world.Components.GridElement.Get(entity)
		if err := lifecycle.SpawnDisassemblyYields(world, stacks, grid.X, grid.Y); err != nil {
			return fmt.Errorf("failed to spawn prop yield: %w", err)
		}
	}
	return nil
}
