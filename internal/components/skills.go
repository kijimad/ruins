package components

// AttributeID はスキルの成長に影響する属性の識別子
type AttributeID int

// 属性ID定数
const (
	AttrSTR AttributeID = iota // 筋力
	AttrSEN                    // 感覚
	AttrDEX                    // 器用
	AttrAGI                    // 敏捷
	AttrVIT                    // 体力
	AttrDEF                    // 防御
)

// ValueOf は指定された属性IDに対応するTotal値を返す
func (a *Attributes) ValueOf(id AttributeID) int {
	switch id {
	case AttrSTR:
		return a.Strength.Total
	case AttrSEN:
		return a.Sensation.Total
	case AttrDEX:
		return a.Dexterity.Total
	case AttrAGI:
		return a.Agility.Total
	case AttrVIT:
		return a.Vitality.Total
	case AttrDEF:
		return a.Defense.Total
	default:
		return 0
	}
}

// SkillID はスキルの識別子
type SkillID string

// スキルID定数
const (
	SkillSword         SkillID = "sword"          // 刀剣
	SkillSpear         SkillID = "spear"          // 長物
	SkillFist          SkillID = "fist"           // 格闘
	SkillWeightBearing SkillID = "weight_bearing" // 荷重

	SkillHandgun     SkillID = "handgun"     // 拳銃
	SkillRifle       SkillID = "rifle"       // 小銃
	SkillCannon      SkillID = "cannon"      // 砲撃
	SkillExploration SkillID = "exploration" // 探索

	SkillCrafting    SkillID = "crafting"    // 合成
	SkillSmithing    SkillID = "smithing"    // 調合
	SkillNegotiation SkillID = "negotiation" // 交渉

	SkillSprinting   SkillID = "sprinting"    // 走破
	SkillStealth     SkillID = "stealth"      // 隠密
	SkillNightVision SkillID = "night_vision" // 暗視

	SkillColdResist   SkillID = "cold_resist"   // 耐寒
	SkillHeatResist   SkillID = "heat_resist"   // 耐暑
	SkillHungerResist SkillID = "hunger_resist" // 耐餓
	SkillHealing      SkillID = "healing"       // 治療

	SkillHeavyArmor    SkillID = "heavy_armor"    // 重装
	SkillFireResist    SkillID = "fire_resist"    // 耐火
	SkillThunderResist SkillID = "thunder_resist" // 耐電
	SkillChillResist   SkillID = "chill_resist"   // 耐冷
	SkillPhotonResist  SkillID = "photon_resist"  // 耐光
)

// SkillAttribute はスキルIDから対応する属性IDへのマッピング。
// 属性値が高いほど、対応するスキルの成長が速くなる。
var SkillAttribute = map[SkillID]AttributeID{
	SkillSword:         AttrSTR,
	SkillSpear:         AttrSTR,
	SkillFist:          AttrSTR,
	SkillWeightBearing: AttrSTR,

	SkillHandgun:     AttrSEN,
	SkillRifle:       AttrSEN,
	SkillCannon:      AttrSEN,
	SkillExploration: AttrSEN,

	SkillCrafting:    AttrDEX,
	SkillSmithing:    AttrDEX,
	SkillNegotiation: AttrDEX,

	SkillSprinting:   AttrAGI,
	SkillStealth:     AttrAGI,
	SkillNightVision: AttrAGI,

	SkillColdResist:   AttrVIT,
	SkillHeatResist:   AttrVIT,
	SkillHungerResist: AttrVIT,
	SkillHealing:      AttrVIT,

	SkillHeavyArmor:    AttrDEF,
	SkillFireResist:    AttrDEF,
	SkillThunderResist: AttrDEF,
	SkillChillResist:   AttrDEF,
	SkillPhotonResist:  AttrDEF,
}

// SkillName はスキルIDの表示名を返す
var SkillName = map[SkillID]string{
	SkillSword:         "刀剣",
	SkillSpear:         "長物",
	SkillFist:          "格闘",
	SkillWeightBearing: "荷重",
	SkillHandgun:       "拳銃",
	SkillRifle:         "小銃",
	SkillCannon:        "砲撃",
	SkillExploration:   "探索",
	SkillCrafting:      "合成",
	SkillSmithing:      "調合",
	SkillNegotiation:   "交渉",
	SkillSprinting:     "走破",
	SkillStealth:       "隠密",
	SkillNightVision:   "暗視",
	SkillColdResist:    "耐寒",
	SkillHeatResist:    "耐暑",
	SkillHungerResist:  "耐餓",
	SkillHealing:       "治療",
	SkillHeavyArmor:    "重装",
	SkillFireResist:    "耐火",
	SkillThunderResist: "耐電",
	SkillChillResist:   "耐冷",
	SkillPhotonResist:  "耐光",
}

// AllSkillIDs は定義済みの全SkillIDのリスト
var AllSkillIDs = []SkillID{
	SkillSword, SkillSpear, SkillFist, SkillWeightBearing,
	SkillHandgun, SkillRifle, SkillCannon, SkillExploration,
	SkillCrafting, SkillSmithing, SkillNegotiation,
	SkillSprinting, SkillStealth, SkillNightVision,
	SkillColdResist, SkillHeatResist, SkillHungerResist, SkillHealing,
	SkillHeavyArmor, SkillFireResist, SkillThunderResist, SkillChillResist, SkillPhotonResist,
}

// weaponSkillIDs は武器に対応するスキルIDのリスト
var weaponSkillIDs = []SkillID{
	SkillSword, SkillSpear, SkillFist,
	SkillHandgun, SkillRifle, SkillCannon,
}

// Skill は個別のスキル
type Skill struct {
	Value int  // スキル値
	Exp   Pool // 蓄積経験値。Maxに達するとスキルアップする
}

// Skills はエンティティが持つスキルセット
type Skills struct {
	Data map[SkillID]*Skill
}

// LevelUpExp はスキルアップに必要な経験値
const LevelUpExp = 100

// NewSkills は全スキルを0で初期化したSkillsを返す
func NewSkills() *Skills {
	data := make(map[SkillID]*Skill, len(AllSkillIDs))
	for _, id := range AllSkillIDs {
		data[id] = &Skill{Exp: Pool{Max: LevelUpExp}}
	}
	return &Skills{Data: data}
}

// weaponSkillMap は武器種別からスキルIDへのマッピング
var weaponSkillMap = map[string]SkillID{
	AttackSword.Type:   SkillSword,
	AttackSpear.Type:   SkillSpear,
	AttackFist.Type:    SkillFist,
	AttackHandgun.Type: SkillHandgun,
	AttackRifle.Type:   SkillRifle,
	AttackCanon.Type:   SkillCannon,
}

// WeaponSkillID は武器種別に対応するスキルIDを返す
func WeaponSkillID(at AttackType) (SkillID, bool) {
	id, ok := weaponSkillMap[at.Type]
	return id, ok
}
