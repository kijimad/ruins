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

// weaponDamageKeys は武器スキルIDからダメージ効果キーへのマッピング
var weaponDamageKeys = map[SkillID]ModifierKey{
	SkillSword:   ModSwordDamage,
	SkillSpear:   ModSpearDamage,
	SkillFist:    ModFistDamage,
	SkillHandgun: ModHandgunDamage,
	SkillRifle:   ModRifleDamage,
	SkillCannon:  ModCannonDamage,
}

// WeaponDamageKey は武器スキルIDに対応するダメージ効果キーを返す。未定義ならpanicする
func WeaponDamageKey(id SkillID) ModifierKey {
	key, ok := weaponDamageKeys[id]
	if !ok {
		panic(fmt.Sprintf("未定義の武器スキルID（ダメージ）: %q", id))
	}
	return key
}

// weaponAccuracyKeys は武器スキルIDから命中効果キーへのマッピング
var weaponAccuracyKeys = map[SkillID]ModifierKey{
	SkillSword:   ModSwordAccuracy,
	SkillSpear:   ModSpearAccuracy,
	SkillFist:    ModFistAccuracy,
	SkillHandgun: ModHandgunAccuracy,
	SkillRifle:   ModRifleAccuracy,
	SkillCannon:  ModCannonAccuracy,
}

// WeaponAccuracyKey は武器スキルIDに対応する命中効果キーを返す。未定義ならpanicする
func WeaponAccuracyKey(id SkillID) ModifierKey {
	key, ok := weaponAccuracyKeys[id]
	if !ok {
		panic(fmt.Sprintf("未定義の武器スキルID（命中）: %q", id))
	}
	return key
}

// elementResistKeys は元素タイプから耐性効果キーへのマッピング
var elementResistKeys = map[ElementType]ModifierKey{
	ElementTypeFire:    ModFireResist,
	ElementTypeThunder: ModThunderResist,
	ElementTypeChill:   ModChillResist,
	ElementTypePhoton:  ModPhotonResist,
}

// ElementResistKey は元素タイプに対応する耐性効果キーを返す。未定義ならpanicする
func ElementResistKey(elem ElementType) ModifierKey {
	key, ok := elementResistKeys[elem]
	if !ok {
		panic(fmt.Sprintf("未定義の元素タイプ（耐性）: %q", elem))
	}
	return key
}

// スキル効果係数の定数。スキル値1あたりの倍率変化量（%）を定義する。
// 正の値はスキルが高いほど効果が増し、負の値は効果が減る。
const (
	coeffWeaponDamage   = 5  // 武器ダメージ: スキルLv1あたり+5%
	coeffWeaponAccuracy = 3  // 武器命中: スキルLv1あたり+3%
	coeffElementResist  = -3 // 元素耐性: スキルLv1あたり-3%（被ダメージ軽減）
	coeffColdProgress   = -3 // 低体温進行: スキルLv1あたり-3%
	coeffHeatProgress   = -3 // 高体温進行: スキルLv1あたり-3%
	coeffHungerProgress = -2 // 空腹進行: スキルLv1あたり-2%
	coeffHealingEffect  = 5  // 回復効果: スキルLv1あたり+5%
	coeffMaxWeight      = 4  // 最大所持重量: スキルLv1あたり+4%
	coeffExploration    = 4  // アイテム発見率: スキルLv1あたり+4%
	coeffEnemyVision    = -3 // 敵視界距離: スキルLv1あたり-3%
	coeffNightVision    = 5  // 暗所視界: スキルLv1あたり+5%
	coeffMoveCost       = -2 // 移動コスト: スキルLv1あたり-2%
	coeffCraftCost      = -3 // 素材消費: スキルLv1あたり-3%
	coeffSmithQuality   = 3  // 合成品質: スキルLv1あたり+3%
	coeffBuyPrice       = -2 // 買値: スキルLv1あたり-2%
	coeffSellPrice      = 2  // 売値: スキルLv1あたり+2%
	coeffHeavyArmor     = -5 // 重装備ペナルティ: スキルLv1あたり-5%
)

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
	Exploration    int                 // TODO: アイテム発見システム実装時に適用する。アイテム発見率倍率%
	EnemyVision    int                 // 敵視界距離倍率%
	NightVision    int                 // TODO: 暗所視界システム実装時に適用する。暗所視界倍率%
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

