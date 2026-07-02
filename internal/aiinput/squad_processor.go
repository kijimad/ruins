package aiinput

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// hpRetreatThreshold はHP割合がこの値以下のとき、ポリシーに関わらず後退する
const hpRetreatThreshold = 25

// SquadProcessor は隊員エンティティの行動処理を管理する。
// 敵AIのProcessorとは独立に動作し、ポリシーに基づいてアクションを決定する
type SquadProcessor struct {
	logger       *logger.Logger
	visionSystem VisionSystem
}

// NewSquadProcessor は新しいSquadProcessorを作成する
func NewSquadProcessor() *SquadProcessor {
	return &SquadProcessor{
		logger:       logger.New(logger.CategoryTurn),
		visionSystem: NewVisionSystem(),
	}
}

// ProcessSquadMembers は全ての隊員エンティティを処理する
func (sp *SquadProcessor) ProcessSquadMembers(world w.World) error {
	turnNumber := query.GetTurnState(world).TurnNumber
	sp.logger.Debug("隊員AI処理開始", "turn", turnNumber)

	entityCount := 0
	world.Manager.Join(
		world.Components.SquadMember,
		world.Components.AIMoveFSM,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entityCount++
		sp.processSquadMember(world, entity)
	}))

	sp.logger.Debug("隊員AI処理完了", "処理数", entityCount, "turn", turnNumber)
	return nil
}

// processSquadMember は個別の隊員エンティティを処理する
func (sp *SquadProcessor) processSquadMember(world w.World, entity ecs.Entity) {
	if entity.HasComponent(world.Components.Dead) {
		return
	}

	ctx, ok := sp.gathersquadContext(world, entity)
	if !ok {
		return
	}

	// enemy_processor.go と同じAPループ構造。統一プロセッサ移行時に共通化する
	activitiesExecuted := 0
	maxActivities := 10

	for activitiesExecuted < maxActivities {
		if entity.HasComponent(world.Components.Dead) {
			break
		}

		actorImpl, actionParams := sp.planAction(world, entity, ctx)
		if actorImpl == nil {
			break
		}

		actionCost := actorImpl.Info().ActionPointCost
		tbComp := world.Components.TurnBased.Get(entity)
		if tbComp == nil || tbComp.(*gc.TurnBased).AP.Current < actionCost {
			sp.logger.Debug("隊員AP不足", "entity", entity, "cost", actionCost)
			break
		}

		result, err := activity.Execute(actorImpl, actionParams, world)
		if err != nil {
			sp.logger.Warn("隊員アクション実行失敗", "entity", entity, "error", err.Error())
			break
		}

		sp.logger.Debug("隊員アクション実行", "entity", entity, "activity", actorImpl.Name(), "success", result.Success)
		activitiesExecuted++

		if !result.Success {
			break
		}
	}
}

// squadContext は隊員AIに必要な情報をまとめる
type squadContext struct {
	Grid         *gc.GridElement
	Vision       *gc.AIVision
	Policy       *gc.AIPolicy
	LeaderEntity ecs.Entity
	LeaderGrid   *gc.GridElement
}

// gathersquadContext は隊員の行動に必要なコンテキストを収集する
func (sp *SquadProcessor) gathersquadContext(world w.World, entity ecs.Entity) (*squadContext, bool) {
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	visionComp := world.Components.AIVision.Get(entity)
	if visionComp == nil {
		sp.logger.Warn("隊員にAIVisionがない", "entity", entity)
		return nil, false
	}

	si := query.GetSpatialIndex(world)
	if si == nil || si.PlayerEntity == nil {
		sp.logger.Warn("プレイヤーが見つからない", "entity", entity)
		return nil, false
	}
	leader := *si.PlayerEntity

	if !leader.HasComponent(world.Components.GridElement) {
		sp.logger.Warn("リーダーにGridElementがない", "entity", entity)
		return nil, false
	}

	defaultPolicy := gc.DefaultAIPolicy
	policy := &defaultPolicy
	if entity.HasComponent(world.Components.AIPolicy) {
		policy = world.Components.AIPolicy.Get(entity).(*gc.AIPolicy)
	}

	return &squadContext{
		Grid:         grid,
		Vision:       visionComp.(*gc.AIVision),
		Policy:       policy,
		LeaderEntity: leader,
		LeaderGrid:   world.Components.GridElement.Get(leader).(*gc.GridElement),
	}, true
}

