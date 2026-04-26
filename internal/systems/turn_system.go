package systems

import (
	"github.com/kijimaD/ruins/internal/activity"
	"github.com/kijimaD/ruins/internal/aiinput"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

// TurnSystem はターン管理を行うシステム
type TurnSystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys TurnSystem) String() string {
	return "TurnSystem"
}

// Update はターン管理を行う
// w.Updater interfaceを実装
func (sys *TurnSystem) Update(world w.World) error {
	turnState, err := worldhelper.GetTurnState(world)
	if err != nil {
		return err
	}

	switch turnState.Phase {
	case gc.TurnPhasePlayer:
		// プレイヤーが継続アクション中かチェック
		if processPlayerContinuousActivity(world) {
			// 継続アクション中はターンを進める
			return nil
		}
		// APが最小行動コストを満たさない場合は自動でターンを終了
		if shouldAutoEndTurn(world) {
			turnState.Phase = gc.TurnPhaseAI
			return nil
		}
		// プレイヤー入力処理はDungeonStateで実行される
	case gc.TurnPhaseAI:
		// AIターン: 全AI・NPCを一括処理
		if err := processAITurn(world); err != nil {
			return err
		}
		turnState.Phase = gc.TurnPhaseEnd
	case gc.TurnPhaseEnd:
		// ターン終了処理
		if err := processTurnEnd(world); err != nil {
			return err
		}
		turnState.TurnNumber++
		turnState.Phase = gc.TurnPhasePlayer
	}
	return nil
}

// shouldAutoEndTurn はプレイヤーのAPがマイナスの場合にtrueを返す
// APがマイナスの間は自動でターンを経過させる
func shouldAutoEndTurn(world w.World) bool {
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return false
	}

	turnBased := world.Components.TurnBased.Get(playerEntity)
	if turnBased == nil {
		return false
	}

	ap := turnBased.(*gc.TurnBased)
	// APが最小閾値未満の場合は自動でターンを終了
	return ap.AP.Current < consts.MinActionThreshold
}

// processAITurn はAIターンの処理を行う
func processAITurn(world w.World) error {
	logger := logger.New(logger.CategoryTurn)
	logger.Debug("AIターン処理開始")

	// AI・NPCエンティティを処理
	processor := aiinput.NewProcessor()
	if err := processor.ProcessAllEntities(world); err != nil {
		return err
	}

	logger.Debug("AIターン処理完了")
	return nil
}

// processTurnEnd はターン終了処理を行う
func processTurnEnd(world w.World) error {
	log := logger.New(logger.CategoryTurn)
	turnState, err := worldhelper.GetTurnState(world)
	if err != nil {
		return err
	}

	log.Debug("ターン終了処理", "turn", turnState.TurnNumber)

	// 全エンティティのアクションポイントを回復
	if err := worldhelper.RestoreAllActionPoints(world); err != nil {
		return err
	}

	return runTurnEndSystems(world)
}

// runTurnEndSystems はターン終了時に実行するシステム群を呼び出す
func runTurnEndSystems(world w.World) error {
	for _, updater := range []w.Updater{
		&AutoInteractionSystem{},
		&TemperatureSystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return err
			}
		}
	}

	return nil
}

// processPlayerContinuousActivity はプレイヤーの継続アクションを処理する
// 継続アクションが進行中の場合は true を返し、ターンを進める
func processPlayerContinuousActivity(world w.World) bool {
	// プレイヤーエンティティを取得
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return false
	}

	// プレイヤーの継続アクションをチェック
	if !worldhelper.HasActivity(world, playerEntity) {
		return false
	}

	log := logger.New(logger.CategoryTurn)
	log.Debug("プレイヤー継続アクション処理")

	// 継続アクションの1ターン分を処理
	activity.ProcessTurn(world)

	if !worldhelper.HasActivity(world, playerEntity) {
		log.Debug("プレイヤー継続アクション完了")
	}

	// 継続中でも完了でもターンコストを消費する
	worldhelper.ConsumeActionPoints(world, playerEntity, consts.StandardActionCost)
	return true
}
