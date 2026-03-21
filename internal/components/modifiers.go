package components

import "fmt"

// ModifierKey は効果倍率の識別キー
type ModifierKey string

// 効果キー定数
const (
	ModFireResist     ModifierKey = "fire_resist"
	ModThunderResist  ModifierKey = "thunder_resist"
	ModChillResist    ModifierKey = "chill_resist"
	ModPhotonResist   ModifierKey = "photon_resist"
	ModColdProgress   ModifierKey = "cold_progress"
	ModHeatProgress   ModifierKey = "heat_progress"
	ModHungerProgress ModifierKey = "hunger_progress"
	ModHealingEffect  ModifierKey = "healing_effect"
	ModMaxWeight      ModifierKey = "max_weight"
	ModExploration    ModifierKey = "exploration"
	ModEnemyVision    ModifierKey = "enemy_vision"
	ModNightVision    ModifierKey = "night_vision"
	ModMoveCost       ModifierKey = "move_cost"
	ModCraftCost      ModifierKey = "craft_cost"
	ModSmithQuality   ModifierKey = "smith_quality"
	ModBuyPrice       ModifierKey = "buy_price"
	ModSellPrice      ModifierKey = "sell_price"
	ModHeavyArmor     ModifierKey = "heavy_armor"

	ModSwordDamage   ModifierKey = "sword_damage"
	ModSpearDamage   ModifierKey = "spear_damage"
	ModFistDamage    ModifierKey = "fist_damage"
	ModHandgunDamage ModifierKey = "handgun_damage"
	ModRifleDamage   ModifierKey = "rifle_damage"
	ModCannonDamage  ModifierKey = "cannon_damage"

	ModSwordAccuracy   ModifierKey = "sword_accuracy"
	ModSpearAccuracy   ModifierKey = "spear_accuracy"
	ModFistAccuracy    ModifierKey = "fist_accuracy"
	ModHandgunAccuracy ModifierKey = "handgun_accuracy"
	ModRifleAccuracy   ModifierKey = "rifle_accuracy"
	ModCannonAccuracy  ModifierKey = "cannon_accuracy"
)

// WeaponDamageKeys は武器スキルIDからダメージ効果キーへのマッピング
var WeaponDamageKeys = map[SkillID]ModifierKey{
	SkillSword:   ModSwordDamage,
	SkillSpear:   ModSpearDamage,
	SkillFist:    ModFistDamage,
	SkillHandgun: ModHandgunDamage,
	SkillRifle:   ModRifleDamage,
	SkillCannon:  ModCannonDamage,
}

// WeaponAccuracyKeys は武器スキルIDから命中効果キーへのマッピング
var WeaponAccuracyKeys = map[SkillID]ModifierKey{
	SkillSword:   ModSwordAccuracy,
	SkillSpear:   ModSpearAccuracy,
	SkillFist:    ModFistAccuracy,
	SkillHandgun: ModHandgunAccuracy,
	SkillRifle:   ModRifleAccuracy,
	SkillCannon:  ModCannonAccuracy,
}

// ElementResistKeys は元素タイプから耐性効果キーへのマッピング
var ElementResistKeys = map[ElementType]ModifierKey{
	ElementTypeFire:    ModFireResist,
	ElementTypeThunder: ModThunderResist,
	ElementTypeChill:   ModChillResist,
	ElementTypePhoton:  ModPhotonResist,
}

// ModifierSource は効果倍率の算出元を表す。
// スキル以外の要因（健康状態など）にも対応できる汎用的な構造にしている。
type ModifierSource struct {
	Label string // 表示名。例: "刀剣 Lv2", "低体温"
	Value int    // この要因による変化量。例: +10, -15
}

