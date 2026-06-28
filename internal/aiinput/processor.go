package aiinput

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// Processor はAIエンティティの行動処理を管理する
type Processor struct {
	logger        *logger.Logger
	stateMachine  StateMachine
	actionPlanner ActionPlanner
	visionSystem  VisionSystem
}

// NewProcessor は新しいProcessorを作成する
func NewProcessor() *Processor {
	return &Processor{
		logger:        logger.New(logger.CategoryTurn),
		stateMachine:  NewStateMachine(),
		actionPlanner: NewActionPlanner(),
		visionSystem:  NewVisionSystem(),
	}
}

// ProcessAllEntities は全てのAIエンティティを処理する
func (p *Processor) ProcessAllEntities(world w.World) error {
	turnNumber := query.GetTurnState(world).TurnNumber
	p.logger.Debug("AI処理開始", "turn", turnNumber)

	entityCount := 0

	// AIMoveFSMコンポーネントを持つ全エンティティを処理する。隊員は専用のSquadProcessorで処理するため除外する
	world.Manager.Join(
		world.Components.AIMoveFSM,
		world.Components.GridElement,
		world.Components.SquadMember.Not(),
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entityCount++
		p.logger.Debug("AIエンティティを処理中", "entity", entity, "count", entityCount)
		p.ProcessEntity(world, entity)
	}))

	p.logger.Debug("AI処理完了", "処理されたエンティティ数", entityCount, "turn", turnNumber)
	return nil
}

// ProcessEntity は個別のAIエンティティを処理する
func (p *Processor) ProcessEntity(world w.World, entity ecs.Entity) {
	turnNumber := query.GetTurnState(world).TurnNumber
	p.logger.Debug("AIエンティティ処理開始", "entity", entity)

	// 死亡しているエンティティは処理しない
	if entity.HasComponent(world.Components.Dead) {
		p.logger.Debug("Deadエンティティのため処理スキップ", "entity", entity)
		return
	}

	// 必要なコンポーネントを取得
	context, err := p.gatherEntityContext(world, entity)
	if err != nil {
		p.logger.Warn("AIエンティティコンテキスト取得失敗", "entity", entity, "error", err.Error())
		return
	}

	// プレイヤー検索
	playerEntity := p.findPlayer(world)
	if playerEntity == nil {
		return
	}

	// プレイヤーエンティティの有効性チェック
	if !playerEntity.HasComponent(world.Components.GridElement) {
		p.logger.Warn("プレイヤーエンティティが無効（GridElementなし）", "entity", entity, "player", *playerEntity)
		return
	}

	// 視界チェック
	canSeePlayer := p.visionSystem.CanSeeTarget(world, entity, *playerEntity, context.Vision)
	p.logger.Debug("プレイヤー視界チェック", "entity", entity, "canSee", canSeePlayer)

	// 状態更新
	oldState := context.Roaming.SubState
	p.stateMachine.UpdateState(context.Roaming, context.Disposition, canSeePlayer, turnNumber)
	if oldState != context.Roaming.SubState {
		p.logger.Debug("AI状態変化", "entity", entity, "from", oldState, "to", context.Roaming.SubState)
	}

	// 残りターン数を計算してログ出力
	elapsedTurns := turnNumber - context.Roaming.StartSubStateTurn
	remainingTurns := context.Roaming.DurationSubStateTurns - elapsedTurns
	if remainingTurns < 0 {
		remainingTurns = 0
	}
	p.logger.Debug("AIRoaming状態", "entity", entity, "subState", context.Roaming.SubState, "remainingTurns", remainingTurns)

	// APが残っている限り連続してアクティビティを実行
	activitiesExecuted := 0
	maxActivities := 10 // 無限ループを防ぐためのリミット

	for activitiesExecuted < maxActivities {
		// アクション実行中に死亡した場合は処理を中断
		if entity.HasComponent(world.Components.Dead) {
			p.logger.Debug("エンティティが死亡したため処理中断", "entity", entity)
			break
		}

		// アクション決定
		actorImpl, actionParams := p.actionPlanner.PlanAction(world, entity, *playerEntity, context)

		// アクション実行
		if actorImpl == nil {
			p.logger.Debug("アクション無し", "entity", entity)
			break
		}
		activityName := actorImpl.Name()
		p.logger.Debug("アクティビティ決定", "entity", entity, "activity", activityName, "state", context.Roaming.SubState, "count", activitiesExecuted)

		// APが足りるか確認する
		actionCost := actorImpl.Info().ActionPointCost
		tbComp := world.Components.TurnBased.Get(entity)
		if tbComp == nil || tbComp.(*gc.TurnBased).AP.Current < 0 {
			p.logger.Debug("AP不足でアクション実行不可", "entity", entity, "activity", activityName, "cost", actionCost)
			break
		}

		result, err := activity.Execute(actorImpl, actionParams, world)
		if err != nil {
			p.logger.Warn("AIアクション実行失敗", "entity", entity, "activity", activityName, "error", err.Error())
			break
		}

		p.logger.Debug("AIアクティビティ実行成功", "entity", entity, "activity", activityName, "success", result.Success, "state", context.Roaming.SubState, "message", result.Message)
		activitiesExecuted++

		// アクション失敗時は停止
		if !result.Success {
			p.logger.Debug("アクション失敗により停止", "entity", entity, "activity", activityName)
			break
		}
	}

	p.logger.Debug("AIエンティティ処理完了", "entity", entity, "実行されたアクティビティ数", activitiesExecuted)
}

// EntityContext はAIエンティティの必要な情報をまとめる
type EntityContext struct {
	GridElement     *gc.GridElement
	Vision          *gc.AIVision
	Roaming         *gc.AIRoaming
	Disposition     *gc.Disposition
	MovementPattern gc.MovementPattern
}

// gatherEntityContext はエンティティから必要なコンポーネントを収集する
func (p *Processor) gatherEntityContext(world w.World, entity ecs.Entity) (*EntityContext, error) {
	// GridElementコンポーネント取得
	gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
	p.logger.Debug("AIエンティティ位置", "entity", entity, "x", gridElement.X, "y", gridElement.Y)

	// AIVisionコンポーネント確認
	aiVision := world.Components.AIVision.Get(entity)
	if aiVision == nil {
		return nil, &AIError{Type: "component_missing", Message: "AIVisionコンポーネントなし", Entity: &entity}
	}
	vision := aiVision.(*gc.AIVision)
	p.logger.Debug("AIVision設定", "entity", entity, "viewDistance", vision.ViewDistance)

	// AIRoamingコンポーネント確認
	aiRoaming := world.Components.AIRoaming.Get(entity)
	if aiRoaming == nil {
		return nil, &AIError{Type: "component_missing", Message: "AIRoamingコンポーネントなし", Entity: &entity}
	}
	roaming := aiRoaming.(*gc.AIRoaming)

	// Dispositionコンポーネント取得
	var disposition *gc.Disposition
	if d := world.Components.Disposition.Get(entity); d != nil {
		disposition = d.(*gc.Disposition)
	}

	// MovementPatternコンポーネント取得
	movementPattern := gc.MovementRandom
	if mp := world.Components.MovementPattern.Get(entity); mp != nil {
		movementPattern = *mp.(*gc.MovementPattern)
	}

	return &EntityContext{
		GridElement:     gridElement,
		Vision:          vision,
		Roaming:         roaming,
		Disposition:     disposition,
		MovementPattern: movementPattern,
	}, nil
}

// findPlayer はプレイヤーエンティティを探す
func (p *Processor) findPlayer(world w.World) *ecs.Entity {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return nil
	}
	return si.PlayerEntity
}
