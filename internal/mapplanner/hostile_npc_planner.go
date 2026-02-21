// Package mapplanner の敵NPC配置プランナー
package mapplanner

import (
	"log"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
)

// 敵NPC配置用の定数
const (
	// 敵NPC生成関連
	baseHostileNPCCount    = 10  // 敵NPC生成の基本数
	randomHostileNPCCount  = 6   // 敵NPC生成のランダム追加数（0-5の範囲）
	maxHostileNPCFailCount = 200 // 敵NPC生成の最大失敗回数
)

// NPCSpec はNPC配置仕様を表す
type NPCSpec struct {
	Coord
	Name string // NPCタイプ
}

// HostileNPCPlanner は敵NPC配置を担当するプランナー
type HostileNPCPlanner struct {
	world       w.World
	plannerType PlannerType
}

// NewHostileNPCPlanner は敵NPCプランナーを作成する
func NewHostileNPCPlanner(world w.World, plannerType PlannerType) *HostileNPCPlanner {
	return &HostileNPCPlanner{
		world:       world,
		plannerType: plannerType,
	}
}

// PlanMeta は敵NPC配置情報をMetaPlanに追加する
func (n *HostileNPCPlanner) PlanMeta(planData *MetaPlan) error {
	// NPCsフィールドが存在しない場合は初期化
	if planData.NPCs == nil {
		planData.NPCs = []NPCSpec{}
	}

	// 敵NPCの配置
	if len(n.plannerType.EnemyEntries) == 0 {
		return nil // エントリがない場合は何もしない
	}

	failCount := 0
	total := baseHostileNPCCount + planData.RNG.IntN(randomHostileNPCCount)
	successCount := 0

	for successCount < total && failCount <= maxHostileNPCFailCount {
		tx := gc.Tile(planData.RNG.IntN(int(planData.Level.TileWidth)))
		ty := gc.Tile(planData.RNG.IntN(int(planData.Level.TileHeight)))

		if !planData.IsSpawnableTile(n.world, tx, ty) {
			failCount++
			continue
		}

		// エントリから重み付き抽選で敵を選択
		enemyName, err := selectByWeight(n.plannerType.EnemyEntries, planData.RNG)
		if err != nil {
			return err
		}
		if enemyName == "" {
			failCount++
			continue
		}

		planData.NPCs = append(planData.NPCs, NPCSpec{
			Coord: Coord{X: int(tx), Y: int(ty)},
			Name:  enemyName,
		})

		successCount++
		failCount = 0
	}

	if failCount > maxHostileNPCFailCount {
		// エラーは記録するが、エラーを返さずに部分的な配置で続行
		log.Printf("HostileNPCPlanner: 敵NPC配置の試行回数が上限に達しました。配置数: %d/%d", successCount, total)
	}
	return nil
}
