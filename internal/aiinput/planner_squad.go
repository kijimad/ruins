package aiinput

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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

// squadContext は隊員AIに必要な情報をまとめる
type squadContext struct {
	Grid         *gc.GridElement
	Squad        *gc.SquadAI
	LeaderEntity ecs.Entity
	LeaderGrid   *gc.GridElement
}

// Plan はsquadContextを収集し、優先度チェーンで行動を決定する
func (sp *squadPlanner) Plan(world w.World, entity ecs.Entity) activity.Behavior {
	ctx, ok := sp.gatherSquadContext(world, entity)
	if !ok {
		return nil
	}
	return sp.planAction(world, entity, ctx)
}

// gatherSquadContext は隊員の行動に必要なコンテキストを収集する
func (sp *squadPlanner) gatherSquadContext(world w.World, entity ecs.Entity) (*squadContext, bool) {
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

	return &squadContext{
		Grid:         grid,
		Squad:        squadComp,
		LeaderEntity: leader,
		LeaderGrid:   world.Components.GridElement.Get(leader),
	}, true
}

// planAction はポリシーと状況に基づいてアクションを決定する。
// 優先順位: HP低下時後退 → エリア制限 → 戦闘 → アイテム転送 → アイテム拾得 → 位置ポリシー
func (sp *squadPlanner) planAction(world w.World, entity ecs.Entity, ctx *squadContext) activity.Behavior {
	if sp.shouldRetreatLowHP(world, entity) {
		if b, ok := sp.planRetreatAction(world, entity, ctx); ok {
			return b
		}
	}

	if sp.isOutsideExploredArea(world, ctx.Grid) {
		if b, ok := sp.planReturnToExploredArea(world, entity, ctx); ok {
			return b
		}
	}

	if b, ok := sp.planCombatAction(world, entity, ctx); ok {
		return b
	}

	if b, ok := sp.planItemHandlingAction(world, entity, ctx); ok {
		return b
	}

	if b, ok := sp.planItemPickupAction(world, entity, ctx); ok {
		return b
	}

	return sp.planPositionAction(world, entity, ctx)
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
func (sp *squadPlanner) planRetreatAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, bool) {
	sp.logger.Debug("隊員HP低下、後退", "entity", entity)
	return sp.tryMoveToward(world, entity, ctx.Grid, ctx.LeaderGrid)
}

// isOutsideExploredArea は現在位置が探索済みエリア外かを判定する
func (sp *squadPlanner) isOutsideExploredArea(world w.World, grid *gc.GridElement) bool {
	dungeon := query.GetDungeon(world)
	if dungeon == nil || dungeon.ExploredTiles == nil {
		return false
	}
	return !dungeon.ExploredTiles[*grid]
}

// planReturnToExploredArea は最寄りの探索済みマスへ移動するアクションを計画する
func (sp *squadPlanner) planReturnToExploredArea(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, bool) {
	sp.logger.Debug("隊員がエリア外、リーダーに向かう", "entity", entity)
	return sp.tryMoveToward(world, entity, ctx.Grid, ctx.LeaderGrid)
}

// planCombatAction は戦闘ポリシーに基づくアクションを計画する
func (sp *squadPlanner) planCombatAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, bool) {
	switch ctx.Squad.CombatCurrent {
	case gc.CombatAttack:
		return sp.planAttackAction(world, entity, ctx)
	case gc.CombatEvade:
		return sp.planEvadeAction(world, entity, ctx)
	default:
		return nil, false
	}
}

// planAttackAction は攻撃ポリシーに基づくアクションを計画する。
// 隣接する敵がいれば攻撃し、視界内の敵がいれば接近する。
// 移動しても敵に近づけない場合は諦めて次の優先度に進む
func (sp *squadPlanner) planAttackAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, bool) {
	nearestEnemy, nearestGrid, dist := sp.findNearestEnemy(world, entity, ctx)
	if nearestEnemy == nil {
		return nil, false
	}

	if dist == 1 {
		return &activity.AttackActivity{Target: *nearestEnemy}, true
	}

	return sp.tryMoveToward(world, entity, ctx.Grid, nearestGrid)
}

