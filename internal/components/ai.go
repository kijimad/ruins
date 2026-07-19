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

// SoloAI と SquadAI は独立したコンポーネントで、エンティティは片方だけを持つ。

// SoloAI は単独行動NPC用の設定と状態を保持する
type SoloAI struct {
	CombatDefault CombatPolicy
	CombatCurrent CombatPolicy
	Movement      SoloMovement
	ViewDistance  consts.Tile

	SubState              AIStateSubState
	StartSubStateTurn     consts.Turn
	DurationSubStateTurns consts.Turn
	// Origin/PatrolDir は AI のグリッド移動計算用に int タイルインデックス空間で保持する。
	// planner_solo の探索・方向計算が int で完結するため Coord[Tile] でなく Coord[int] にし、
	// GridElement(Coord[Tile]) との境界でだけ変換する。FindNextStep の BFS が int で完結するのと同じ方針。
	Origin       consts.Coord[int] // パトロール原点のタイル座標
	PatrolDir    consts.Coord[int] // パトロール方向。各成分は -1/0/1
	TargetEntity *ecs.Entity
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
