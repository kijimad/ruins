package components

import "fmt"

// PlannerType は適用する行動計画の種別を表す
type PlannerType int

const (
	// PlannerRoaming はAIStateのタイミングサイクルで行動する。敵・中立NPCが使用する
	PlannerRoaming PlannerType = iota
	// PlannerSquad はリーダー追従とアイテム処理を含む。隊員が使用する
	PlannerSquad
)

// String はPlannerType名を返す
func (p PlannerType) String() string {
	switch p {
	case PlannerRoaming:
		return "roaming"
	case PlannerSquad:
		return "squad"
	default:
		panic(fmt.Sprintf("未知のPlannerType: %d", p))
	}
}

// CombatPolicy は戦闘時の行動方針を表す
type CombatPolicy int

const (
	// CombatAttack は敵対行動。視界内のプレイヤーまたは敵を攻撃する
	CombatAttack CombatPolicy = iota
	// CombatEvade は回避行動。敵から距離を取って逃げる
	CombatEvade
	// CombatIgnore は無関心。戦闘に反応しない。被ダメージで CombatAttack に変化する
	CombatIgnore
)

// String はCombatPolicy名を返す
func (p CombatPolicy) String() string {
	switch p {
	case CombatAttack:
		return "攻撃"
	case CombatEvade:
		return "回避"
	case CombatIgnore:
		return "無関心"
	default:
		panic(fmt.Sprintf("未知のCombatPolicy: %d", p))
	}
}

// AllSquadCombatPolicies は隊員UIで巡回可能な戦闘ポリシーを返す。
// CombatIgnore は中立NPC用であり、隊員には選択させない
func AllSquadCombatPolicies() []CombatPolicy {
	return []CombatPolicy{CombatAttack, CombatEvade}
}

// MovementPolicy は非戦闘時の移動方針を表す
type MovementPolicy string

// 敵・中立NPC用の移動ポリシーと、隊員用の移動ポリシーを定義する
const (
	MovementRandom      MovementPolicy = "random"
	MovementPatrol      MovementPolicy = "patrol"
	MovementWallHug     MovementPolicy = "wallHug"
	MovementStationary  MovementPolicy = "stationary"
	MovementWander      MovementPolicy = "wander"
	MovementTerritorial MovementPolicy = "territorial"
	MovementSwarm       MovementPolicy = "swarm"

	MovementEscort   MovementPolicy = "escort"
	MovementVanguard MovementPolicy = "vanguard"
	MovementRetreat  MovementPolicy = "retreat"
)

// String は日本語表示名を返す
func (p MovementPolicy) String() string {
	switch p {
	case MovementEscort:
		return "護衛"
	case MovementVanguard:
		return "前衛"
	case MovementPatrol:
		return "巡回"
	case MovementStationary:
		return "固定"
	case MovementRetreat:
		return "後退"
	case MovementRandom:
		return "ランダム"
	case MovementWallHug:
		return "壁沿い"
	case MovementWander:
		return "徘徊"
	case MovementTerritorial:
		return "縄張り"
	case MovementSwarm:
		return "群れ"
	default:
		return string(p)
	}
}

// AllSquadMovementPolicies は隊員UIで巡回可能な移動ポリシーを返す
func AllSquadMovementPolicies() []MovementPolicy {
	return []MovementPolicy{
		MovementEscort, MovementVanguard, MovementPatrol,
		MovementStationary, MovementRetreat,
	}
}

// ItemPickupPolicy はアイテム回収ポリシーを表す
type ItemPickupPolicy int

const (
	// PolicyPickup は回収。探索済みエリアのアイテムを拾う
	PolicyPickup ItemPickupPolicy = iota
	// PolicyIgnore は無視。アイテムを拾わない
	PolicyIgnore
)

// String はポリシー名を返す
func (p ItemPickupPolicy) String() string {
	switch p {
	case PolicyPickup:
		return "回収"
	case PolicyIgnore:
		return "無視"
	default:
		panic(fmt.Sprintf("未知のItemPickupPolicy: %d", p))
	}
}

// AllItemPickupPolicies は全アイテム回収ポリシーを返す
func AllItemPickupPolicies() []ItemPickupPolicy {
	return []ItemPickupPolicy{PolicyPickup, PolicyIgnore}
}

// ItemHandlingPolicy はアイテム処理ポリシーを表す
type ItemHandlingPolicy int

const (
	// PolicyKeep は保持。アイテムを持ち続ける
	PolicyKeep ItemHandlingPolicy = iota
	// PolicyDistribute は分配。運搬役に渡す
	PolicyDistribute
)

// String はポリシー名を返す
func (p ItemHandlingPolicy) String() string {
	switch p {
	case PolicyKeep:
		return "保持"
	case PolicyDistribute:
		return "分配"
	default:
		panic(fmt.Sprintf("未知のItemHandlingPolicy: %d", p))
	}
}

// AllItemHandlingPolicies は全アイテム処理ポリシーを返す
func AllItemHandlingPolicies() []ItemHandlingPolicy {
	return []ItemHandlingPolicy{PolicyKeep, PolicyDistribute}
}

// AIPolicy は全AIエンティティ共通の行動ポリシーを保持する。
// 敵、中立NPC、隊員が同じ構造で方針を表現する
type AIPolicy struct {
	// Planner は適用する行動計画の種別。resolvePlanner が参照する
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
}

// DefaultAIPolicy は隊員のデフォルトポリシーを定義する
var DefaultAIPolicy = AIPolicy{
	Planner:       PlannerSquad,
	CombatDefault: CombatAttack,
	CombatCurrent: CombatAttack,
	Movement:      MovementEscort,
	ItemPickup:    PolicyPickup,
	ItemHandling:  PolicyDistribute,
}

// ReactToHostile は被ダメージ時に戦闘方針を変化させる。
// CombatIgnore は反撃のため CombatAttack に遷移する
func (p *AIPolicy) ReactToHostile() {
	switch p.CombatDefault {
	case CombatIgnore:
		p.CombatCurrent = CombatAttack
	case CombatAttack, CombatEvade:
		// 既に戦闘的または回避中なので変化なし
	}
}

// ResetCombat は戦闘方針をデフォルトに復帰させる
func (p *AIPolicy) ResetCombat() {
	p.CombatCurrent = p.CombatDefault
}
