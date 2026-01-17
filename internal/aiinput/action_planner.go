package aiinput

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/actions"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/movement"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ActionPlanner はAIのアクション計画システム
type ActionPlanner interface {
	PlanAction(world w.World, aiEntity, playerEntity ecs.Entity, context *EntityContext, canSeePlayer bool) (actions.ActivityInterface, actions.ActionParams)
}

// DefaultActionPlanner は標準的なアクション計画実装
type DefaultActionPlanner struct{}

// NewActionPlanner は新しいActionPlannerを作成する
func NewActionPlanner() ActionPlanner {
	return &DefaultActionPlanner{}
}

// PlanAction は現在の状態に基づいてアクションを決定する
func (ap *DefaultActionPlanner) PlanAction(world w.World, aiEntity, playerEntity ecs.Entity, context *EntityContext, _ bool) (actions.ActivityInterface, actions.ActionParams) {
	switch context.Roaming.SubState {
	case gc.AIRoamingChasing:
		// 追跡モード：プレイヤーに向かって移動
		return ap.planChaseAction(world, aiEntity, playerEntity, context.GridElement)

	case gc.AIRoamingDriving:
		// 移動モード：ランダム移動
		return ap.planRandomMoveAction(world, aiEntity, context.GridElement)

	case gc.AIRoamingWaiting:
		// 待機モード：何もしない
		return &actions.WaitActivity{}, actions.ActionParams{Actor: aiEntity, Duration: 1, Reason: "AI待機"}

	default:
		// 不明な状態：待機
		return &actions.WaitActivity{}, actions.ActionParams{Actor: aiEntity, Duration: 1, Reason: "AIデフォルト待機"}
	}
}

// planChaseAction はプレイヤー追跡アクションを計画
func (ap *DefaultActionPlanner) planChaseAction(world w.World, aiEntity, playerEntity ecs.Entity, aiGrid *gc.GridElement) (actions.ActivityInterface, actions.ActionParams) {
	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)

	// プレイヤーと隣接タイルにいる場合は攻撃
	if ap.isAdjacent(aiGrid, playerGrid) {
		return &actions.AttackActivity{}, actions.ActionParams{
			Actor:  aiEntity,
			Target: &playerEntity,
		}
	}

	// プレイヤーに向かう方向を計算
	dx := int(playerGrid.X) - int(aiGrid.X)
	dy := int(playerGrid.Y) - int(aiGrid.Y)

	// 移動候補を優先度順で試行
	moveCandidates := ap.calculateMoveCandidates(dx, dy)

	// 移動可能な候補を探す
	for _, candidate := range moveCandidates {
		destX := int(aiGrid.X) + candidate.x
		destY := int(aiGrid.Y) + candidate.y

		if movement.CanMoveTo(world, destX, destY, aiEntity) {
			dest := gc.Position{X: gc.Pixel(destX), Y: gc.Pixel(destY)}
			return &actions.MoveActivity{}, actions.ActionParams{
				Actor:       aiEntity,
				Destination: &dest,
			}
		}
	}

	// どこにも移動できない場合は待機
	return &actions.WaitActivity{}, actions.ActionParams{Actor: aiEntity, Duration: 1, Reason: "AI追跡失敗"}
}

// planRandomMoveAction はランダム移動アクションを計画
func (ap *DefaultActionPlanner) planRandomMoveAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) (actions.ActivityInterface, actions.ActionParams) {
	// 30%の確率で待機
	if rand.Float64() < 0.3 {
		return &actions.WaitActivity{}, actions.ActionParams{Actor: aiEntity, Duration: 1, Reason: "AIランダム待機"}
	}

	// ランダムに隣接する8方向から選択
	directions := []struct{ x, y int }{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	// 移動可能な方向をシャッフルして試行
	shuffledDirections := make([]struct{ x, y int }, len(directions))
	copy(shuffledDirections, directions)

	// Fisher-Yatesアルゴリズムでシャッフル
	for i := len(shuffledDirections) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		shuffledDirections[i], shuffledDirections[j] = shuffledDirections[j], shuffledDirections[i]
	}

	// 移動可能な方向を探す
	for _, direction := range shuffledDirections {
		destX := int(aiGrid.X) + direction.x
		destY := int(aiGrid.Y) + direction.y

		if movement.CanMoveTo(world, destX, destY, aiEntity) {
			dest := gc.Position{X: gc.Pixel(destX), Y: gc.Pixel(destY)}
			return &actions.MoveActivity{}, actions.ActionParams{
				Actor:       aiEntity,
				Destination: &dest,
			}
		}
	}

	// どこにも移動できない場合は待機
	return &actions.WaitActivity{}, actions.ActionParams{Actor: aiEntity, Duration: 1, Reason: "AI追跡失敗"}
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
		absDx := dx
		if absDx < 0 {
			absDx = -absDx
		}
		absDy := dy
		if absDy < 0 {
			absDy = -absDy
		}

		if absDx > absDy {
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

// isAdjacent は2つのタイルが隣接しているかを判定する（同じタイルは除く）
func (ap *DefaultActionPlanner) isAdjacent(aiGrid, playerGrid *gc.GridElement) bool {
	dx := int(playerGrid.X) - int(aiGrid.X)
	dy := int(playerGrid.Y) - int(aiGrid.Y)

	// 絶対値で距離を計算（チェビシェフ距離）
	absDx := dx
	if absDx < 0 {
		absDx = -absDx
	}
	absDy := dy
	if absDy < 0 {
		absDy = -absDy
	}

	// 隣接タイル（距離1だが同じタイルは除く）
	maxDistance := absDx
	if absDy > maxDistance {
		maxDistance = absDy
	}
	return maxDistance == 1
}
