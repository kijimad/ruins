package mapplanner

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// ポータル配置用の定数
const (
	maxPortalPlacementAttempts = 200 // ポータル配置処理の最大試行回数
	minPortalDistance          = 10  // ポータル間およびプレイヤーからの最低歩数
)

// PortalPlanner はポータル配置を担当するプランナー
type PortalPlanner struct {
	world       w.World
	plannerType PlannerType
}

// NewPortalPlanner はポータルプランナーを作成する
func NewPortalPlanner(world w.World, plannerType PlannerType) *PortalPlanner {
	return &PortalPlanner{
		world:       world,
		plannerType: plannerType,
	}
}

// PlanMeta はポータルをMetaPlanに追加する
func (p *PortalPlanner) PlanMeta(planData *MetaPlan) error {
	// テンプレートベースのマップではポータルは固定位置なのでスキップ
	if p.plannerType.UseFixedPortalPos {
		return nil
	}

	// プロシージャルマップの場合はランダム配置（到達可能な位置のみ）
	pathFinder := NewPathFinder(planData)
	playerPos, err := pathFinder.FindPlayerStartPosition()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConnectivity, err)
	}

	// 最低距離付きセレクタを優先し、失敗時は距離制約なしにフォールバック
	refs := []consts.Coord[consts.Tile]{playerPos}
	distSelector := minDistanceReachableSelector(pathFinder, refs, minPortalDistance, maxPortalPlacementAttempts)
	fallbackSelector := reachableSelector(pathFinder, playerPos, maxPortalPlacementAttempts)

	// 次の階へ進むポータルを配置する
	x, y, err := findPosition(planData, p.world, distSelector, fallbackSelector)
	if err != nil {
		return fmt.Errorf("%w: NextPortalの配置に失敗しました（%d回試行）", ErrConnectivity, maxPortalPlacementAttempts)
	}
	nextPortalPos := consts.Coord[consts.Tile]{X: x, Y: y}
	planData.NextPortals = append(planData.NextPortals, nextPortalPos)

	return nil
}
