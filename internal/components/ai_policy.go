package components

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
		return "Attack"
	case CombatEvade:
		return "Evade"
	case CombatIgnore:
		return "Indifferent"
	default:
		return string(p)
	}
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
		return "Random"
	case SoloPatrol:
		return "Patrol"
	case SoloWallHug:
		return "Along walls"
	case SoloStationary:
		return "Fixed"
	case SoloWander:
		return "Wander"
	case SoloTerritorial:
		return "Territory"
	case SoloSwarm:
		return "Flock"
	default:
		return string(p)
	}
}
