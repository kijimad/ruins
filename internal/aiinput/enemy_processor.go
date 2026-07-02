package aiinput

import (
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// Processor はAIエンティティの行動処理を管理する。
// roamingPlannerとsquadPlannerを使い分けて全AIエンティティを統一的に処理する
type Processor struct {
	logger         *logger.Logger
	stateMachine   StateMachine
	visionSystem   VisionSystem
	roamingPlanner *roamingPlanner
	squadPlanner   *squadPlanner
}

// NewProcessor は新しいProcessorを作成する
func NewProcessor() *Processor {
	return &Processor{
		logger:         logger.New(logger.CategoryTurn),
		stateMachine:   NewStateMachine(),
		visionSystem:   NewVisionSystem(),
		roamingPlanner: newRoamingPlanner(),
		squadPlanner:   newSquadPlanner(),
	}
}

// ProcessAll は全AIエンティティを処理する。
// 敵・中立NPCの後に隊員を処理し、敵の移動結果を反映した判断を可能にする
func (p *Processor) ProcessAll(world w.World) error {
	if err := p.ProcessNonSquadAI(world); err != nil {
		return err
	}
	return p.ProcessSquadAI(world)
}

// ProcessNonSquadAI はAIMoveFSMを持つ非隊員エンティティを処理する。
// 敵・中立NPC問わず、AIPolicyと状態機械で行動を分岐する
func (p *Processor) ProcessNonSquadAI(world w.World) error {
	turnNumber := query.GetTurnState(world).TurnNumber
	p.logger.Debug("AI処理開始", "turn", turnNumber)

	entityCount := 0
	world.Manager.Join(
		world.Components.AIMoveFSM,
		world.Components.GridElement,
		world.Components.SquadMember.Not(),
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entityCount++
		if entity.HasComponent(world.Components.Dead) {
			p.logger.Debug("Deadエンティティのため処理スキップ", "entity", entity)
			return
		}

		p.prepareRoamingEntity(world, entity, turnNumber)
		runAPLoop(world, entity, p.roamingPlanner, p.logger)
	}))

	p.logger.Debug("AI処理完了", "処理されたエンティティ数", entityCount, "turn", turnNumber)
	return nil
}

// ProcessSquadAI は全ての隊員エンティティを処理する
func (p *Processor) ProcessSquadAI(world w.World) error {
	turnNumber := query.GetTurnState(world).TurnNumber
	p.logger.Debug("隊員AI処理開始", "turn", turnNumber)

	entityCount := 0
	world.Manager.Join(
		world.Components.SquadMember,
		world.Components.AIMoveFSM,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entityCount++
		if entity.HasComponent(world.Components.Dead) {
			return
		}
		runAPLoop(world, entity, p.squadPlanner, p.logger)
	}))

	p.logger.Debug("隊員AI処理完了", "処理数", entityCount, "turn", turnNumber)
	return nil
}

// prepareRoamingEntity はAPループ前に状態遷移を更新する。
// 1エンティティにつき1ターンに1回だけ呼ばれる
func (p *Processor) prepareRoamingEntity(world w.World, entity ecs.Entity, turnNumber int) {
	context, err := gatherEntityContext(world, entity)
	if err != nil {
		p.logger.Warn("AIエンティティコンテキスト取得失敗", "entity", entity, "error", err.Error())
		return
	}

	playerEntity := findPlayer(world)
	if playerEntity == nil {
		return
	}
	if !playerEntity.HasComponent(world.Components.GridElement) {
		p.logger.Warn("プレイヤーエンティティが無効（GridElementなし）", "entity", entity)
		return
	}

	canSeePlayer := p.visionSystem.CanSeeTarget(world, entity, *playerEntity, context.Vision)

	oldState := context.State.SubState
	p.stateMachine.UpdateState(context.State, context.Policy, canSeePlayer, turnNumber)
	if oldState != context.State.SubState {
		p.logger.Debug("AI状態変化", "entity", entity, "from", oldState, "to", context.State.SubState)
	}
}
