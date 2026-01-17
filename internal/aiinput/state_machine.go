package aiinput

import (
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
)

// StateMachine はAIの状態遷移ロジックを管理する
type StateMachine interface {
	UpdateState(roaming *gc.AIRoaming, canSeePlayer bool, currentTurn int)
}

// DefaultStateMachine は標準的な状態遷移実装
type DefaultStateMachine struct{}

// NewStateMachine は新しいStateMachineを作成する
func NewStateMachine() StateMachine {
	return &DefaultStateMachine{}
}

// UpdateState はAIの状態を更新する有限状態機械
func (sm *DefaultStateMachine) UpdateState(roaming *gc.AIRoaming, canSeePlayer bool, currentTurn int) {
	elapsedTurns := currentTurn - roaming.StartSubStateTurn

	// 現在の状態によって遷移ロジックを決定
	switch roaming.SubState {
	case gc.AIRoamingWaiting:
		sm.updateFromWaiting(roaming, canSeePlayer, elapsedTurns, currentTurn)

	case gc.AIRoamingDriving:
		sm.updateFromDriving(roaming, canSeePlayer, elapsedTurns, currentTurn)

	case gc.AIRoamingChasing:
		sm.updateFromChasing(roaming, canSeePlayer, elapsedTurns, currentTurn)

	default:
		// 不明な状態：待機状態に初期化
		sm.initializeToWaiting(roaming, currentTurn)
	}
}

// updateFromWaiting は待機状態からの遷移処理
func (sm *DefaultStateMachine) updateFromWaiting(roaming *gc.AIRoaming, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if canSeePlayer {
		// プレイヤー発見 → 追跡状態へ
		sm.transitionToChasing(roaming, currentTurn)
	} else if elapsedTurns >= roaming.DurationSubStateTurns {
		// 待機ターン終了 → 移動状態へ
		sm.transitionToDriving(roaming, currentTurn)
	}
}

// updateFromDriving は移動状態からの遷移処理
func (sm *DefaultStateMachine) updateFromDriving(roaming *gc.AIRoaming, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if canSeePlayer {
		// プレイヤー発見 → 追跡状態へ
		sm.transitionToChasing(roaming, currentTurn)
	} else if elapsedTurns >= roaming.DurationSubStateTurns {
		// 移動ターン終了 → 待機状態へ
		sm.transitionToWaiting(roaming, currentTurn)
	}
}

// updateFromChasing は追跡状態からの遷移処理
func (sm *DefaultStateMachine) updateFromChasing(roaming *gc.AIRoaming, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if !canSeePlayer {
		// プレイヤーを見失った場合
		if elapsedTurns >= 3 {
			// 3ターン以上見失った → 移動状態へ
			roaming.SubState = gc.AIRoamingDriving
			roaming.StartSubStateTurn = currentTurn
			roaming.DurationSubStateTurns = 5 + rand.IntN(5) // 5-9ターン移動
		}
		// 3ターン以内なら追跡継続
	} else if elapsedTurns >= roaming.DurationSubStateTurns {
		// 追跡ターン終了 → 待機状態へ
		sm.transitionToWaiting(roaming, currentTurn)
	} else {
		// プレイヤー視認中：追跡継続、ターンリセット
		roaming.StartSubStateTurn = currentTurn
	}
}

// transitionToWaiting は待機状態への遷移
func (sm *DefaultStateMachine) transitionToWaiting(roaming *gc.AIRoaming, currentTurn int) {
	roaming.SubState = gc.AIRoamingWaiting
	roaming.StartSubStateTurn = currentTurn
	roaming.DurationSubStateTurns = 2 + rand.IntN(4) // 2-5ターン待機
}

// transitionToDriving は移動状態への遷移
func (sm *DefaultStateMachine) transitionToDriving(roaming *gc.AIRoaming, currentTurn int) {
	roaming.SubState = gc.AIRoamingDriving
	roaming.StartSubStateTurn = currentTurn
	roaming.DurationSubStateTurns = 3 + rand.IntN(7) // 3-9ターン移動
}

// transitionToChasing は追跡状態への遷移
func (sm *DefaultStateMachine) transitionToChasing(roaming *gc.AIRoaming, currentTurn int) {
	roaming.SubState = gc.AIRoamingChasing
	roaming.StartSubStateTurn = currentTurn
	roaming.DurationSubStateTurns = 10 + rand.IntN(5) // 10-14ターン追跡
}

// initializeToWaiting は待機状態への初期化
func (sm *DefaultStateMachine) initializeToWaiting(roaming *gc.AIRoaming, currentTurn int) {
	roaming.SubState = gc.AIRoamingWaiting
	roaming.StartSubStateTurn = currentTurn
	roaming.DurationSubStateTurns = 2 + rand.IntN(3) // 2-4ターン待機
}