// planEvadeAction は回避ポリシーに基づくアクションを計画する。
// 視界内の最寄りの敵から距離を取る
func (sp *squadPlanner) planEvadeAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, bool) {
	nearestEnemy, _, _ := sp.findNearestEnemy(world, entity, ctx)
	if nearestEnemy == nil {
		return nil, false
	}

	enemyGrid := world.Components.GridElement.Get(*nearestEnemy)
	return sp.tryMoveAway(world, entity, ctx.Grid, enemyGrid)
}

// planPositionAction は位置ポリシーに基づくアクションを計画する
func (sp *squadPlanner) planPositionAction(world w.World, entity ecs.Entity, ctx *squadContext) activity.Behavior {
	switch ctx.Squad.Movement {
	case gc.SquadEscort:
		return sp.planEscortAction(world, entity, ctx)
	case gc.SquadVanguard:
		return sp.planVanguardAction(world, entity, ctx)
	case gc.SquadPatrol:
		return sp.planSquadPatrolAction(world, entity, ctx)
	case gc.SquadStationary:
		return waitAction("隊員待機")
	case gc.SquadRetreat:
		return sp.planEscortAction(world, entity, ctx)
	default:
		return waitAction("隊員デフォルト待機")
	}
}

// planEscortAction はリーダーから2マス以内にとどまるアクションを計画する
func (sp *squadPlanner) planEscortAction(world w.World, entity ecs.Entity, ctx *squadContext) activity.Behavior {
	dist := gridDistance(ctx.Grid, ctx.LeaderGrid)
	if dist <= escortMaxDistance {
		return waitAction("隊員護衛位置")
	}
	if b, ok := sp.tryMoveToward(world, entity, ctx.Grid, ctx.LeaderGrid); ok {
		return b
	}
	return waitAction("隊員護衛移動失敗")
}

// planVanguardAction はリーダーの前方に展開するアクションを計画する。
// リーダーから離れすぎている場合はリーダーに接近する
func (sp *squadPlanner) planVanguardAction(world w.World, entity ecs.Entity, ctx *squadContext) activity.Behavior {
	dist := gridDistance(ctx.Grid, ctx.LeaderGrid)
	if dist > vanguardMaxDistance {
		if b, ok := sp.tryMoveToward(world, entity, ctx.Grid, ctx.LeaderGrid); ok {
			return b
		}
		return waitAction("隊員前衛接近失敗")
	}
	if b, ok := sp.tryRandomMove(world, entity, ctx); ok {
		return b
	}
	return waitAction("隊員前衛移動失敗")
}

// planSquadPatrolAction は探索済みエリア内を自律的に巡回するアクションを計画する
func (sp *squadPlanner) planSquadPatrolAction(world w.World, entity ecs.Entity, ctx *squadContext) activity.Behavior {
	if b, ok := sp.tryRandomMove(world, entity, ctx); ok {
		return b
	}
	return waitAction("隊員巡回移動失敗")
}