// CharModifiers はエンティティの効果倍率を集約するコンポーネント。
// スキル、健康状態など複数の要因から算出される。100が基準値で変化なし。
type CharModifiers struct {
	WeaponDamage   map[SkillID]int     // 武器ダメージ倍率%
	WeaponAccuracy map[SkillID]int     // 武器命中倍率%
	ElementResist  map[ElementType]int // 元素耐性倍率%
	ColdProgress   int                 // 低体温進行倍率%
	HeatProgress   int                 // 高体温進行倍率%
	HungerProgress int                 // 空腹進行倍率%
	HealingEffect  int                 // 回復効果倍率%
	MaxWeight      int                 // 最大所持重量倍率%
	Exploration    int                 // アイテム発見率倍率%
	EnemyVision    int                 // 敵視界距離倍率%
	NightVision    int                 // 暗所視界倍率%
	MoveCost       int                 // 移動APコスト倍率%
	CraftCost      int                 // 素材消費量倍率%
	SmithQuality   int                 // 合成品質倍率%
	BuyPrice       int                 // 買値倍率%
	SellPrice      int                 // 売値倍率%
	HeavyArmor     int                 // 重装備AGIペナルティ倍率%

	// Sources は各効果の算出元を保持する。
	// 1つの効果に複数の要因が影響しうるためスライスにしている。
	Sources map[ModifierKey][]ModifierSource
}

// RecalculateCharModifiers はスキルと健康状態から全効果倍率を計算する。
// hs は nil でもよい。
func RecalculateCharModifiers(skills *Skills, hs *HealthStatus) *CharModifiers {
	e := &CharModifiers{}
	src := make(map[ModifierKey][]ModifierSource)

	calcEffect := func(key ModifierKey, skillID SkillID, coeff int) int {
		v := skills.Data[skillID].Value
		bonus := v * coeff
		src[key] = append(src[key], ModifierSource{
			Label: fmt.Sprintf("%s Lv%d", SkillName[skillID], v),
			Value: bonus,
		})
		return 100 + bonus
	}

	e.WeaponDamage = make(map[SkillID]int, len(weaponSkillIDs))
	e.WeaponAccuracy = make(map[SkillID]int, len(weaponSkillIDs))
	for _, id := range weaponSkillIDs {
		e.WeaponDamage[id] = calcEffect(WeaponDamageKeys[id], id, 5)
		e.WeaponAccuracy[id] = calcEffect(WeaponAccuracyKeys[id], id, 3)
	}

	e.ElementResist = map[ElementType]int{
		ElementTypeFire:    calcEffect(ModFireResist, SkillFireResist, -3),
		ElementTypeThunder: calcEffect(ModThunderResist, SkillThunderResist, -3),
		ElementTypeChill:   calcEffect(ModChillResist, SkillChillResist, -3),
		ElementTypePhoton:  calcEffect(ModPhotonResist, SkillPhotonResist, -3),
	}

	e.ColdProgress = calcEffect(ModColdProgress, SkillColdResist, -3)
	e.HeatProgress = calcEffect(ModHeatProgress, SkillHeatResist, -3)
	e.HungerProgress = calcEffect(ModHungerProgress, SkillHungerResist, -2)
	e.HealingEffect = calcEffect(ModHealingEffect, SkillHealing, 5)
	e.MaxWeight = calcEffect(ModMaxWeight, SkillWeightBearing, 4)
	e.Exploration = calcEffect(ModExploration, SkillExploration, 4)
	e.EnemyVision = calcEffect(ModEnemyVision, SkillStealth, -3)
	e.NightVision = calcEffect(ModNightVision, SkillNightVision, 5)
	e.MoveCost = calcEffect(ModMoveCost, SkillSprinting, -2)
	e.CraftCost = calcEffect(ModCraftCost, SkillCrafting, -3)
	e.SmithQuality = calcEffect(ModSmithQuality, SkillSmithing, 3)
	e.BuyPrice = calcEffect(ModBuyPrice, SkillNegotiation, -2)
	e.SellPrice = calcEffect(ModSellPrice, SkillNegotiation, 2)
	e.HeavyArmor = calcEffect(ModHeavyArmor, SkillHeavyArmor, -5)

	// 健康状態によるペナルティ
	if hs != nil {
		wb := &hs.Parts[BodyPartWholeBody]
		for _, cond := range wb.Conditions {
			if penalty := temperatureMovePenalty(cond.Severity); penalty != 0 {
				e.MoveCost += penalty
				src[ModMoveCost] = append(src[ModMoveCost], ModifierSource{
					Label: ConditionTypeDisplayName(cond.Type),
					Value: penalty,
				})
			}
		}
	}

	e.Sources = src
	return e
}

// temperatureMovePenalty は体温異常の重症度に応じた移動コスト増加量を返す
func temperatureMovePenalty(severity Severity) int {
	switch severity {
	case SeveritySevere:
		return 30
	case SeverityMedium:
		return 20
	case SeverityMinor:
		return 10
	default:
		return 0
	}
}
