package aiinput

import (
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
)

// StateMachine はAIの状態遷移ロジックを管理する
type StateMachine interface {
	UpdateState(state *gc.AIState, policy *gc.AIPolicy, canSeePlayer bool, currentTurn int)
}

// DefaultStateMachine は標準的な状態遷移実装
type DefaultStateMachine struct{}

// NewStateMachine は新しいStateMachineを作成する
func NewStateMachine() StateMachine {
	return &DefaultStateMachine{}
}

// UpdateState はAIの状態を更新する有限状態機械
func (sm *DefaultStateMachine) UpdateState(state *gc.AIState, policy *gc.AIPolicy, canSeePlayer bool, currentTurn int) {
	elapsedTurns := currentTurn - state.StartSubStateTurn

	// 現在の状態によって遷移ロジックを決定
	switch state.SubState {
	case gc.AIStateWaiting:
		sm.updateFromWaiting(state, policy, canSeePlayer, elapsedTurns, currentTurn)

	case gc.AIStateDriving:
		sm.updateFromDriving(state, policy, canSeePlayer, elapsedTurns, currentTurn)

	case gc.AIStateChasing:
		sm.updateFromChasing(state, canSeePlayer, elapsedTurns, currentTurn)

	case gc.AIStateFleeing:
		sm.updateFromFleeing(state, policy, canSeePlayer, elapsedTurns, currentTurn)

	default:
		// 不明な状態：待機状態に初期化
		sm.initializeToWaiting(state, currentTurn)
	}
}

// shouldChase はAIPolicyに基づいて追跡すべきかを判定する
func shouldChase(policy *gc.AIPolicy) bool {
	if policy == nil {
		return true // AIPolicyがない場合は既存動作を維持する
	}
	return policy.CombatCurrent == gc.CombatAttack
}

// shouldFlee はAIPolicyに基づいて逃亡すべきかを判定する
func shouldFlee(policy *gc.AIPolicy) bool {
	if policy == nil {
		return false
	}
	return policy.CombatCurrent == gc.CombatEvade
}

// updateFromWaiting は待機状態からの遷移処理
func (sm *DefaultStateMachine) updateFromWaiting(state *gc.AIState, policy *gc.AIPolicy, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if canSeePlayer {
		if shouldFlee(policy) {
			sm.transitionToFleeing(state, currentTurn)
		} else if shouldChase(policy) {
			sm.transitionToChasing(state, currentTurn)
		}
		// Neutral: プレイヤーを見ても何もしない
	} else if elapsedTurns >= state.DurationSubStateTurns {
		sm.transitionToDriving(state, currentTurn)
	}
}

// updateFromDriving は移動状態からの遷移処理
func (sm *DefaultStateMachine) updateFromDriving(state *gc.AIState, policy *gc.AIPolicy, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if canSeePlayer {
		if shouldFlee(policy) {
			sm.transitionToFleeing(state, currentTurn)
		} else if shouldChase(policy) {
			sm.transitionToChasing(state, currentTurn)
		}
	} else if elapsedTurns >= state.DurationSubStateTurns {
		sm.transitionToWaiting(state, currentTurn)
	}
}

// updateFromChasing は追跡状態からの遷移処理
func (sm *DefaultStateMachine) updateFromChasing(state *gc.AIState, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if !canSeePlayer {
		// プレイヤーを見失った場合
		if elapsedTurns >= 3 {
			// 3ターン以上見失った → 移動状態へ
			state.SubState = gc.AIStateDriving
			state.StartSubStateTurn = currentTurn
			state.DurationSubStateTurns = 5 + rand.IntN(5) // 5-9ターン移動
		}
		// 3ターン以内なら追跡継続
	} else if elapsedTurns >= state.DurationSubStateTurns {
		// 追跡ターン終了 → 待機状態へ
		sm.transitionToWaiting(state, currentTurn)
	} else {
		// プレイヤー視認中：追跡継続、ターンリセット
		state.StartSubStateTurn = currentTurn
	}
}

// updateFromFleeing は逃亡状態からの遷移処理
func (sm *DefaultStateMachine) updateFromFleeing(state *gc.AIState, policy *gc.AIPolicy, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if !canSeePlayer && elapsedTurns >= state.DurationSubStateTurns {
		// プレイヤーを見失い、逃亡時間が終了 → デフォルト態度に復帰して移動へ
		if policy != nil {
			policy.ResetCombat()
		}
		sm.transitionToDriving(state, currentTurn)
	} else if canSeePlayer {
		// プレイヤーが見えている間は逃亡継続、ターンリセット
		state.StartSubStateTurn = currentTurn
	}
}

// transitionToWaiting は待機状態への遷移
func (sm *DefaultStateMachine) transitionToWaiting(state *gc.AIState, currentTurn int) {
	state.SubState = gc.AIStateWaiting
	state.StartSubStateTurn = currentTurn
	state.DurationSubStateTurns = 2 + rand.IntN(4) // 2-5ターン待機
}

// transitionToDriving は移動状態への遷移
func (sm *DefaultStateMachine) transitionToDriving(state *gc.AIState, currentTurn int) {
	state.SubState = gc.AIStateDriving
	state.StartSubStateTurn = currentTurn
	state.DurationSubStateTurns = 3 + rand.IntN(7) // 3-9ターン移動
}

// transitionToChasing は追跡状態への遷移
func (sm *DefaultStateMachine) transitionToChasing(state *gc.AIState, currentTurn int) {
	state.SubState = gc.AIStateChasing
	state.StartSubStateTurn = currentTurn
	state.DurationSubStateTurns = 10 + rand.IntN(5) // 10-14ターン追跡
}

// transitionToFleeing は逃亡状態への遷移
func (sm *DefaultStateMachine) transitionToFleeing(state *gc.AIState, currentTurn int) {
	state.SubState = gc.AIStateFleeing
	state.StartSubStateTurn = currentTurn
	state.DurationSubStateTurns = 5 + rand.IntN(5) // 5-9ターン逃亡
}

// initializeToWaiting は待機状態への初期化
func (sm *DefaultStateMachine) initializeToWaiting(state *gc.AIState, currentTurn int) {
	state.SubState = gc.AIStateWaiting
	state.StartSubStateTurn = currentTurn
	state.DurationSubStateTurns = 2 + rand.IntN(3) // 2-4ターン待機
}
