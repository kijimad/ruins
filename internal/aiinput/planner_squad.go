package aiinput

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// hpRetreatThreshold はHP割合がこの値以下のとき、ポリシーに関わらず後退する
const hpRetreatThreshold = 25

// escortMaxDistance は護衛ポリシーでリーダーから離れてよい最大距離を表す
const escortMaxDistance = 2

// vanguardMaxDistance は前衛ポリシーでリーダーから離れてよい最大距離を表す
const vanguardMaxDistance = 3

// squadPlanner は隊員用の行動計画を実装する。
// リーダー追従とアイテム処理を含む優先度ベースの行動決定を行う
type squadPlanner struct {
	visionSystem VisionSystem
	logger       *logger.Logger
	rng          *rand.Rand
}

func newSquadPlanner(rng *rand.Rand) *squadPlanner {
	return &squadPlanner{
		visionSystem: NewVisionSystem(),
		logger:       logger.New(logger.CategoryTurn),
		rng:          rng,
	}
}

// squadSnapshot は隊員AIの1ターンの判断に使う情報の束。
// コンポーネントへのポインタは構造変更で無効化されるため、ターンごとに作り直す
type squadSnapshot struct {
	Grid         *gc.GridElement
	Squad        *gc.SquadAI
	LeaderEntity ecs.Entity
	LeaderGrid   *gc.GridElement
}

// Plan はsquadSnapshotを収集し、優先度チェーンで行動を決定する
func (sp *squadPlanner) Plan(world w.World, entity ecs.Entity) activity.Behavior {
	snap, ok := sp.gatherSquadSnapshot(world, entity)
	if !ok {
		return nil
	}
	return sp.planAction(world, entity, snap)
}

// gatherSquadSnapshot は隊員の行動に必要な情報をまとめる
func (sp *squadPlanner) gatherSquadSnapshot(world w.World, entity ecs.Entity) (*squadSnapshot, bool) {
	grid := world.Components.GridElement.Get(entity)

	squadComp := world.Components.SquadAI.Get(entity)
	if squadComp == nil {
		sp.logger.Warn("隊員にSquadAIがない", "entity", entity)
		return nil, false
	}

	si := query.GetSpatialIndex(world)
	if si == nil || si.PlayerEntity == nil {
		sp.logger.Warn("プレイヤーが見つからない", "entity", entity)
		return nil, false
	}
	leader := *si.PlayerEntity

	if !world.Components.GridElement.Has(leader) {
		sp.logger.Warn("リーダーにGridElementがない", "entity", entity)
		return nil, false
	}

	return &squadSnapshot{
		Grid:         grid,
		Squad:        squadComp,
		LeaderEntity: leader,
		LeaderGrid:   world.Components.GridElement.Get(leader),
	}, true
}

// planAction はポリシーと状況に基づいてアクションを決定する。
// 優先順位: HP低下時後退 → エリア制限 → 補給 → 戦闘 → アイテム転送 → アイテム拾得 → 位置ポリシー
func (sp *squadPlanner) planAction(world w.World, entity ecs.Entity, snap *squadSnapshot) activity.Behavior {
	if sp.shouldRetreatLowHP(world, entity) {
		if b, ok := sp.planRetreatAction(world, entity, snap); ok {
			return b
		}
	}

	if sp.isOutsideExploredArea(world, snap.Grid) {
		if b, ok := sp.planReturnToExploredArea(world, entity, snap); ok {
			return b
		}
	}

	if b, ok := sp.planSupplyAction(world, entity, snap); ok {
		return b
	}

	if b, ok := sp.planCombatAction(world, entity, snap); ok {
		return b
	}

	if b, ok := sp.planItemHandlingAction(world, entity, snap); ok {
		return b
	}

	if b, ok := sp.planItemPickupAction(world, entity, snap); ok {
		return b
	}

	return sp.planPositionAction(world, entity, snap)
}

// shouldRetreatLowHP はHP25%以下で後退すべきかを判定する
func (sp *squadPlanner) shouldRetreatLowHP(world w.World, entity ecs.Entity) bool {
	hp := world.Components.HP.Get(entity)
	if hp == nil {
		return false
	}
	if hp.Max == 0 {
		return false
	}
	return hp.Current*100/hp.Max <= hpRetreatThreshold
}

