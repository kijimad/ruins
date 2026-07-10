package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/mlange-42/ark/ecs"
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

// SoloAI と SquadAI は独立したコンポーネント（プレーンデータ）。
// 以前は AI{Planner PlannerConfig} でインターフェース多態にしていたが、
// ECS のデータ指向（コンポーネントは振る舞いを持たないデータ）と serde 互換のため
// 別コンポーネントに分割した。エンティティは片方だけを持つ。

// SoloAI は単独行動NPC用の設定と状態を保持する
type SoloAI struct {
	CombatDefault CombatPolicy
	CombatCurrent CombatPolicy
	Movement      SoloMovement
	ViewDistance  consts.Tile

	SubState               AIStateSubState
	StartSubStateTurn      int
	DurationSubStateTurns  int
	OriginX, OriginY       int
	PatrolDirX, PatrolDirY int
	TargetEntity           *ecs.Entity
}

// Type はPlannerSoloを返す
func (s *SoloAI) Type() PlannerType { return PlannerSolo }

// ReactToHostile は被ダメージ時に戦闘方針を変化させる。
// CombatIgnore は反撃のため CombatAttack に遷移する
func (s *SoloAI) ReactToHostile() {
	switch s.CombatDefault {
	case CombatIgnore:
		s.CombatCurrent = CombatAttack
	case CombatAttack, CombatEvade:
	}
}

// ResetCombat は戦闘方針をデフォルトに復帰させる
func (s *SoloAI) ResetCombat() {
	s.CombatCurrent = s.CombatDefault
}

// SquadAI は隊員用の設定を保持する
type SquadAI struct {
	CombatDefault CombatPolicy
	CombatCurrent CombatPolicy
	Movement      SquadMovement
	ViewDistance  consts.Tile
	ItemPickup    ItemPickupPolicy
	ItemHandling  ItemHandlingPolicy
}

// Type はPlannerSquadを返す
func (s *SquadAI) Type() PlannerType { return PlannerSquad }

// ReactToHostile は被ダメージ時に戦闘方針を変化させる
func (s *SquadAI) ReactToHostile() {
	switch s.CombatDefault {
	case CombatIgnore:
		s.CombatCurrent = CombatAttack
	case CombatAttack, CombatEvade:
	}
}

// ResetCombat は戦闘方針をデフォルトに復帰させる
func (s *SquadAI) ResetCombat() {
	s.CombatCurrent = s.CombatDefault
}

// DefaultSquadAI は隊員のデフォルトSquadAIを返す
func DefaultSquadAI() SquadAI {
	return SquadAI{
		CombatDefault: CombatAttack,
		CombatCurrent: CombatAttack,
		Movement:      SquadEscort,
		ViewDistance:  consts.AIVisionDistance,
		ItemPickup:    PolicyPickup,
		ItemHandling:  PolicyDistribute,
	}
}
