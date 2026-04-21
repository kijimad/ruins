package mapplanner

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// ポータル配置用の定数
const (
	maxPortalPlacementAttempts = 200 // ポータル配置処理の最大試行回数
	escapePortalInterval       = 5   // 帰還ポータル配置間隔（n階層ごと）
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
	playerPos, err := planData.GetPlayerStartPosition()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectivity, err)
	}
	pathFinder := NewPathFinder(planData)

	// 次の階へ進むポータルを配置する
	placed := false
	for attempt := 0; attempt < maxPortalPlacementAttempts; attempt++ {
		x := planData.RNG.IntN(int(planData.Level.TileWidth))
		y := planData.RNG.IntN(int(planData.Level.TileHeight))

		if planData.IsSpawnableTile(p.world, consts.Tile(x), consts.Tile(y)) && pathFinder.IsReachable(playerPos.X, playerPos.Y, x, y) {
			planData.NextPortals = append(planData.NextPortals, consts.Coord[int]{X: x, Y: y})
			placed = true
			break
		}
	}
	if !placed {
		return fmt.Errorf("%w: NextPortalの配置に失敗しました（%d回試行）", ErrConnectivity, maxPortalPlacementAttempts)
	}

	if p.world.Resources.Dungeon == nil {
		return fmt.Errorf("Dungeonが初期化されてない")
	}
	// 間隔ごとに帰還ポータルを配置する
	if p.world.Resources.Dungeon.Depth%escapePortalInterval == 0 {
		placed = false
		for attempt := 0; attempt < maxPortalPlacementAttempts; attempt++ {
			x := planData.RNG.IntN(int(planData.Level.TileWidth))
			y := planData.RNG.IntN(int(planData.Level.TileHeight))

			if planData.IsSpawnableTile(p.world, consts.Tile(x), consts.Tile(y)) && pathFinder.IsReachable(playerPos.X, playerPos.Y, x, y) {
				planData.EscapePortals = append(planData.EscapePortals, consts.Coord[int]{X: x, Y: y})
				placed = true
				break
			}
		}
		if !placed {
			return fmt.Errorf("%w: EscapePortalの配置に失敗しました（%d回試行）", ErrConnectivity, maxPortalPlacementAttempts)
		}
	}

	return nil
}