// planRetreatAction はリーダーに向かって後退するアクションを計画する
func (sp *squadPlanner) planRetreatAction(world w.World, entity ecs.Entity, snap *squadSnapshot) (activity.Behavior, bool) {
	sp.logger.Debug("隊員HP低下、後退", "entity", entity)
	return sp.tryMoveToward(world, entity, snap.Grid, snap.LeaderGrid)
}

// isOutsideExploredArea は現在位置が探索済みエリア外かを判定する
func (sp *squadPlanner) isOutsideExploredArea(world w.World, grid *gc.GridElement) bool {
	field := query.GetCurrentStageField(world)
	if field == nil || field.ExploredTiles == nil {
		return false
	}
	return !field.ExploredTiles[*grid]
}

// planReturnToExploredArea は最寄りの探索済みマスへ移動するアクションを計画する
func (sp *squadPlanner) planReturnToExploredArea(world w.World, entity ecs.Entity, snap *squadSnapshot) (activity.Behavior, bool) {
	sp.logger.Debug("隊員がエリア外、リーダーに向かう", "entity", entity)
	return sp.tryMoveToward(world, entity, snap.Grid, snap.LeaderGrid)
}

// planCombatAction は戦闘ポリシーに基づくアクションを計画する
func (sp *squadPlanner) planCombatAction(world w.World, entity ecs.Entity, snap *squadSnapshot) (activity.Behavior, bool) {
	switch snap.Squad.CombatCurrent {
	case gc.CombatAttack:
		return sp.planAttackAction(world, entity, snap)
	case gc.CombatEvade:
		return sp.planEvadeAction(world, entity, snap)
	default:
		return nil, false
	}
}

// planAttackAction は攻撃ポリシーに基づくアクションを計画する。
// 隣接する敵がいれば攻撃し、視界内の敵がいれば接近する。
// 移動しても敵に近づけない場合は諦めて次の優先度に進む
func (sp *squadPlanner) planAttackAction(world w.World, entity ecs.Entity, snap *squadSnapshot) (activity.Behavior, bool) {
	nearestEnemy, nearestGrid, dist := sp.findNearestEnemy(world, entity, snap)
	if nearestEnemy == nil {
		return nil, false
	}

	if dist == 1 {
		return &activity.AttackActivity{Target: *nearestEnemy}, true
	}

	return sp.tryMoveToward(world, entity, snap.Grid, nearestGrid)
}

// planEvadeAction は回避ポリシーに基づくアクションを計画する。
// 視界内の最寄りの敵から距離を取る
func (sp *squadPlanner) planEvadeAction(world w.World, entity ecs.Entity, snap *squadSnapshot) (activity.Behavior, bool) {
	nearestEnemy, _, _ := sp.findNearestEnemy(world, entity, snap)
	if nearestEnemy == nil {
		return nil, false
	}

	enemyGrid := world.Components.GridElement.Get(*nearestEnemy)
	return sp.tryMoveAway(world, entity, snap.Grid, enemyGrid)
}

// planPositionAction は位置ポリシーに基づくアクションを計画する
func (sp *squadPlanner) planPositionAction(world w.World, entity ecs.Entity, snap *squadSnapshot) activity.Behavior {
	switch snap.Squad.Movement {
	case gc.SquadEscort:
		return sp.planEscortAction(world, entity, snap)
	case gc.SquadVanguard:
		return sp.planVanguardAction(world, entity, snap)
	case gc.SquadPatrol:
		return sp.planSquadPatrolAction(world, entity, snap)
	case gc.SquadStationary:
		return waitAction("隊員待機")
	case gc.SquadRetreat:
		return sp.planEscortAction(world, entity, snap)
	default:
		return waitAction("隊員デフォルト待機")
	}
}

// planEscortAction はリーダーから2マス以内にとどまるアクションを計画する
func (sp *squadPlanner) planEscortAction(world w.World, entity ecs.Entity, snap *squadSnapshot) activity.Behavior {
	dist := gridDistance(snap.Grid, snap.LeaderGrid)
	if dist <= escortMaxDistance {
		return waitAction("隊員護衛位置")
	}
	if b, ok := sp.tryMoveToward(world, entity, snap.Grid, snap.LeaderGrid); ok {
		return b
	}
	return waitAction("隊員護衛移動失敗")
}