// planAction はポリシーと状況に基づいてアクションを決定する。
// 優先順位: HP低下時後退 → エリア制限 → 戦闘 → アイテム転送 → アイテム拾得 → 位置ポリシー
func (sp *SquadProcessor) planAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams) {
	// HP低下時は後退する
	if sp.shouldRetreatLowHP(world, entity) {
		if b, p, ok := sp.planRetreatAction(world, entity, ctx); ok {
			return b, p
		}
	}

	// エリア制限: 探索済みエリア外なら最寄りの探索済みマスへ移動する
	if sp.isOutsideExploredArea(world, ctx.Grid) {
		if b, p, ok := sp.planReturnToExploredArea(world, entity, ctx); ok {
			return b, p
		}
	}

	// 戦闘ポリシーを評価する
	if b, p, ok := sp.planCombatAction(world, entity, ctx); ok {
		return b, p
	}

	// アイテム処理ポリシーを評価する。拾得より先に評価して、持っているアイテムを先に渡す
	if b, p, ok := sp.planItemHandlingAction(world, entity, ctx); ok {
		return b, p
	}

	// アイテム拾得ポリシーを評価する
	if b, p, ok := sp.planItemPickupAction(world, entity, ctx); ok {
		return b, p
	}

	// 位置ポリシーを評価する
	return sp.planPositionAction(world, entity, ctx)
}

// shouldRetreatLowHP はHP25%以下で後退すべきかを判定する
func (sp *SquadProcessor) shouldRetreatLowHP(world w.World, entity ecs.Entity) bool {
	hpComp := world.Components.HP.Get(entity)
	if hpComp == nil {
		return false
	}
	hp := hpComp.(*gc.HP)
	if hp.Max == 0 {
		return false
	}
	return hp.Current*100/hp.Max <= hpRetreatThreshold
}

// planRetreatAction はリーダーに向かって後退するアクションを計画する
func (sp *SquadProcessor) planRetreatAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams, bool) {
	sp.logger.Debug("隊員HP低下、後退", "entity", entity)
	return sp.tryMoveToward(world, entity, ctx.Grid, ctx.LeaderGrid)
}

// isOutsideExploredArea は現在位置が探索済みエリア外かを判定する
func (sp *SquadProcessor) isOutsideExploredArea(world w.World, grid *gc.GridElement) bool {
	dungeon := query.GetDungeon(world)
	if dungeon == nil || dungeon.ExploredTiles == nil {
		return false
	}
	return !dungeon.ExploredTiles[*grid]
}

// planReturnToExploredArea は最寄りの探索済みマスへ移動するアクションを計画する
func (sp *SquadProcessor) planReturnToExploredArea(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams, bool) {
	sp.logger.Debug("隊員がエリア外、リーダーに向かう", "entity", entity)
	return sp.tryMoveToward(world, entity, ctx.Grid, ctx.LeaderGrid)
}

// planCombatAction は戦闘ポリシーに基づくアクションを計画する
func (sp *SquadProcessor) planCombatAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams, bool) {
	switch ctx.Policy.CombatCurrent {
	case gc.CombatAttack:
		return sp.planAttackAction(world, entity, ctx)
	case gc.CombatEvade:
		return sp.planEvadeAction(world, entity, ctx)
	case gc.CombatIgnore:
		return nil, activity.ActionParams{}, false
	}
	return nil, activity.ActionParams{}, false
}

