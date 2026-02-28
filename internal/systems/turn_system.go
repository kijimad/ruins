package systems

import (
	"github.com/kijimaD/ruins/internal/activity"
	"github.com/kijimaD/ruins/internal/aiinput"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/turns"
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
	turnManager := world.Resources.TurnManager.(*turns.TurnManager)

	switch turnManager.TurnPhase {
	case turns.PlayerTurn:
		// プレイヤーが継続アクション中かチェック
		if processPlayerContinuousActivity(world, turnManager) {
			// 継続アクション中はターンを進める
			return nil
		}
		// APが最小行動コストを満たさない場合は自動でターンを終了
		if shouldAutoEndTurn(world) {
			turnManager.AdvanceToAITurn()
			return nil
		}
		// プレイヤー入力処理はDungeonStateで実行される
	case turns.AITurn:
		// AIターン: 全AI・NPCを一括処理
		if err := processAITurn(world); err != nil {
			return err
		}
		turnManager.AdvanceToTurnEnd()
	case turns.TurnEnd:
		// ターン終了処理
		if err := processTurnEnd(world); err != nil {
			return err
		}
		turnManager.StartNewTurn()
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
	// APがマイナスの場合は自動でターンを終了
	return ap.AP.Current < 0
}

// processPlayerContinuousActivity はプレイヤーの継続アクションを処理する
// 継続アクションが進行中の場合は true を返し、ターンを進める
func processPlayerContinuousActivity(world w.World, turnManager *turns.TurnManager) bool {
	// ActivityManager が存在しない場合は何もしない
	if world.Resources.ActivityManager == nil {
		return false
	}

	manager := world.Resources.ActivityManager.(*activity.Manager)

	// プレイヤーエンティティを取得
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return false
	}

	// プレイヤーの継続アクションをチェック
	if !manager.HasActivity(playerEntity) {
		return false
	}

	log := logger.New(logger.CategoryTurn)
	log.Debug("プレイヤー継続アクション処理")

	// 継続アクションの1ターン分を処理
	manager.ProcessTurn(world)

	// アクティビティが完了したかチェック
	if !manager.HasActivity(playerEntity) {
		log.Debug("プレイヤー継続アクション完了")
		// 完了した場合はターンコストを消費
		turnManager.ConsumeActionPoints(world, playerEntity, "継続アクション", 100)
		return true
	}

	// まだ進行中の場合もターンを進める
	turnManager.ConsumeActionPoints(world, playerEntity, "継続アクション", 100)
	return true
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
	logger := logger.New(logger.CategoryTurn)
	turnManager := world.Resources.TurnManager.(*turns.TurnManager)

	logger.Debug("ターン終了処理", "turn", turnManager.TurnNumber)

	// 全エンティティのアクションポイントを回復
	if err := turnManager.RestoreAllActionPoints(world); err != nil {
		return err
	}

	for _, updater := range []w.Updater{
		&DeadCleanupSystem{},
		&AutoInteractionSystem{},
		&TemperatureSystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return err
			}
		}
	}

	// TODO: ターン終了時の共通処理をここに追加
	// - エフェクトの持続時間減少
	// - 状態異常の更新
	// - 環境変化の処理
	// など
	return nil
}