// planVanguardAction はリーダーの前方に展開するアクションを計画する。
// リーダーから離れすぎている場合はリーダーに接近する
func (sp *squadPlanner) planVanguardAction(world w.World, entity ecs.Entity, snap *squadSnapshot) activity.Behavior {
	dist := gridDistance(snap.Grid, snap.LeaderGrid)
	if dist > vanguardMaxDistance {
		if b, ok := sp.tryMoveToward(world, entity, snap.Grid, snap.LeaderGrid); ok {
			return b
		}
		return waitAction("隊員前衛接近失敗")
	}
	if b, ok := sp.tryRandomMove(world, entity, snap); ok {
		return b
	}
	return waitAction("隊員前衛移動失敗")
}

// planSquadPatrolAction は探索済みエリア内を自律的に巡回するアクションを計画する
func (sp *squadPlanner) planSquadPatrolAction(world w.World, entity ecs.Entity, snap *squadSnapshot) activity.Behavior {
	if b, ok := sp.tryRandomMove(world, entity, snap); ok {
		return b
	}
	return waitAction("隊員巡回移動失敗")
}

// planItemPickupAction は拾得可能アイテムを拾うアクションを計画する。
// 足元にアイテムがあれば拾い、なければ視界内のアイテムに向かって移動する。
// PolicyIgnoreの場合は何もしない
func (sp *squadPlanner) planItemPickupAction(world w.World, entity ecs.Entity, snap *squadSnapshot) (activity.Behavior, bool) {
	if snap.Squad.ItemPickup == gc.PolicyIgnore {
		return nil, false
	}

	hasPickableHere := false
	var nearestItemGrid *gc.GridElement
	nearestDist := -1

	itemQuery := query.ActiveFilter2[gc.GridElement, gc.LocationOnField](world).Query()
	for itemQuery.Next() {
		item := itemQuery.Entity()
		if !query.IsPickable(item, world) {
			continue
		}
		grid := world.Components.GridElement.Get(item)

		if grid.X == snap.Grid.X && grid.Y == snap.Grid.Y {
			hasPickableHere = true
			continue
		}

		dist := gridDistance(snap.Grid, grid)
		if dist > int(snap.Squad.ViewDistance) {
			continue
		}
		if nearestDist < 0 || dist < nearestDist {
			nearestItemGrid = grid
			nearestDist = dist
		}
	}

	if hasPickableHere {
		sp.logger.Debug("隊員アイテム拾得", "entity", entity, "x", snap.Grid.X, "y", snap.Grid.Y)
		dest := *snap.Grid
		return &activity.PickupActivity{Destination: &dest}, true
	}

	if nearestItemGrid != nil {
		sp.logger.Debug("隊員アイテムへ移動", "entity", entity, "dist", nearestDist)
		return sp.tryMoveToward(world, entity, snap.Grid, nearestItemGrid)
	}

	return nil, false
}

// planSupplyAction は空腹の隊員に補給行動を計画する。
// まず自分の背嚢の食料を食べ、無ければ共有プールであるリーダーの所持品へ接近して受け取る。
// 敵が視界内にいる間は発火しない
func (sp *squadPlanner) planSupplyAction(world w.World, entity ecs.Entity, snap *squadSnapshot) (activity.Behavior, bool) {
	if snap.Squad.Supply != gc.SupplyAuto {
		return nil, false
	}
	if !world.Components.Hunger.Has(entity) {
		return nil, false
	}
	if world.Components.Hunger.Get(entity).GetLevel() < gc.HungerHungry {
		return nil, false
	}
	// 戦闘中は食べない
	if enemy, _, _ := sp.findNearestEnemy(world, entity, snap); enemy != nil {
		return nil, false
	}

	// 自分の背嚢から食べる。栄養価の低いものを先に消費して高価値食料を温存する
	if food, ok := findLowestNutritionFood(world, entity); ok {
		sp.logger.Debug("隊員が食事する", "entity", entity)
		return &activity.UseItemActivity{Target: food}, true
	}

	// 共有プールから受け取る
	poolFood, ok := findLowestNutritionFood(world, snap.LeaderEntity)
	if !ok {
		// プール枯渇。受け取れず空腹が進む。食料確保はプレイヤーの兵站判断に残す
		sp.logger.Debug("隊の食料が尽きている", "entity", entity)
		return nil, false
	}
	if gridDistance(snap.Grid, snap.LeaderGrid) <= 1 {
		sp.logger.Debug("隊員が食料を受け取る", "entity", entity)
		// 1食ぶんだけ引く。丸ごと受け取ると共有プールが一気に空になり、他の隊員が飢える
		return &activity.TransferActivity{Target: poolFood, Recipient: entity, Single: true}, true
	}
	return sp.tryMoveToward(world, entity, snap.Grid, snap.LeaderGrid)
}

