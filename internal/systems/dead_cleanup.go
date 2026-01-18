package systems

import (
	"math/rand/v2"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
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

	// ドロップアイテム生成
	rawMaster := world.Resources.RawMaster.(*raw.Master)

	// 乱数生成器を取得（テスト用に注入可能）
	rng := world.Resources.RNG
	if rng == nil {
		// 本番環境では現在時刻をシードにする
		rng = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))
	}

	for _, entity := range toDelete {
		// ドロップテーブルコンポーネントをチェック
		if !entity.HasComponent(world.Components.DropTable) {
			continue
		}

		dropTableComp := world.Components.DropTable.Get(entity).(*gc.DropTable)
		dropTable, err := rawMaster.GetDropTable(dropTableComp.Name)
		if err != nil {
			logger.Debug("ドロップテーブル取得失敗", "error", err, "table_name", dropTableComp.Name)
			continue
		}

		// アイテム選択
		materialName := dropTable.SelectByWeight(rng)
		if materialName == "" {
			continue
		}

		// エンティティの位置を取得（GridElementはタイル座標を保持）
		if !entity.HasComponent(world.Components.GridElement) {
			continue
		}
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// フィールドにアイテムをスポーン
		_, err = worldhelper.SpawnFieldItem(world, materialName, gridElement.X, gridElement.Y)
		if err != nil {
			logger.Debug("ドロップアイテム生成失敗", "error", err, "material", materialName)
		} else {
			logger.Debug("ドロップアイテム生成", "material", materialName, "x", gridElement.X, "y", gridElement.Y)
		}
	}

	// エンティティを削除
	for _, entity := range toDelete {
		world.Manager.DeleteEntity(entity)
	}

	if len(toDelete) > 0 {
		logger.Debug("Dead cleanup completed", "deleted_count", len(toDelete))
	}

	return nil
}
