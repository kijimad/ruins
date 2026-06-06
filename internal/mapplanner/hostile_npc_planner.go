// Package mapplanner の敵NPC配置プランナー
package mapplanner

import (
	"log"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// 敵NPC配置用の定数
const (
	// 敵NPC生成関連
	baseHostileNPCCount    = 5   // 敵NPC生成の基本数
	randomHostileNPCCount  = 4   // 敵NPC生成のランダム追加数（0-3の範囲）
	maxHostileNPCFailCount = 200 // 敵NPC生成の最大失敗回数

	// クラスタ関連
	maxRoomAttempts = 50 // 部屋内の座標探索の最大試行回数
	clusterRadius   = 4  // ホットスポットクラスタの半径（タイル数）
)

// NPCSpec はNPC配置仕様を表す
type NPCSpec struct {
	consts.Coord[int]
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
// 部屋がある場合は部屋ベースで同種クラスタ配置する
func (n *HostileNPCPlanner) PlanMeta(planData *MetaPlan) error {
	if planData.NPCs == nil {
		planData.NPCs = []NPCSpec{}
	}

	if len(n.plannerType.EnemyEntries) == 0 {
		return nil
	}

	total := baseHostileNPCCount + planData.RNG.IntN(randomHostileNPCCount)

	// 部屋がある場合は部屋ベースでクラスタ配置
	if len(planData.Rooms) > 0 {
		return n.planWithRoomCluster(planData, total)
	}

	// 部屋がない場合はフォールバック
	return n.planWithRandomPosition(planData, total)
}

// planWithRoomCluster は部屋を選び、同種の敵をクラスタとしてまとめて配置する
// パックサイズはSpawnEntryのPackMin/PackMaxで制御する
func (n *HostileNPCPlanner) planWithRoomCluster(planData *MetaPlan, total int) error {
	placed := 0
	failCount := 0
	// 部屋ごとに割り当てられた敵種を記録する
	roomSpecies := map[int]SpawnEntry{}

	for placed < total && failCount <= maxHostileNPCFailCount {
		room, roomIdx, _ := planData.selectRoom()

		entry, decided := roomSpecies[roomIdx]
		if !decided {
			var err error
			entry, err = selectSpawnEntry(n.plannerType.EnemyEntries, planData.RNG)
			if err != nil {
				return err
			}
			if entry.Name == "" {
				failCount++
				continue
			}
			roomSpecies[roomIdx] = entry
		}

		// パックサイズはデータから取得する
		packSize := entry.PackSize(planData.RNG)
		remaining := total - placed
		if packSize > remaining {
			packSize = remaining
		}

		// パックの最初の1体を部屋内でランダムに配置する
		anchorX, anchorY, err := findPosition(planData, n.world, inRoomSelector(room, maxRoomAttempts))
		if err != nil {
			failCount++
			continue
		}
		planData.NPCs = append(planData.NPCs, NPCSpec{
			Coord: consts.Coord[int]{X: int(anchorX), Y: int(anchorY)},
			Name:  entry.Name,
		})
		placed++
		failCount = 0

		// 残りのパックメンバーはアンカーの近くに配置する
		for i := 1; i < packSize; i++ {
			tx, ty, nearErr := findPosition(planData, n.world, nearSelector(anchorX, anchorY, clusterRadius, room, maxRoomAttempts))
			if nearErr != nil {
				failCount++
				break
			}
			planData.NPCs = append(planData.NPCs, NPCSpec{
				Coord: consts.Coord[int]{X: int(tx), Y: int(ty)},
				Name:  entry.Name,
			})
			placed++
			failCount = 0
		}
	}

	if failCount > maxHostileNPCFailCount {
		log.Printf("HostileNPCPlanner: 敵NPC配置の試行回数が上限に達しました。配置数: %d/%d", placed, total)
	}
	return nil
}

// planWithRandomPosition は部屋がない場合のフォールバック。マップ全体からランダムに配置する
func (n *HostileNPCPlanner) planWithRandomPosition(planData *MetaPlan, total int) error {
	failCount := 0
	successCount := 0

	for successCount < total && failCount <= maxHostileNPCFailCount {
		tx := consts.Tile(planData.RNG.IntN(int(planData.Level.TileWidth)))
		ty := consts.Tile(planData.RNG.IntN(int(planData.Level.TileHeight)))

		if !planData.IsSpawnableTile(n.world, tx, ty) {
			failCount++
			continue
		}

		entry, err := selectSpawnEntry(n.plannerType.EnemyEntries, planData.RNG)
		if err != nil {
			return err
		}
		if entry.Name == "" {
			failCount++
			continue
		}

		planData.NPCs = append(planData.NPCs, NPCSpec{
			Coord: consts.Coord[int]{X: int(tx), Y: int(ty)},
			Name:  entry.Name,
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
