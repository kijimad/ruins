package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetTurnState はワールドからターン状態シングルトンを取得する
// シングルトンが0個または2個以上の場合はエラーを返す
func GetTurnState(world w.World) (*gc.TurnState, error) {
	var states []*gc.TurnState
	world.Manager.Join(world.Components.TurnState).Visit(ecs.Visit(func(entity ecs.Entity) {
		states = append(states, world.Components.TurnState.Get(entity).(*gc.TurnState))
	}))

	if len(states) == 0 {
		return nil, fmt.Errorf("TurnStateシングルトンが存在しません")
	}
	if len(states) > 1 {
		return nil, fmt.Errorf("TurnStateシングルトンが複数存在します: %d個", len(states))
	}

	return states[0], nil
}

// GetTurnNumber はシングルトンからターン番号を取得する
func GetTurnNumber(world w.World) int {
	state, err := GetTurnState(world)
	if err != nil {
		return 0
	}
	return state.TurnNumber
}

// CanPlayerAct はプレイヤーが行動可能かを判定する
// プレイヤーターンかつAP >= 0 の場合にtrueを返す
func CanPlayerAct(world w.World) bool {
	// プレイヤーターンでなければ行動不可
	turnState, err := GetTurnState(world)
	if err != nil || turnState.Phase != gc.TurnPhasePlayer {
		return false
	}

	// プレイヤーのAPをチェック
	playerEntity, err := GetPlayerEntity(world)
	if err != nil {
		return false
	}

	tbComp := world.Components.TurnBased.Get(playerEntity)
	if tbComp == nil {
		return false
	}

	tb := tbComp.(*gc.TurnBased)
	return tb.AP.Current >= 0
}

// ConsumeActionPoints はエンティティのアクションポイントを消費する
func ConsumeActionPoints(world w.World, entity ecs.Entity, cost int) bool {
	tbComp := world.Components.TurnBased.Get(entity)
	if tbComp == nil {
		return false
	}

	tb := tbComp.(*gc.TurnBased)
	tb.AP.Current -= cost

	log := logger.New(logger.CategoryTurn)
	log.Debug("アクションポイント消費",
		"entity", entity,
		"cost", cost,
		"remaining", tb.AP.Current)

	return true
}

// RestoreAllActionPoints は全エンティティのAPを回復する
func RestoreAllActionPoints(world w.World) error {
	log := logger.New(logger.CategoryTurn)
	var err error

	world.Manager.Join(world.Components.TurnBased).Visit(ecs.Visit(func(entity ecs.Entity) {
		tb := world.Components.TurnBased.Get(entity).(*gc.TurnBased)

		// MaxAPとSpeedを計算
		maxAP, calcErr := CalculateMaxActionPoints(world, entity)
		if calcErr != nil {
			err = calcErr
			return
		}

		speed := CalculateSpeed(world, entity)
		tb.Speed = speed
		tb.AP.Max = maxAP

		// 現在AP + Speed で上限まで回復
		newAP := tb.AP.Current + speed
		if newAP > maxAP {
			newAP = maxAP
		}
		tb.AP.Current = newAP

		log.Debug("アクションポイント回復",
			"entity", entity,
			"speed", speed,
			"current", tb.AP.Current,
			"max", maxAP)
	}))

	return err
}
