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
	return ClampHitRate(hitRate)
}

// ClampHitRate は命中率を下限 MinHitRate と上限 MaxHitRate に収める
func ClampHitRate(hitRate int) int {
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

// DamageInput は1回の命中のダメージの式への入力
type DamageInput struct {
	Ability      int  // 攻撃側の能力値。近接は筋力、遠距離は感覚
	Die          int  // ダメージ乱数の目。1〜DamageRandomRange
	WeaponDamage int  // 武器の攻撃力
	Modifier     int  // 攻撃ごとの加算補正
	SkillMult    int  // 武器熟練の倍率。百分率
	Critical     bool // クリティカルか
	ElementMult  int  // 対象の属性耐性の倍率。百分率で、耐性が無ければ100
	Defense      int  // 対象の防御
}

// Damage は1回の命中のダメージを返す。能力値・ダイス・武器・補正の和に熟練の倍率を切り捨てで掛け、
// クリティカル、属性耐性、防御の順に適用する。属性耐性の後と防御の後で最低 MinDamage を保証する
func Damage(in DamageInput) int {
	damage := applyPercent(in.Ability+in.Die+in.WeaponDamage+in.Modifier, in.SkillMult)
	if in.Critical {
		damage = ApplyCritical(damage)
	}
	damage = max(applyPercent(damage, in.ElementMult), MinDamage)
	return max(damage-in.Defense, MinDamage)
}

// applyPercent は value に百分率 percent を切り捨てで掛ける
func applyPercent(value, percent int) int {
	return value * percent / 100
}

// CalcHP はHP最大値を算出する
func CalcHP(vitality, strength, sensation int) int {
	return HPBaseValue + vitality*HPVitalityMultiply + strength + sensation
}
