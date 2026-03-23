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

// skillAbility はスキルIDから対応する能力値IDへのマッピング。
// 能力値が高いほど、対応するスキルの成長が速くなる。
var skillAbility = map[SkillID]AbilityID{
	SkillSword:         AblSTR,
	SkillSpear:         AblSTR,
	SkillFist:          AblSTR,
	SkillWeightBearing: AblSTR,

	SkillHandgun:     AblSEN,
	SkillRifle:       AblSEN,
	SkillCannon:      AblSEN,
	SkillExploration: AblSEN,

	SkillCrafting:    AblDEX,
	SkillSmithing:    AblDEX,
	SkillNegotiation: AblDEX,

	SkillSprinting:   AblAGI,
	SkillStealth:     AblAGI,
	SkillNightVision: AblAGI,

	SkillColdResist:   AblVIT,
	SkillHeatResist:   AblVIT,
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
		panic(fmt.Sprintf("スキル%qに対応する能力値が定義されていません", id))
	}
	return ablID
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

// SkillInfo はスキルの詳細情報
type SkillInfo struct {
	Summary  string // 概要文
	GainedBy string // 獲得条件
	Effect   string // 効果
}

// SkillDescription はスキルの詳細情報を返す
var SkillDescription = map[SkillID]SkillInfo{
	SkillSword:         {Summary: "刀剣類の扱いに関する技術", GainedBy: "刀剣で攻撃すると上がる", Effect: "刀剣のダメージと命中が上昇する"},
	SkillSpear:         {Summary: "槍や棒などの長物を扱う技術", GainedBy: "長物で攻撃すると上がる", Effect: "長物のダメージと命中が上昇する"},
	SkillFist:          {Summary: "素手や拳で戦う技術", GainedBy: "格闘で攻撃すると上がる", Effect: "格闘のダメージと命中が上昇する"},
	SkillWeightBearing: {Summary: "重い荷物を運ぶ能力", GainedBy: "重い装備で行動すると上がる", Effect: "最大荷重が上昇する"},
	SkillHandgun:       {Summary: "拳銃の射撃技術", GainedBy: "拳銃で攻撃すると上がる", Effect: "拳銃のダメージと命中が上昇する"},
	SkillRifle:         {Summary: "小銃の射撃技術", GainedBy: "小銃で攻撃すると上がる", Effect: "小銃のダメージと命中が上昇する"},
	SkillCannon:        {Summary: "大型火器の運用技術", GainedBy: "砲撃で攻撃すると上がる", Effect: "砲撃のダメージと命中が上昇する"},
	SkillExploration:   {Summary: "未知の場所を調査する技術", GainedBy: "装備や本で上がる", Effect: "アイテム発見率が上昇する"},
	SkillCrafting:      {Summary: "素材からアイテムを作る技術", GainedBy: "アイテムを合成すると上がる", Effect: "合成時の素材消費が減少する"},
	SkillSmithing:      {Summary: "素材を精製・調合する技術", GainedBy: "素材を調合すると上がる", Effect: "調合時の品質が上昇する"},
	SkillNegotiation:   {Summary: "有利な取引をする話術", GainedBy: "取引を行うと上がる", Effect: "売買の価格が有利になる"},
	SkillSprinting:     {Summary: "長距離を素早く移動する能力", GainedBy: "装備や本で上がる", Effect: "移動時のAPコストが減少する"},
	SkillStealth:       {Summary: "敵に気づかれずに行動する技術", GainedBy: "装備や本で上がる", Effect: "敵に発見される距離が短くなる"},
	SkillNightVision:   {Summary: "暗所での視認能力", GainedBy: "装備や本で上がる", Effect: "暗所での視界が広がる"},
	SkillColdResist:    {Summary: "寒さへの耐性", GainedBy: "装備や本で上がる", Effect: "低体温の進行が遅くなる"},
	SkillHeatResist:    {Summary: "暑さへの耐性", GainedBy: "装備や本で上がる", Effect: "高体温の進行が遅くなる"},
	SkillHungerResist:  {Summary: "空腹への耐性", GainedBy: "装備や本で上がる", Effect: "空腹の進行が遅くなる"},
	SkillHealing:       {Summary: "傷を治す医療技術", GainedBy: "回復アイテムを使用すると上がる", Effect: "回復アイテムの効果が上昇する"},
	SkillHeavyArmor:    {Summary: "重い防具を着こなす技術", GainedBy: "重装備で被弾すると上がる", Effect: "最大荷重が上昇する"},
	SkillFireResist:    {Summary: "火への耐性", GainedBy: "装備や本で上がる", Effect: "火属性ダメージが軽減される"},
	SkillThunderResist: {Summary: "雷への耐性", GainedBy: "装備や本で上がる", Effect: "雷属性ダメージが軽減される"},
	SkillChillResist:   {Summary: "氷への耐性", GainedBy: "装備や本で上がる", Effect: "氷属性ダメージが軽減される"},
	SkillPhotonResist:  {Summary: "光への耐性", GainedBy: "装備や本で上がる", Effect: "光属性ダメージが軽減される"},
}

// SkillCategory はスキルのカテゴリを表す
type SkillCategory struct {
	Name string    // カテゴリの表示名
	IDs  []SkillID // カテゴリに属するスキルID
}

// SkillCategories はカテゴリごとにグループ化されたスキル定義。
// 表示順序はこのスライスの順序に従う。
var SkillCategories = []SkillCategory{
	{Name: "近接", IDs: []SkillID{SkillSword, SkillSpear, SkillFist}},
	{Name: "射撃", IDs: []SkillID{SkillHandgun, SkillRifle, SkillCannon}},
	{Name: "技巧", IDs: []SkillID{SkillCrafting, SkillSmithing, SkillNegotiation}},
	{Name: "機動", IDs: []SkillID{SkillSprinting, SkillStealth, SkillNightVision, SkillWeightBearing}},
	{Name: "生存", IDs: []SkillID{SkillColdResist, SkillHeatResist, SkillHungerResist, SkillHealing, SkillExploration}},
	{Name: "防御", IDs: []SkillID{SkillHeavyArmor, SkillFireResist, SkillThunderResist, SkillChillResist, SkillPhotonResist}},
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
