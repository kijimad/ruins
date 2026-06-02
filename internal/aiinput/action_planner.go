package aiinput

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// eightDirections は隣接8方向の座標差分
var eightDirections = []struct{ x, y int }{
	{-1, -1}, {0, -1}, {1, -1},
	{-1, 0}, {1, 0},
	{-1, 1}, {0, 1}, {1, 1},
}

// ActionPlanner はAIのアクション計画システム
type ActionPlanner interface {
	PlanAction(world w.World, aiEntity, playerEntity ecs.Entity, context *EntityContext) (activity.Behavior, activity.ActionParams)
}

// DefaultActionPlanner は標準的なアクション計画実装
type DefaultActionPlanner struct{}

// NewActionPlanner は新しいActionPlannerを作成する
func NewActionPlanner() ActionPlanner {
	return &DefaultActionPlanner{}
}

// PlanAction は現在の状態に基づいてアクションを決定する
func (ap *DefaultActionPlanner) PlanAction(world w.World, aiEntity, playerEntity ecs.Entity, context *EntityContext) (activity.Behavior, activity.ActionParams) {
	switch context.Roaming.SubState {
	case gc.AIRoamingChasing:
		return ap.planChaseAction(world, aiEntity, playerEntity, context.GridElement)
	case gc.AIRoamingFleeing:
		return ap.planFleeAction(world, aiEntity, playerEntity, context.GridElement)
	case gc.AIRoamingDriving:
		return ap.planDrivingAction(world, aiEntity, context)
	case gc.AIRoamingWaiting:
		return waitAction(aiEntity, "AI待機")
	default:
		return waitAction(aiEntity, "AIデフォルト待機")
	}
}

// planChaseAction はプレイヤー追跡アクションを計画する
func (ap *DefaultActionPlanner) planChaseAction(world w.World, aiEntity, playerEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)

	// プレイヤーと隣接タイルにいる場合は攻撃
	if ap.isAdjacent(aiGrid, playerGrid) {
		return &activity.AttackActivity{}, activity.ActionParams{
			Actor:  aiEntity,
			Target: &playerEntity,
		}
	}

	// プレイヤーに向かう方向を計算
	dx := int(playerGrid.X) - int(aiGrid.X)
	dy := int(playerGrid.Y) - int(aiGrid.Y)

	candidates := ap.calculateMoveCandidates(dx, dy)
	if b, p, ok := ap.tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b, p
	}

	return waitAction(aiEntity, "AI追跡失敗")
}

// planFleeAction はプレイヤーから逃亡するアクションを計画する。追跡の逆方向に移動する
func (ap *DefaultActionPlanner) planFleeAction(world w.World, aiEntity, playerEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)

	// プレイヤーと逆方向に移動する
	dx := int(aiGrid.X) - int(playerGrid.X)
	dy := int(aiGrid.Y) - int(playerGrid.Y)

	candidates := ap.calculateMoveCandidates(dx, dy)
	if b, p, ok := ap.tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b, p
	}

	// 逃げ場がない場合はランダム移動を試みる
	return ap.planRandomMoveAction(world, aiEntity, aiGrid)
}

// planRandomMoveAction はランダム移動アクションを計画する
func (ap *DefaultActionPlanner) planRandomMoveAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	// 30%の確率で待機
	if rand.Float64() < 0.3 {
		return waitAction(aiEntity, "AIランダム待機")
	}

	shuffled := shuffledEightDirections()
	fromX, fromY := int(aiGrid.X), int(aiGrid.Y)
	for _, d := range shuffled {
		destX := fromX + d.x
		destY := fromY + d.y
		if activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
			return moveAction(aiEntity, destX, destY)
		}
	}

	return waitAction(aiEntity, "AIランダム移動失敗")
}

// MoveCandidate は移動候補を表す
type MoveCandidate struct {
	x, y int
}

