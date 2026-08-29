package components

import "fmt"

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

	SkillCrafting    SkillID = "crafting"    // 合成
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

// skillAbility はスキルIDから対応する能力値IDへのマッピング。
// 能力値が高いほど、対応するスキルの成長が速くなる。
var skillAbility = map[SkillID]AbilityID{
	SkillSword:         AblSTR,
	SkillSpear:         AblSTR,
	SkillFist:          AblSTR,
	SkillWeightBearing: AblSTR,

	SkillBow:         AblSEN,
	SkillHandgun:     AblSEN,
	SkillRifle:       AblSEN,
	SkillCannon:      AblSEN,
	SkillExploration: AblSEN,

	SkillCrafting:    AblDEX,
	SkillSmithing:    AblDEX,
	SkillNegotiation: AblDEX,
	SkillMechanic:    AblDEX,

	SkillSprinting:   AblAGI,
	SkillStealth:     AblAGI,
	SkillNightVision: AblAGI,

	SkillColdResist:   AblVIT,
	SkillHungerResist: AblVIT,
	SkillHealing:      AblVIT,

	SkillHeavyArmor:    AblDEF,
	SkillFireResist:    AblDEF,
	SkillThunderResist: AblDEF,
	SkillChillResist:   AblDEF,
	SkillPhotonResist:  AblDEF,
}

// SkillAbilityID はスキルに対応する能力値IDを返す
func SkillAbilityID(id SkillID) AbilityID {
	ablID, ok := skillAbility[id]
	if !ok {
		panic(fmt.Sprintf("no ability defined for skill %q", id))
	}
	return ablID
}

// skillName はスキルIDの表示名マップ
var skillName = map[SkillID]string{
	SkillSword:         "Swordsmanship",
	SkillSpear:         "Polearm",
	SkillFist:          "Unarmed",
	SkillWeightBearing: "Load bearing",
	SkillBow:           "Archery",
	SkillHandgun:       "Pistol skill",
	SkillRifle:         "Rifle skill",
	SkillCannon:        "Artillery",
	SkillExploration:   "Exploration",
	SkillCrafting:      "Crafting",
	SkillSmithing:      "Smithing",
	SkillNegotiation:   "Negotiation",
	SkillMechanic:      "Mechanic",
	SkillSprinting:     "Sprinting",
	SkillStealth:       "Stealth",
	SkillNightVision:   "Night vision",
	SkillColdResist:    "Cold resistance",
	SkillHungerResist:  "Hunger resistance",
	SkillHealing:       "Healing",
	SkillHeavyArmor:    "Heavy armor",
	SkillFireResist:    "Fire resistance",
	SkillThunderResist: "Thunder resistance",
	SkillChillResist:   "Chill resistance",
	SkillPhotonResist:  "Photon resistance",
}

// SkillName はスキルIDの表示名を返す。未定義ならpanicする
func SkillName(id SkillID) string {
	name, ok := skillName[id]
	if !ok {
		panic(fmt.Sprintf("undefined skill ID: %q", id))
	}
	return name
}

// HasSkillName はスキルIDの表示名が定義されているかを返す。ロード時のバリデーション用
func HasSkillName(id SkillID) bool {
	_, ok := skillName[id]
	return ok
}

// SkillInfo はスキルの詳細情報
type SkillInfo struct {
	Summary  string // 概要文
	GainedBy string // 獲得条件
	Effect   string // 効果
}

// gainedByEquipmentOrBook は装備や本で上昇するスキルの習得条件。多数のスキルで共通する
const gainedByEquipmentOrBook = "Raised by equipment or books"