// RecalculateCharModifiers はスキル、能力値、健康状態から全効果倍率を計算する。
// abils, hs は nil でもよい。
func RecalculateCharModifiers(skills *Skills, abils *Abilities, hs *HealthStatus) *CharModifiers {
	e := &CharModifiers{}
	src := make(map[ModifierKey][]ModifierSource)

	calcEffect := func(key ModifierKey, skillID SkillID, coeff int) int {
		v := skills.Get(skillID).Value
		bonus := v * coeff
		src[key] = append(src[key], ModifierSource{
			Label: fmt.Sprintf("%s Lv%d", SkillName(skillID), v),
			Value: bonus,
		})

		// 対応する能力値による補正。能力値1ポイントにつきスキル係数と同じ方向に±1%
		if abils != nil {
			ablID := SkillAbilityID(skillID)
			ablVal := abils.ValueOf(ablID)
			ablCoeff := 1
			if coeff < 0 {
				ablCoeff = -1
			}
			ablBonus := ablVal * ablCoeff
			src[key] = append(src[key], ModifierSource{
				Label: fmt.Sprintf("%s %d", AbilityName(ablID), ablVal),
				Value: ablBonus,
			})
			bonus += ablBonus
		}

		return 100 + bonus
	}

	e.WeaponDamage = make(map[SkillID]int, len(weaponSkillIDs))
	e.WeaponAccuracy = make(map[SkillID]int, len(weaponSkillIDs))
	for _, id := range weaponSkillIDs {
		e.WeaponDamage[id] = calcEffect(WeaponDamageKey(id), id, coeffWeaponDamage)
		e.WeaponAccuracy[id] = calcEffect(WeaponAccuracyKey(id), id, coeffWeaponAccuracy)
	}

	e.ElementResist = map[ElementType]int{
		ElementTypeFire:    calcEffect(ModFireResist, SkillFireResist, coeffElementResist),
		ElementTypeThunder: calcEffect(ModThunderResist, SkillThunderResist, coeffElementResist),
		ElementTypeChill:   calcEffect(ModChillResist, SkillChillResist, coeffElementResist),
		ElementTypePhoton:  calcEffect(ModPhotonResist, SkillPhotonResist, coeffElementResist),
	}

	e.ColdProgress = calcEffect(ModColdProgress, SkillColdResist, coeffColdProgress)
	e.HeatProgress = calcEffect(ModHeatProgress, SkillHeatResist, coeffHeatProgress)
	e.HungerProgress = calcEffect(ModHungerProgress, SkillHungerResist, coeffHungerProgress)
	e.HealingEffect = calcEffect(ModHealingEffect, SkillHealing, coeffHealingEffect)
	e.MaxWeight = calcEffect(ModMaxWeight, SkillWeightBearing, coeffMaxWeight)
	e.Exploration = calcEffect(ModExploration, SkillExploration, coeffExploration)
	e.EnemyVision = calcEffect(ModEnemyVision, SkillStealth, coeffEnemyVision)
	e.NightVision = calcEffect(ModNightVision, SkillNightVision, coeffNightVision)
	e.MoveCost = calcEffect(ModMoveCost, SkillSprinting, coeffMoveCost)
	e.CraftCost = calcEffect(ModCraftCost, SkillCrafting, coeffCraftCost)
	e.SmithQuality = calcEffect(ModSmithQuality, SkillSmithing, coeffSmithQuality)
	e.BuyPrice = calcEffect(ModBuyPrice, SkillNegotiation, coeffBuyPrice)
	e.SellPrice = calcEffect(ModSellPrice, SkillNegotiation, coeffSellPrice)
	e.HeavyArmor = calcEffect(ModHeavyArmor, SkillHeavyArmor, coeffHeavyArmor)

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