// calculateMoveCandidates はプレイヤーに向かう移動候補を計算する
func (ap *DefaultActionPlanner) calculateMoveCandidates(dx, dy int) []MoveCandidate {
	var candidates []MoveCandidate

	if dx != 0 && dy != 0 {
		// 斜め移動が最優先
		moveX := 1
		if dx < 0 {
			moveX = -1
		}
		moveY := 1
		if dy < 0 {
			moveY = -1
		}
		candidates = append(candidates, MoveCandidate{moveX, moveY})

		// 代替案として軸に沿った移動
		if geometry.Abs(dx) > geometry.Abs(dy) {
			candidates = append(candidates, MoveCandidate{moveX, 0})
			candidates = append(candidates, MoveCandidate{0, moveY})
		} else {
			candidates = append(candidates, MoveCandidate{0, moveY})
			candidates = append(candidates, MoveCandidate{moveX, 0})
		}
	} else if dx != 0 {
		// 水平移動のみ
		moveX := 1
		if dx < 0 {
			moveX = -1
		}
		candidates = append(candidates, MoveCandidate{moveX, 0})
		// 代替案として垂直移動
		candidates = append(candidates, MoveCandidate{0, 1})
		candidates = append(candidates, MoveCandidate{0, -1})
	} else if dy != 0 {
		// 垂直移動のみ
		moveY := 1
		if dy < 0 {
			moveY = -1
		}
		candidates = append(candidates, MoveCandidate{0, moveY})
		// 代替案として水平移動
		candidates = append(candidates, MoveCandidate{1, 0})
		candidates = append(candidates, MoveCandidate{-1, 0})
	}

	return candidates
}

// planDrivingAction はMovementPatternに基づく移動アクションを計画する
func (ap *DefaultActionPlanner) planDrivingAction(world w.World, aiEntity ecs.Entity, context *EntityContext) (activity.Behavior, activity.ActionParams) {
	switch context.MovementPattern {
	case gc.MovementStationary:
		return waitAction(aiEntity, "AI固定待機")
	case gc.MovementWander:
		return ap.planWanderAction(world, aiEntity, context.GridElement)
	case gc.MovementWallHug:
		return ap.planWallHugAction(world, aiEntity, context.GridElement)
	case gc.MovementSwarm:
		return ap.planSwarmAction(world, aiEntity, context.GridElement)
	case gc.MovementPatrol:
		return ap.planPatrolAction(world, aiEntity, context)
	case gc.MovementTerritorial:
		return ap.planTerritorialAction(world, aiEntity, context)
	default:
		return ap.planRandomMoveAction(world, aiEntity, context.GridElement)
	}
}

// planWanderAction は低頻度でランダム移動するアクションを計画する。街のNPC向け
func (ap *DefaultActionPlanner) planWanderAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	// 80%の確率で待機する
	if rand.Float64() < 0.8 {
		return waitAction(aiEntity, "AI徘徊待機")
	}
	return ap.planRandomMoveAction(world, aiEntity, aiGrid)
}

// planWallHugAction は壁に隣接するタイルを優先して移動するアクションを計画する
func (ap *DefaultActionPlanner) planWallHugAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	// 30%の確率で待機
	if rand.Float64() < 0.3 {
		return waitAction(aiEntity, "AI壁沿い待機")
	}

	si := worldhelper.GetSpatialIndex(world)

	// 移動可能な方向を壁隣接スコアでソートする
	type scoredDir struct {
		x, y  int
		score int
	}
	var candidates []scoredDir

	fromX, fromY := int(aiGrid.X), int(aiGrid.Y)
	for _, d := range eightDirections {
		destX := fromX + d.x
		destY := fromY + d.y

		if !activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
			continue
		}

		// 移動先の隣接4方向に壁がいくつあるかをカウントする
		wallCount := 0
		for _, adj := range []struct{ x, y int }{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			if si.IsBlockPass(destX+adj.x, destY+adj.y) {
				wallCount++
			}
		}
		candidates = append(candidates, scoredDir{d.x, d.y, wallCount})
	}

	if len(candidates) == 0 {
		return waitAction(aiEntity, "AI壁沿い移動失敗")
	}

	// 同スコアの最高スコア候補からランダムに選ぶ
	best := candidates[0].score
	for _, c := range candidates[1:] {
		if c.score > best {
			best = c.score
		}
	}
	var tied []scoredDir
	for _, c := range candidates {
		if c.score == best {
			tied = append(tied, c)
		}
	}
	chosen := tied[rand.IntN(len(tied))]

	return moveAction(aiEntity, fromX+chosen.x, fromY+chosen.y)
}

// planSwarmAction は最寄りのAIエンティティに接近するアクションを計画する
func (ap *DefaultActionPlanner) planSwarmAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	// 最寄りのAIエンティティを探す
	var nearestGrid *gc.GridElement
	nearestDist := -1

	world.Manager.Join(
		world.Components.AIMoveFSM,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if entity == aiEntity {
			return
		}
		if entity.HasComponent(world.Components.Dead) {
			return
		}

		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		dist := geometry.Abs(int(grid.X)-int(aiGrid.X)) + geometry.Abs(int(grid.Y)-int(aiGrid.Y))
		if nearestDist < 0 || dist < nearestDist {
			nearestDist = dist
			nearestGrid = grid
		}
	}))

	// 仲間が見つからない、または既に隣接している場合はランダム移動する
	if nearestGrid == nil || nearestDist <= 1 {
		return ap.planRandomMoveAction(world, aiEntity, aiGrid)
	}

	// 仲間に向かって移動する
	dx := int(nearestGrid.X) - int(aiGrid.X)
	dy := int(nearestGrid.Y) - int(aiGrid.Y)

	candidates := ap.calculateMoveCandidates(dx, dy)
	if b, p, ok := ap.tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b, p
	}

	return ap.planRandomMoveAction(world, aiEntity, aiGrid)
}

