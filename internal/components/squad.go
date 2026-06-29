package components

import ecs "github.com/x-hgg-x/goecs/v2"

// SquadMember は隊員であることを示し、所属するリーダーを参照する
type SquadMember struct {
	Leader ecs.Entity
}

// SquadPolicy は隊員の自律行動を制御するポリシーを保持する
type SquadPolicy struct {
	Position     PositionPolicy
	Combat       CombatPolicy
	ItemPickup   ItemPickupPolicy
	ItemHandling ItemHandlingPolicy
}

// DefaultSquadPolicy はデフォルトのポリシーを返す
func DefaultSquadPolicy() SquadPolicy {
	return SquadPolicy{
		Position:     PolicyEscort,
		Combat:       PolicyAttack,
		ItemPickup:   PolicyPickup,
		ItemHandling: PolicyDistribute,
	}
}

// PositionPolicy は位置ポリシーを表す
type PositionPolicy int

const (
	// PolicyEscort は護衛。リーダーの近くにとどまる
	PolicyEscort PositionPolicy = iota
	// PolicyVanguard は前衛。リーダーの前方に展開する
	PolicyVanguard
	// PolicyPatrol は巡回。探索済みエリア内を自律巡回する
	PolicyPatrol
	// PolicyHold は待機。その場にとどまる
	PolicyHold
	// PolicyRetreat は撤退。出口に向かって移動する
	PolicyRetreat
)

// String はポリシー名を返す
func (p PositionPolicy) String() string {
	switch p {
	case PolicyEscort:
		return "護衛"
	case PolicyVanguard:
		return "前衛"
	case PolicyPatrol:
		return "巡回"
	case PolicyHold:
		return "待機"
	case PolicyRetreat:
		return "撤退"
	default:
		return unknownLabel
	}
}

// AllPositionPolicies は全位置ポリシーを返す
func AllPositionPolicies() []PositionPolicy {
	return []PositionPolicy{PolicyEscort, PolicyVanguard, PolicyPatrol, PolicyHold, PolicyRetreat}
}

// CombatPolicy は戦闘ポリシーを表す
type CombatPolicy int

const (
	// PolicyAttack は攻撃。敵を攻撃する
	PolicyAttack CombatPolicy = iota
	// PolicyEvade は回避。戦闘を避けて逃げる
	PolicyEvade
)

// String はポリシー名を返す
func (p CombatPolicy) String() string {
	switch p {
	case PolicyAttack:
		return "攻撃"
	case PolicyEvade:
		return "回避"
	default:
		return unknownLabel
	}
}

// AllCombatPolicies は全戦闘ポリシーを返す
func AllCombatPolicies() []CombatPolicy {
	return []CombatPolicy{PolicyAttack, PolicyEvade}
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
		return unknownLabel
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
		return unknownLabel
	}
}

// AllItemHandlingPolicies は全アイテム処理ポリシーを返す
func AllItemHandlingPolicies() []ItemHandlingPolicy {
	return []ItemHandlingPolicy{PolicyKeep, PolicyDistribute}
}