// planAttackAction は攻撃ポリシーに基づくアクションを計画する。
// 隣接する敵がいれば攻撃し、視界内の敵がいれば接近する。
// 移動しても敵に近づけない場合は諦めて次の優先度に進む
func (sp *SquadProcessor) planAttackAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams, bool) {
	nearestEnemy, nearestGrid, dist := sp.findNearestEnemy(world, entity, ctx)
	if nearestEnemy == nil {
		return nil, activity.ActionParams{}, false
	}

	// 隣接していれば攻撃する
	if dist <= 1 {
		target := *nearestEnemy
		return &activity.AttackActivity{}, activity.ActionParams{
			Actor:  entity,
			Target: &target,
		}, true
	}

	// 視界内の敵に接近する
	return sp.tryMoveToward(world, entity, ctx.Grid, nearestGrid)
}

// planEvadeAction は回避ポリシーに基づくアクションを計画する。
// 視界内の最寄りの敵から距離を取る
func (sp *SquadProcessor) planEvadeAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams, bool) {
	nearestEnemy, _, _ := sp.findNearestEnemy(world, entity, ctx)
	if nearestEnemy == nil {
		return nil, activity.ActionParams{}, false
	}

	enemyGrid := world.Components.GridElement.Get(*nearestEnemy).(*gc.GridElement)
	return sp.tryMoveAway(world, entity, ctx.Grid, enemyGrid)
}

// planPositionAction は位置ポリシーに基づくアクションを計画する
func (sp *SquadProcessor) planPositionAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams) {
	switch ctx.Policy.Movement {
	case gc.MovementEscort:
		return sp.planEscortAction(world, entity, ctx)
	case gc.MovementVanguard:
		return sp.planVanguardAction(world, entity, ctx)
	case gc.MovementPatrol:
		return sp.planPatrolAction(world, entity, ctx)
	case gc.MovementStationary:
		return waitAction(entity, "隊員待機")
	case gc.MovementRetreat:
		return sp.planEscortAction(world, entity, ctx)
	default:
		return waitAction(entity, "隊員デフォルト待機")
	}
}

// escortMaxDistance は護衛ポリシーでリーダーから離れてよい最大距離を表す
const escortMaxDistance = 2

// planEscortAction はリーダーから2マス以内にとどまるアクションを計画する
func (sp *SquadProcessor) planEscortAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams) {
	dist := gridDistance(ctx.Grid, ctx.LeaderGrid)
	if dist <= escortMaxDistance {
		return waitAction(entity, "隊員護衛位置")
	}
	if b, p, ok := sp.tryMoveToward(world, entity, ctx.Grid, ctx.LeaderGrid); ok {
		return b, p
	}
	return waitAction(entity, "隊員護衛移動失敗")
}

// vanguardMaxDistance は前衛ポリシーでリーダーから離れてよい最大距離を表す
const vanguardMaxDistance = 3

// planVanguardAction はリーダーの前方に展開するアクションを計画する。
// リーダーから離れすぎている場合はリーダーに接近する
func (sp *SquadProcessor) planVanguardAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams) {
	dist := gridDistance(ctx.Grid, ctx.LeaderGrid)
	if dist > vanguardMaxDistance {
		if b, p, ok := sp.tryMoveToward(world, entity, ctx.Grid, ctx.LeaderGrid); ok {
			return b, p
		}
		return waitAction(entity, "隊員前衛接近失敗")
	}
	// リーダーの近くにいる場合はランダムに移動する
	if b, p, ok := sp.tryRandomMove(world, entity, ctx); ok {
		return b, p
	}
	return waitAction(entity, "隊員前衛移動失敗")
}

// planPatrolAction は探索済みエリア内を自律的に巡回するアクションを計画する
func (sp *SquadProcessor) planPatrolAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams) {
	if b, p, ok := sp.tryRandomMove(world, entity, ctx); ok {
		return b, p
	}
	return waitAction(entity, "隊員巡回移動失敗")
}