// planItemPickupAction は拾得可能アイテムを拾うアクションを計画する。
// 足元にアイテムがあれば拾い、なければ視界内のアイテムに向かって移動する。
// PolicyIgnoreの場合は何もしない
func (sp *squadPlanner) planItemPickupAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, bool) {
	if ctx.Squad.ItemPickup == gc.PolicyIgnore {
		return nil, false
	}

	hasPickableHere := false
	var nearestItemGrid *gc.GridElement
	nearestDist := -1

	itemQuery := ecs.NewFilter2[gc.GridElement, gc.LocationOnField](world.ECS).Query()
	for itemQuery.Next() {
		item := itemQuery.Entity()
		if !query.IsPickable(item, world) {
			continue
		}
		grid := world.Components.GridElement.Get(item)

		if grid.X == ctx.Grid.X && grid.Y == ctx.Grid.Y {
			hasPickableHere = true
			continue
		}

		dist := gridDistance(ctx.Grid, grid)
		if dist > int(ctx.Squad.ViewDistance) {
			continue
		}
		if nearestDist < 0 || dist < nearestDist {
			nearestItemGrid = grid
			nearestDist = dist
		}
	}

	if hasPickableHere {
		sp.logger.Debug("隊員アイテム拾得", "entity", entity, "x", ctx.Grid.X, "y", ctx.Grid.Y)
		dest := *ctx.Grid
		return &activity.PickupActivity{Destination: &dest}, true
	}

	if nearestItemGrid != nil {
		sp.logger.Debug("隊員アイテムへ移動", "entity", entity, "dist", nearestDist)
		return sp.tryMoveToward(world, entity, ctx.Grid, nearestItemGrid)
	}

	return nil, false
}

// planItemHandlingAction はバックパック内のアイテムをポリシーに基づいて処理する。
// PolicyDistributeの場合はリーダーにアイテムを転送する
func (sp *squadPlanner) planItemHandlingAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, bool) {
	if ctx.Squad.ItemHandling != gc.PolicyDistribute {
		return nil, false
	}

	dist := gridDistance(ctx.Grid, ctx.LeaderGrid)
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
			e := item
			itemToTransfer = &e
		}
	}

	if itemToTransfer == nil {
		return nil, false
	}

	sp.logger.Debug("隊員アイテム転送", "entity", entity, "item", *itemToTransfer)
	return &activity.TransferActivity{Target: *itemToTransfer, Recipient: ctx.LeaderEntity}, true
}

// findNearestEnemy は視界内の最も近い敵を探す
func (sp *squadPlanner) findNearestEnemy(world w.World, entity ecs.Entity, ctx *squadContext) (*ecs.Entity, *gc.GridElement, int) {
	return query.FindNearestCharacter(world, entity, ctx.Grid, func(target ecs.Entity) bool {
		return query.FactionRelation(world, entity, target) == query.RelationHostile &&
			sp.visionSystem.CanSeeTarget(world, entity, target, ctx.Squad.ViewDistance)
	})
}

// tryMoveToward はBFSで壁を迂回した最短経路でターゲットに向かう移動を試みる
func (sp *squadPlanner) tryMoveToward(world w.World, entity ecs.Entity, from, target *gc.GridElement) (activity.Behavior, bool) {
	fromPos := from.Coord
	nextTile, ok := activity.FindNextStep(world, entity,
		from.Coord,
		target.Coord)
	if !ok {
		return nil, false
	}

	next := nextTile
	if !activity.CanMoveTo(world, next, fromPos, entity) {
		return nil, false
	}

	return moveAction(next), true
}

// tryMoveAway はターゲットから離れる移動を試みる
func (sp *squadPlanner) tryMoveAway(world w.World, entity ecs.Entity, from, threat *gc.GridElement) (activity.Behavior, bool) {
	dx := from.X - threat.X
	dy := from.Y - threat.Y

	candidates := calculateMoveCandidates(consts.Coord[consts.Tile]{X: dx, Y: dy})
	return tryMoveCandidates(world, entity, from, candidates)
}

// tryRandomMove は探索済みエリア内でランダム移動を試みる
func (sp *squadPlanner) tryRandomMove(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, bool) {
	dungeon := query.GetDungeon(world)
	from := ctx.Grid.Coord

	for _, d := range shuffledEightDirections(sp.rng) {
		dest := from.Add(d)

		if dungeon != nil && dungeon.ExploredTiles != nil {
			destGrid := gc.GridElement{Coord: dest}
			if !dungeon.ExploredTiles[destGrid] {
				continue
			}
		}

		if activity.CanMoveTo(world, dest, from, entity) {
			return moveAction(dest), true
		}
	}
	return nil, false
}
