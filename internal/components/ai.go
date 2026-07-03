package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// AIStateSubState はAI行動のサブ状態を表す
type AIStateSubState string

const (
	// AIStateWaiting は待機状態
	AIStateWaiting = AIStateSubState("WAIT")
	// AIStateDriving は移動状態
	AIStateDriving = AIStateSubState("DRIVING")
	// AIStateChasing は追跡状態
	AIStateChasing = AIStateSubState("CHASING")
	// AIStateFleeing は逃亡状態
	AIStateFleeing = AIStateSubState("FLEEING")
)

// AI はAIエンティティの全状態を保持する統合コンポーネント。
// 行動方針、ランタイム状態、視覚情報を1つの構造体で表現する
type AI struct {
	// 行動方針 ================
	// Planner は適用する行動計画の種別。スポーン時のバリデーションとセーブに使用する
	Planner PlannerType
	// CombatDefault は初期方針を保持する。逃亡後にこの値へ復帰する
	CombatDefault CombatPolicy
	// CombatCurrent は現在の方針を保持する。被ダメージなどで変化する
	CombatCurrent CombatPolicy
	// Movement は非戦闘時の移動方針を保持する
	Movement MovementPolicy
	// ItemPickup はアイテム拾得方針。隊員のみが使用する
	ItemPickup ItemPickupPolicy
	// ItemHandling はアイテム処理方針。隊員のみが使用する
	ItemHandling ItemHandlingPolicy

	// ランタイム状態 ================
	SubState AIStateSubState
	// StartSubStateTurn はサブステートの開始ターンを保持する
	StartSubStateTurn int
	// DurationSubStateTurns はサブステートの持続ターン数を保持する
	DurationSubStateTurns int
	// SpawnX, SpawnY はスポーン地点。Territorial移動の範囲基準に使う
	SpawnX, SpawnY int
	// PatrolDirX, PatrolDirY は巡回方向。Patrol移動で現在の進行方向を保持する
	PatrolDirX, PatrolDirY int

	// 視覚 ================
	// ViewDistance は視界距離（タイル単位）
	ViewDistance consts.Tile
	// TargetEntity は追跡対象のエンティティ
	TargetEntity *ecs.Entity
}

// ReactToHostile は被ダメージ時に戦闘方針を変化させる。
// CombatIgnore は反撃のため CombatAttack に遷移する
func (ai *AI) ReactToHostile() {
	switch ai.CombatDefault {
	case CombatIgnore:
		ai.CombatCurrent = CombatAttack
	case CombatAttack, CombatEvade:
		// 既に戦闘的または回避中なので変化なし
	}
}

// ResetCombat は戦闘方針をデフォルトに復帰させる
func (ai *AI) ResetCombat() {
	ai.CombatCurrent = ai.CombatDefault
}

// DefaultSquadAI は隊員のデフォルトAIを返す。
// SpawnX/Y 等のランタイム値は呼び出し元で設定する
func DefaultSquadAI() AI {
	return AI{
		Planner:       PlannerSquad,
		CombatDefault: CombatAttack,
		CombatCurrent: CombatAttack,
		Movement:      MovementEscort,
		ItemPickup:    PolicyPickup,
		ItemHandling:  PolicyDistribute,
		SubState:      AIStateWaiting,
		ViewDistance:  5,
	}
}