// planItemPickupAction は拾得可能アイテムを拾うアクションを計画する。
// 足元にアイテムがあれば拾い、なければ視界内のアイテムに向かって移動する。
// PolicyIgnoreの場合は何もしない
func (sp *SquadProcessor) planItemPickupAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams, bool) {
	if ctx.Policy.ItemPickup == gc.PolicyIgnore {
		return nil, activity.ActionParams{}, false
	}

	// 足元のアイテムを探す。同時に視界内の最寄りアイテムも探す
	hasPickableHere := false
	var nearestItemGrid *gc.GridElement
	nearestDist := -1

	world.Manager.Join(
		world.Components.GridElement,
		world.Components.LocationOnField,
	).Visit(ecs.Visit(func(item ecs.Entity) {
		if !query.IsPickable(item, world) {
			return
		}
		grid := world.Components.GridElement.Get(item).(*gc.GridElement)

		// 足元チェック
		if grid.X == ctx.Grid.X && grid.Y == ctx.Grid.Y {
			hasPickableHere = true
			return
		}

		// 視界内かチェック
		dist := gridDistance(ctx.Grid, grid)
		if dist > int(ctx.Vision.ViewDistance) {
			return
		}
		if nearestDist < 0 || dist < nearestDist {
			nearestItemGrid = grid
			nearestDist = dist
		}
	}))

	// 足元にアイテムがあれば拾う
	if hasPickableHere {
		sp.logger.Debug("隊員アイテム拾得", "entity", entity, "x", ctx.Grid.X, "y", ctx.Grid.Y)
		dest := *ctx.Grid
		return &activity.PickupActivity{}, activity.ActionParams{
			Actor:       entity,
			Destination: &dest,
		}, true
	}

	// 視界内にアイテムがあれば向かう。距離が縮まらない場合は壁越しと判断して諦める
	if nearestItemGrid != nil {
		sp.logger.Debug("隊員アイテムへ移動", "entity", entity, "dist", nearestDist)
		return sp.tryMoveToward(world, entity, ctx.Grid, nearestItemGrid)
	}

	return nil, activity.ActionParams{}, false
}

// planItemHandlingAction はバックパック内のアイテムをポリシーに基づいて処理する。
// PolicyDistributeの場合はリーダーにアイテムを転送する
func (sp *SquadProcessor) planItemHandlingAction(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams, bool) {
	if ctx.Policy.ItemHandling != gc.PolicyDistribute {
		return nil, activity.ActionParams{}, false
	}

	// リーダーと隣接しているときだけアイテムを渡す
	dist := gridDistance(ctx.Grid, ctx.LeaderGrid)
	if dist > 1 {
		return nil, activity.ActionParams{}, false
	}

	// バックパック内にアイテムがあるか確認する
	var itemToTransfer *ecs.Entity
	world.Manager.Join(
		world.Components.LocationInBackpack,
	).Visit(ecs.Visit(func(item ecs.Entity) {
		if itemToTransfer != nil {
			return
		}
		loc := world.Components.LocationInBackpack.Get(item).(*gc.LocationInBackpack)
		if loc.Owner == entity {
			e := item
			itemToTransfer = &e
		}
	}))

	if itemToTransfer == nil {
		return nil, activity.ActionParams{}, false
	}

	sp.logger.Debug("隊員アイテム転送", "entity", entity, "item", *itemToTransfer)
	leader := ctx.LeaderEntity
	return &activity.TransferActivity{}, activity.ActionParams{
		Actor:     entity,
		Target:    itemToTransfer,
		Recipient: &leader,
	}, true
}

