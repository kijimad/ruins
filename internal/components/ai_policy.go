package components

// PlannerType は適用する行動計画の種別を表す
type PlannerType string

const (
	// PlannerSolo は単独行動NPC用。状態遷移とMovementPolicyで行動を決定する
	PlannerSolo PlannerType = "solo"
	// PlannerSquad はリーダー追従とアイテム処理を含む。隊員が使用する
	PlannerSquad PlannerType = "squad"
)

// String は日本語表示名を返す
func (p PlannerType) String() string {
	switch p {
	case PlannerSolo:
		return "単独"
	case PlannerSquad:
		return "隊員"
	default:
		return string(p)
	}
}

// CombatPolicy は戦闘時の行動方針を表す
type CombatPolicy string

const (
	// CombatAttack は敵対行動。視界内のプレイヤーまたは敵を攻撃する
	CombatAttack CombatPolicy = "attack"
	// CombatEvade は回避行動。敵から距離を取って逃げる
	CombatEvade CombatPolicy = "evade"
	// CombatIgnore は無関心。戦闘に反応しない。被ダメージで CombatAttack に変化する
	CombatIgnore CombatPolicy = "ignore"
)

// String は日本語表示名を返す
func (p CombatPolicy) String() string {
	switch p {
	case CombatAttack:
		return "攻撃"
	case CombatEvade:
		return "回避"
	case CombatIgnore:
		return "無関心"
	default:
		return string(p)
	}
}

// AllSquadCombatPolicies は隊員UIで巡回可能な戦闘ポリシーを返す。
// CombatIgnore は中立NPC用であり、隊員には選択させない
func AllSquadCombatPolicies() []CombatPolicy {
	return []CombatPolicy{CombatAttack, CombatEvade}
}

// SoloMovement は単独行動NPC用の移動方針を表す
type SoloMovement string

const (
	// SoloRandom はランダムに移動する
	SoloRandom SoloMovement = "random"
	// SoloPatrol は決まった経路を巡回する
	SoloPatrol SoloMovement = "patrol"
	// SoloWallHug は壁沿いに移動する
	SoloWallHug SoloMovement = "wallHug"
	// SoloStationary はその場に留まる
	SoloStationary SoloMovement = "stationary"
	// SoloWander は緩やかにさまよう
	SoloWander SoloMovement = "wander"
	// SoloTerritorial はスポーン地点の周辺を守る
	SoloTerritorial SoloMovement = "territorial"
	// SoloSwarm は近くの同種と群れで行動する
	SoloSwarm SoloMovement = "swarm"
)

// String は日本語表示名を返す
func (p SoloMovement) String() string {
	switch p {
	case SoloRandom:
		return "ランダム"
	case SoloPatrol:
		return "巡回"
	case SoloWallHug:
		return "壁沿い"
	case SoloStationary:
		return "固定"
	case SoloWander:
		return "徘徊"
	case SoloTerritorial:
		return "縄張り"
	case SoloSwarm:
		return "群れ"
	default:
		return string(p)
	}
}

// SquadMovement は隊員用の移動方針を表す
type SquadMovement string

const (
	// SquadEscort はリーダーに追従する
	SquadEscort SquadMovement = "escort"
	// SquadVanguard はリーダーの前方に位置する
	SquadVanguard SquadMovement = "vanguard"
	// SquadPatrol はリーダー周辺を巡回する
	SquadPatrol SquadMovement = "patrol"
	// SquadStationary はその場に留まる
	SquadStationary SquadMovement = "stationary"
	// SquadRetreat はリーダーの後方に退避する
	SquadRetreat SquadMovement = "retreat"
)

// String は日本語表示名を返す
func (p SquadMovement) String() string {
	switch p {
	case SquadEscort:
		return "護衛"
	case SquadVanguard:
		return "前衛"
	case SquadPatrol:
		return "巡回"
	case SquadStationary:
		return "固定"
	case SquadRetreat:
		return "後退"
	default:
		return string(p)
	}
}

// AllSquadMovements は隊員UIで巡回可能な移動ポリシーを返す
func AllSquadMovements() []SquadMovement {
	return []SquadMovement{
		SquadEscort, SquadVanguard, SquadPatrol,
		SquadStationary, SquadRetreat,
	}
}

// ItemPickupPolicy はアイテム回収ポリシーを表す
type ItemPickupPolicy string

const (
	// PolicyPickup は回収。探索済みエリアのアイテムを拾う
	PolicyPickup ItemPickupPolicy = "pickup"
	// PolicyIgnore は無視。アイテムを拾わない
	PolicyIgnore ItemPickupPolicy = "ignore"
)

// String は日本語表示名を返す
func (p ItemPickupPolicy) String() string {
	switch p {
	case PolicyPickup:
		return "回収"
	case PolicyIgnore:
		return "無視"
	default:
		return string(p)
	}
}

// AllItemPickupPolicies は全アイテム回収ポリシーを返す
func AllItemPickupPolicies() []ItemPickupPolicy {
	return []ItemPickupPolicy{PolicyPickup, PolicyIgnore}
}

// ItemHandlingPolicy はアイテム処理ポリシーを表す
type ItemHandlingPolicy string

const (
	// PolicyKeep は保持。アイテムを持ち続ける
	PolicyKeep ItemHandlingPolicy = "keep"
	// PolicyDistribute は分配。運搬役に渡す
	PolicyDistribute ItemHandlingPolicy = "distribute"
)

// String は日本語表示名を返す
func (p ItemHandlingPolicy) String() string {
	switch p {
	case PolicyKeep:
		return "保持"
	case PolicyDistribute:
		return "分配"
	default:
		return string(p)
	}
}

// AllItemHandlingPolicies は全アイテム処理ポリシーを返す
func AllItemHandlingPolicies() []ItemHandlingPolicy {
	return []ItemHandlingPolicy{PolicyKeep, PolicyDistribute}
}
