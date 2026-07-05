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

// AI はAIエンティティの統合コンポーネント。
// Plannerフィールドに具体的な設定を保持する
type AI struct {
	Planner PlannerConfig
}

// PlannerConfig はプランナー種別ごとの設定を表すインターフェース
type PlannerConfig interface {
	// Type はプランナーの種別を返す
	Type() PlannerType
	// ReactToHostile は被ダメージ時に戦闘方針を変化させる
	ReactToHostile()
	// ResetCombat は戦闘方針をデフォルトに復帰させる
	ResetCombat()
}

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

// DefaultSquadAI は隊員のデフォルトAIを返す
func DefaultSquadAI() AI {
	return AI{
		Planner: &SquadAI{
			CombatDefault: CombatAttack,
			CombatCurrent: CombatAttack,
			Movement:      SquadEscort,
			ViewDistance:  5,
			ItemPickup:    PolicyPickup,
			ItemHandling:  PolicyDistribute,
		},
	}
}