// skillDescription はスキルIDの詳細情報マップ
var skillDescription = map[SkillID]SkillInfo{
	SkillSword:         {Summary: "Technique for handling swords", GainedBy: "Raised by attacking with swords", Effect: "Increases sword damage and accuracy"},
	SkillSpear:         {Summary: "Technique for handling polearms like spears and staves", GainedBy: "Raised by attacking with polearms", Effect: "Increases polearm damage and accuracy"},
	SkillFist:          {Summary: "Technique for fighting with fists", GainedBy: "Raised by attacking unarmed", Effect: "Increases unarmed damage and accuracy"},
	SkillWeightBearing: {Summary: "Ability to carry heavy loads", GainedBy: "Raised by acting with heavy equipment", Effect: "Increases maximum load"},
	SkillBow:           {Summary: "Technique for handling bows", GainedBy: "Raised by attacking with bows", Effect: "Increases bow damage and accuracy"},
	SkillHandgun:       {Summary: "Pistol shooting technique", GainedBy: "Raised by attacking with pistols", Effect: "Increases pistol damage and accuracy"},
	SkillRifle:         {Summary: "Rifle shooting technique", GainedBy: "Raised by attacking with rifles", Effect: "Increases rifle damage and accuracy"},
	SkillCannon:        {Summary: "Technique for operating large firearms", GainedBy: "Raised by attacking with artillery", Effect: "Increases artillery damage and accuracy"},
	SkillExploration:   {Summary: "Technique for surveying unknown places", GainedBy: gainedByEquipmentOrBook, Effect: "Increases item discovery rate"},
	SkillCrafting:      {Summary: "Technique for making items from materials", GainedBy: "Raised by crafting items", Effect: "Reduces material consumption when crafting"},
	SkillSmithing:      {Summary: "Technique for refining and blending materials", GainedBy: "Raised by smithing materials", Effect: "Increases quality when smithing"},
	SkillNegotiation:   {Summary: "Persuasion for favorable deals", GainedBy: "Raised by trading", Effect: "Improves buying and selling prices"},
	SkillMechanic:      {Summary: "Technique for understanding and repairing machines", GainedBy: "Raised by disassembling or reading mechanic books", Effect: "Speeds up disassembly and increases yield"},
	SkillSprinting:     {Summary: "Ability to move quickly over long distances", GainedBy: gainedByEquipmentOrBook, Effect: "Reduces AP cost when moving"},
	SkillStealth:       {Summary: "Technique for acting unnoticed by enemies", GainedBy: gainedByEquipmentOrBook, Effect: "Shortens the distance at which enemies detect you"},
	SkillNightVision:   {Summary: "Ability to see in the dark", GainedBy: gainedByEquipmentOrBook, Effect: "Widens vision in the dark"},
	SkillColdResist:    {Summary: "Resistance to cold", GainedBy: gainedByEquipmentOrBook, Effect: "Slows hypothermia progress"},
	SkillHungerResist:  {Summary: "Resistance to hunger", GainedBy: gainedByEquipmentOrBook, Effect: "Slows hunger progress"},
	SkillHealing:       {Summary: "Medical technique for healing wounds", GainedBy: "Raised by using healing items", Effect: "Increases healing item effect"},
	SkillHeavyArmor:    {Summary: "Technique for wearing heavy armor", GainedBy: "Raised by taking hits in heavy armor", Effect: "Increases maximum load"},
	SkillFireResist:    {Summary: "Resistance to fire", GainedBy: gainedByEquipmentOrBook, Effect: "Reduces fire element damage"},
	SkillThunderResist: {Summary: "Resistance to thunder", GainedBy: gainedByEquipmentOrBook, Effect: "Reduces thunder element damage"},
	SkillChillResist:   {Summary: "Resistance to ice", GainedBy: gainedByEquipmentOrBook, Effect: "Reduces ice element damage"},
	SkillPhotonResist:  {Summary: "Resistance to light", GainedBy: gainedByEquipmentOrBook, Effect: "Reduces light element damage"},
}

// SkillDescription はスキルIDの詳細情報を返す。未定義ならpanicする
func SkillDescription(id SkillID) SkillInfo {
	info, ok := skillDescription[id]
	if !ok {
		panic(fmt.Sprintf("undefined skill ID: %q", id))
	}
	return info
}

// SkillCategory はスキルのカテゴリを表す
type SkillCategory struct {
	Name string    // カテゴリの表示名
	IDs  []SkillID // カテゴリに属するスキルID
}

// SkillCategories はカテゴリごとにグループ化されたスキル定義。
// 表示順序はこのスライスの順序に従う。
var SkillCategories = []SkillCategory{
	{Name: "Melee", IDs: []SkillID{SkillSword, SkillSpear, SkillFist}},
	{Name: "Ranged", IDs: []SkillID{SkillBow, SkillHandgun, SkillRifle, SkillCannon}},
	{Name: "Craft", IDs: []SkillID{SkillCrafting, SkillSmithing, SkillNegotiation, SkillMechanic}},
	{Name: "Mobility", IDs: []SkillID{SkillSprinting, SkillStealth, SkillNightVision, SkillWeightBearing}},
	{Name: "Survival", IDs: []SkillID{SkillColdResist, SkillHungerResist, SkillHealing, SkillExploration}},
	{Name: "Defense", IDs: []SkillID{SkillHeavyArmor, SkillFireResist, SkillThunderResist, SkillChillResist, SkillPhotonResist}},
}

// AllSkillIDs は定義済みの全SkillIDのリスト。
// SkillCategoriesの順序に従う。
var AllSkillIDs = func() []SkillID {
	var ids []SkillID
	for _, cat := range SkillCategories {
		ids = append(ids, cat.IDs...)
	}
	return ids
}()

// weaponSkillIDs は武器に対応するスキルIDのリスト
var weaponSkillIDs = []SkillID{
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

// weaponSkillMap は武器種別からスキルIDへのマッピング
var weaponSkillMap = map[string]SkillID{
	AttackSword.Type:   SkillSword,
	AttackSpear.Type:   SkillSpear,
	AttackFist.Type:    SkillFist,
	AttackBow.Type:     SkillBow,
	AttackHandgun.Type: SkillHandgun,
	AttackRifle.Type:   SkillRifle,
	AttackCanon.Type:   SkillCannon,
}

// WeaponSkillID は武器種別に対応するスキルIDを返す
func WeaponSkillID(at AttackType) (SkillID, bool) {
	id, ok := weaponSkillMap[at.Type]
	return id, ok
}
