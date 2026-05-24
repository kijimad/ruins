package formula

// 命中率関連の定数
const (
	BaseHitRate          = 80 // 基本命中率（%）
	HitRatePerStatPoint  = 2  // 器用度と敏捷度の差1点あたりの命中率変化（%）
	MaxHitRate           = 95 // 最大命中率（%）
	MinHitRate           = 5  // 最小命中率（%）
	CriticalHitThreshold = 5  // クリティカルヒット判定しきい値（%以下）
	DiceMax              = 100
)

// ダメージ関連の定数
const (
	DamageRandomRange        = 6 // ダメージのランダム要素（1-6）
	CriticalDamageMultiplier = 3 // クリティカルダメージ倍率の分子
	CriticalDamageBase       = 2 // クリティカルダメージ倍率の分母（3/2 = 1.5倍）
	MinDamage                = 1 // 最低保証ダメージ
)

// HP計算関連の定数
const (
	HPBaseValue        = 30 // HP計算の基本値
	HPVitalityMultiply = 8  // HP計算の体力係数
)

// CalcHitRate は命中率を算出する。スキル補正などを含まない基本的な命中率計算
func CalcHitRate(dexterity, agility, weaponAccuracy int) int {
	hitRate := BaseHitRate + (dexterity-agility)*HitRatePerStatPoint
	hitRate += weaponAccuracy - BaseHitRate
	return clampHitRate(hitRate)
}

func clampHitRate(hitRate int) int {
	if hitRate > MaxHitRate {
		hitRate = MaxHitRate
	}
	if hitRate < MinHitRate {
		hitRate = MinHitRate
	}
	return hitRate
}

// ApplyCritical はクリティカルヒット倍率を適用する
func ApplyCritical(damage int) int {
	return damage * CriticalDamageMultiplier / CriticalDamageBase
}

// CalcHP はHP最大値を算出する
func CalcHP(vitality, strength, sensation int) int {
	return HPBaseValue + vitality*HPVitalityMultiply + strength + sensation
}
