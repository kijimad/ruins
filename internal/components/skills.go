package components

import (
	"fmt"
	"slices"
)

// SkillID はスキルの識別子
type SkillID string

// スキルID定数
const (
	SkillSword         SkillID = "sword"          // 刀剣
	SkillSpear         SkillID = "spear"          // 長物
	SkillFist          SkillID = "fist"           // 格闘
	SkillWeightBearing SkillID = "weight_bearing" // 荷重

	SkillBow         SkillID = "bow"         // 弓術
	SkillHandgun     SkillID = "handgun"     // 拳銃
	SkillRifle       SkillID = "rifle"       // 小銃
	SkillCannon      SkillID = "cannon"      // 砲撃
	SkillExploration SkillID = "exploration" // 探索

	SkillCrafting    SkillID = "crafting"    // クラフト
	SkillSmithing    SkillID = "smithing"    // 調合
	SkillNegotiation SkillID = "negotiation" // 交渉
	SkillMechanic    SkillID = "mechanic"    // 機械

	SkillSprinting   SkillID = "sprinting"    // 走破
	SkillStealth     SkillID = "stealth"      // 隠密
	SkillNightVision SkillID = "night_vision" // 暗視

	SkillColdResist   SkillID = "cold_resist"   // 耐寒
	SkillHungerResist SkillID = "hunger_resist" // 耐餓
	SkillHealing      SkillID = "healing"       // 治療

	SkillHeavyArmor    SkillID = "heavy_armor"    // 重装
	SkillFireResist    SkillID = "fire_resist"    // 耐火
	SkillThunderResist SkillID = "thunder_resist" // 耐電
	SkillChillResist   SkillID = "chill_resist"   // 耐冷
	SkillPhotonResist  SkillID = "photon_resist"  // 耐光
)

// AllSkillIDs は定義済みの全SkillIDのリスト。raw の skills の行と同じ集合になることを読み込み時に検証する
var AllSkillIDs = []SkillID{
	SkillSword, SkillSpear, SkillFist,
	SkillBow, SkillHandgun, SkillRifle, SkillCannon,
	SkillCrafting, SkillSmithing, SkillNegotiation, SkillMechanic,
	SkillSprinting, SkillStealth, SkillNightVision, SkillWeightBearing,
	SkillColdResist, SkillHungerResist, SkillHealing, SkillExploration,
	SkillHeavyArmor, SkillFireResist, SkillThunderResist, SkillChillResist, SkillPhotonResist,
}

// IsSkillID は id が定義済みのスキルかを返す
func IsSkillID(id SkillID) bool {
	return slices.Contains(AllSkillIDs, id)
}

// WeaponSkillIDs は武器に対応するスキルIDのリスト
var WeaponSkillIDs = []SkillID{
	SkillSword, SkillSpear, SkillFist,
	SkillBow, SkillHandgun, SkillRifle, SkillCannon,
}

// Skill は個別のスキル
type Skill struct {
	Value int     // スキル値
	Exp   IntPool // 蓄積経験値。Maxに達するとスキルアップする
}

// Skills はエンティティが持つスキルセット
type Skills struct {
	Data map[SkillID]*Skill
}

// Get は指定されたスキルIDのSkillを返す。未定義ならpanicする
func (s *Skills) Get(id SkillID) *Skill {
	sk, ok := s.Data[id]
	if !ok {
		panic(fmt.Sprintf("undefined skill ID: %q", id))
	}
	return sk
}

// LevelUpExp はスキルアップに必要な経験値
const LevelUpExp = 100

// NewSkills は全スキルを0で初期化したSkillsを返す
func NewSkills() *Skills {
	data := make(map[SkillID]*Skill, len(AllSkillIDs))
	for _, id := range AllSkillIDs {
		data[id] = &Skill{Exp: IntPool{Max: LevelUpExp}}
	}
	return &Skills{Data: data}
}

// WeaponSkillID は武器種別に対応するスキルIDを返す。武器種別はそれと同じ名前の武器スキルで上達する
func WeaponSkillID(at AttackType) (SkillID, bool) {
	id := SkillID(at.Type)
	return id, slices.Contains(WeaponSkillIDs, id)
}