// findNearestEnemy は視界内の最も近い敵を探す
func (sp *SquadProcessor) findNearestEnemy(world w.World, entity ecs.Entity, ctx *squadContext) (*ecs.Entity, *gc.GridElement, int) {
	var nearestEntity *ecs.Entity
	var nearestGrid *gc.GridElement
	nearestDist := -1

	world.Manager.Join(
		world.Components.FactionEnemy,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(enemy ecs.Entity) {
		if enemy.HasComponent(world.Components.Dead) {
			return
		}
		if !sp.visionSystem.CanSeeTarget(world, entity, enemy, ctx.Vision) {
			return
		}
		enemyGrid := world.Components.GridElement.Get(enemy).(*gc.GridElement)
		dist := gridDistance(ctx.Grid, enemyGrid)
		if nearestDist < 0 || dist < nearestDist {
			e := enemy
			nearestEntity = &e
			nearestGrid = enemyGrid
			nearestDist = dist
		}
	}))

	return nearestEntity, nearestGrid, nearestDist
}

// tryMoveToward はBFSで壁を迂回した最短経路でターゲットに向かう移動を試みる
func (sp *SquadProcessor) tryMoveToward(world w.World, entity ecs.Entity, from, target *gc.GridElement) (activity.Behavior, activity.ActionParams, bool) {
	fromX, fromY := int(from.X), int(from.Y)
	goalX, goalY := int(target.X), int(target.Y)

	nextX, nextY, ok := activity.FindNextStep(world, entity, fromX, fromY, goalX, goalY)
	if !ok {
		return nil, activity.ActionParams{}, false
	}

	if !activity.CanMoveTo(world, nextX, nextY, fromX, fromY, entity) {
		return nil, activity.ActionParams{}, false
	}

	b, p := moveAction(entity, nextX, nextY)
	return b, p, true
}

// tryMoveAway はターゲットから離れる移動を試みる
func (sp *SquadProcessor) tryMoveAway(world w.World, entity ecs.Entity, from, threat *gc.GridElement) (activity.Behavior, activity.ActionParams, bool) {
	dx := int(from.X) - int(threat.X)
	dy := int(from.Y) - int(threat.Y)

	candidates := calculateMoveCandidates(dx, dy)
	return tryMoveCandidates(world, entity, from, candidates)
}

// tryRandomMove は探索済みエリア内でランダム移動を試みる
func (sp *SquadProcessor) tryRandomMove(world w.World, entity ecs.Entity, ctx *squadContext) (activity.Behavior, activity.ActionParams, bool) {
	dungeon := query.GetDungeon(world)
	fromX, fromY := int(ctx.Grid.X), int(ctx.Grid.Y)

	for _, d := range shuffledEightDirections() {
		destX := fromX + d.x
		destY := fromY + d.y

		// 探索済みエリア内のみ移動可能にする
		if dungeon != nil && dungeon.ExploredTiles != nil {
			destGrid := gc.GridElement{X: consts.Tile(destX), Y: consts.Tile(destY)}
			if !dungeon.ExploredTiles[destGrid] {
				continue
			}
		}

		if activity.CanMoveTo(world, destX, destY, fromX, fromY, entity) {
			b, p := moveAction(entity, destX, destY)
			return b, p, true
		}
	}
	return nil, activity.ActionParams{}, false
}

// gridDistance は2つのGridElement間のチェビシェフ距離を返す
func gridDistance(a, b *gc.GridElement) int {
	return geometry.ChebyshevDistance(int(a.X), int(a.Y), int(b.X), int(b.Y))
}

// calculateMoveCandidates はターゲットに向かう移動候補を計算する。
// DefaultActionPlannerと同じロジックを共有する
func calculateMoveCandidates(dx, dy int) []MoveCandidate {
	ap := &DefaultActionPlanner{}
	return ap.calculateMoveCandidates(dx, dy)
}

// tryMoveCandidates は移動候補を順に試行する
func tryMoveCandidates(world w.World, entity ecs.Entity, from *gc.GridElement, candidates []MoveCandidate) (activity.Behavior, activity.ActionParams, bool) {
	ap := &DefaultActionPlanner{}
	return ap.tryMoveCandidates(world, entity, from, candidates)
}