// territorialRadius はTerritorial移動パターンのスポーン地点からの最大距離（タイル数）
const territorialRadius = 5

// planPatrolAction は一方向に直進し、進めなくなったら反転する巡回アクションを計画する
func (ap *DefaultActionPlanner) planPatrolAction(world w.World, aiEntity ecs.Entity, context *EntityContext) (activity.Behavior, activity.ActionParams) {
	aiGrid := context.GridElement
	roaming := context.Roaming
	fromX, fromY := int(aiGrid.X), int(aiGrid.Y)

	// 現在の巡回方向に移動を試みる
	destX := fromX + roaming.PatrolDirX
	destY := fromY + roaming.PatrolDirY
	if activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
		return moveAction(aiEntity, destX, destY)
	}

	// 進めないので方向を反転する
	roaming.PatrolDirX = -roaming.PatrolDirX
	roaming.PatrolDirY = -roaming.PatrolDirY

	// 反転方向に移動を試みる
	destX = fromX + roaming.PatrolDirX
	destY = fromY + roaming.PatrolDirY
	if activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
		return moveAction(aiEntity, destX, destY)
	}

	return waitAction(aiEntity, "AI巡回移動失敗")
}

// planTerritorialAction はスポーン地点から一定範囲内でランダム移動するアクションを計画する
func (ap *DefaultActionPlanner) planTerritorialAction(world w.World, aiEntity ecs.Entity, context *EntityContext) (activity.Behavior, activity.ActionParams) {
	aiGrid := context.GridElement
	roaming := context.Roaming
	fromX, fromY := int(aiGrid.X), int(aiGrid.Y)

	for _, d := range shuffledEightDirections() {
		destX := fromX + d.x
		destY := fromY + d.y

		// スポーン地点からの距離をチェックする
		dx := geometry.Abs(destX - roaming.SpawnX)
		dy := geometry.Abs(destY - roaming.SpawnY)
		if dx > territorialRadius || dy > territorialRadius {
			continue
		}

		if activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
			return moveAction(aiEntity, destX, destY)
		}
	}

	return waitAction(aiEntity, "AI縄張り移動失敗")
}

// tryMoveCandidates は移動候補を順に試行し、最初に移動可能な方向へ移動するアクションを返す。
// 移動可能な候補がなければ nil を返す
func (ap *DefaultActionPlanner) tryMoveCandidates(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement, candidates []MoveCandidate) (activity.Behavior, activity.ActionParams, bool) {
	fromX, fromY := int(aiGrid.X), int(aiGrid.Y)
	for _, candidate := range candidates {
		destX := fromX + candidate.x
		destY := fromY + candidate.y
		if activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
			b, p := moveAction(aiEntity, destX, destY)
			return b, p, true
		}
	}
	return nil, activity.ActionParams{}, false
}

// moveAction は指定座標への移動アクションを生成する
func moveAction(aiEntity ecs.Entity, destX, destY int) (activity.Behavior, activity.ActionParams) {
	dest := gc.GridElement{X: consts.Tile(destX), Y: consts.Tile(destY)}
	return &activity.MoveActivity{}, activity.ActionParams{
		Actor:       aiEntity,
		Destination: &dest,
	}
}

// waitAction は待機アクションを生成する
func waitAction(aiEntity ecs.Entity, reason string) (activity.Behavior, activity.ActionParams) {
	return &activity.WaitActivity{}, activity.ActionParams{Actor: aiEntity, Duration: 1, Reason: reason}
}

// shuffledEightDirections は8方向をシャッフルして返す
func shuffledEightDirections() []struct{ x, y int } {
	shuffled := make([]struct{ x, y int }, len(eightDirections))
	copy(shuffled, eightDirections)
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}

// isAdjacent は2つのタイルが隣接しているかを判定する（同じタイルは除く）
func (ap *DefaultActionPlanner) isAdjacent(aiGrid, playerGrid *gc.GridElement) bool {
	return geometry.IsAdjacent(int(aiGrid.X), int(aiGrid.Y), int(playerGrid.X), int(playerGrid.Y))
}
