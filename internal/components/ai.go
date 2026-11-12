package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
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

// SoloAI は単独行動NPC用の設定と状態を保持する
type SoloAI struct {
	CombatDefault rawschema.CombatPolicyType
	CombatCurrent rawschema.CombatPolicyType
	Movement      rawschema.MovementPatternType
	ViewDistance  consts.Tile

	SubState              AIStateSubState
	StartSubStateTurn     consts.Turn
	DurationSubStateTurns consts.Turn
	Origin                consts.Coord[consts.Tile] // パトロール原点のタイル座標
	PatrolDir             consts.Coord[consts.Tile] // パトロール方向。各成分は -1/0/1
	TargetEntity          *ecs.Entity
}

// ReactToHostile は被ダメージ時に戦闘方針を変化させる。
// rawschema.Ignore は反撃のため rawschema.Attack に遷移する
func (s *SoloAI) ReactToHostile() {
	switch s.CombatDefault {
	case rawschema.Ignore:
		s.CombatCurrent = rawschema.Attack
	case rawschema.Attack, rawschema.Evade:
	}
}

// ResetCombat は戦闘方針をデフォルトに復帰させる
func (s *SoloAI) ResetCombat() {
	s.CombatCurrent = s.CombatDefault
}