// findLowestNutritionFood は所持品から最も栄養価の低い食料を返す
func findLowestNutritionFood(world w.World, owner ecs.Entity) (ecs.Entity, bool) {
	best := gc.InvalidEntity
	bestNutrition := -1
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		item := q.Entity()
		if world.Components.LocationInBackpack.Get(item).Owner != owner {
			continue
		}
		if !world.Components.ProvidesNutrition.Has(item) {
			continue
		}
		nutrition := world.Components.ProvidesNutrition.Get(item).Amount
		if bestNutrition < 0 || nutrition < bestNutrition {
			best = item
			bestNutrition = nutrition
		}
	}
	return best, bestNutrition >= 0
}

// planItemHandlingAction はバックパック内のアイテムをポリシーに基づいて処理する。
// PolicyDistributeの場合はリーダーにアイテムを転送する
func (sp *squadPlanner) planItemHandlingAction(world w.World, entity ecs.Entity, snap *squadSnapshot) (activity.Behavior, bool) {
	if snap.Squad.ItemHandling != gc.PolicyDistribute {
		return nil, false
	}

	dist := gridDistance(snap.Grid, snap.LeaderGrid)
	if dist > 1 {
		return nil, false
	}

	var itemToTransfer *ecs.Entity
	backpackQuery := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for backpackQuery.Next() {
		item := backpackQuery.Entity()
		if itemToTransfer != nil {
			continue
		}
		loc := world.Components.LocationInBackpack.Get(item)
		if loc.Owner == entity {
			itemToTransfer = &item
		}
	}

	if itemToTransfer == nil {
		return nil, false
	}

	sp.logger.Debug("隊員アイテム転送", "entity", entity, "item", *itemToTransfer)
	return &activity.TransferActivity{Target: *itemToTransfer, Recipient: snap.LeaderEntity}, true
}

// findNearestEnemy は視界内の最も近い敵を探す
func (sp *squadPlanner) findNearestEnemy(world w.World, entity ecs.Entity, snap *squadSnapshot) (*ecs.Entity, *gc.GridElement, int) {
	return query.FindNearestCharacter(world, entity, snap.Grid, func(target ecs.Entity) bool {
		return query.FactionRelation(world, entity, target) == query.RelationHostile &&
			sp.visionSystem.CanSeeTarget(world, entity, target, snap.Squad.ViewDistance)
	})
}

// tryMoveToward はBFSで壁を迂回した最短経路でターゲットに向かう移動を試みる
func (sp *squadPlanner) tryMoveToward(world w.World, entity ecs.Entity, from, target *gc.GridElement) (activity.Behavior, bool) {
	next, ok := activity.FindNextStep(world, entity, from.Coord, target.Coord)
	if !ok {
		return nil, false
	}

	if !activity.CanMoveTo(world, next, from.Coord, entity) {
		return nil, false
	}

	return moveAction(next), true
}

// tryMoveAway はターゲットから離れる移動を試みる
func (sp *squadPlanner) tryMoveAway(world w.World, entity ecs.Entity, from, threat *gc.GridElement) (activity.Behavior, bool) {
	candidates := calculateMoveCandidates(from.Sub(threat.Coord))
	return tryMoveCandidates(world, entity, from, candidates)
}

// tryRandomMove は探索済みエリア内でランダム移動を試みる
func (sp *squadPlanner) tryRandomMove(world w.World, entity ecs.Entity, snap *squadSnapshot) (activity.Behavior, bool) {
	field := query.GetCurrentStageField(world)
	from := snap.Grid.Coord

	for _, d := range shuffledEightDirections(sp.rng) {
		dest := from.Add(d)

		if field != nil && field.ExploredTiles != nil {
			destGrid := gc.GridElement{Coord: dest}
			if !field.ExploredTiles[destGrid] {
				continue
			}
		}

		if activity.CanMoveTo(world, dest, from, entity) {
			return moveAction(dest), true
		}
	}
	return nil, false
}
