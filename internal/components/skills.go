package components

import (
	"slices"

	"github.com/kijimaD/ruins/internal/oapi"
)

// SkillID はスキルの識別子
type SkillID string

// スキルID定数。値は raw のスキーマの enum から取るので、enum に無い値はコンパイルで弾かれる
const (
	SkillSword         = SkillID(oapi.SkillIdSword)         // 刀剣
	SkillSpear         = SkillID(oapi.SkillIdSpear)         // 長物
	SkillFist          = SkillID(oapi.SkillIdFist)          // 格闘
	SkillWeightBearing = SkillID(oapi.SkillIdWeightBearing) // 荷重

	SkillBow         = SkillID(oapi.SkillIdBow)         // 弓術
	SkillHandgun     = SkillID(oapi.SkillIdHandgun)     // 拳銃
	SkillRifle       = SkillID(oapi.SkillIdRifle)       // 小銃
	SkillCannon      = SkillID(oapi.SkillIdCannon)      // 砲撃
	SkillExploration = SkillID(oapi.SkillIdExploration) // 探索

	SkillCrafting    = SkillID(oapi.SkillIdCrafting)    // クラフト
	SkillSmithing    = SkillID(oapi.SkillIdSmithing)    // 調合
	SkillNegotiation = SkillID(oapi.SkillIdNegotiation) // 交渉
	SkillMechanic    = SkillID(oapi.SkillIdMechanic)    // 機械

	SkillSprinting   = SkillID(oapi.SkillIdSprinting)   // 走破
	SkillStealth     = SkillID(oapi.SkillIdStealth)     // 隠密
	SkillNightVision = SkillID(oapi.SkillIdNightVision) // 暗視

	SkillColdResist   = SkillID(oapi.SkillIdColdResist)   // 耐寒
	SkillHungerResist = SkillID(oapi.SkillIdHungerResist) // 耐餓
	SkillHealing      = SkillID(oapi.SkillIdHealing)      // 治療

	SkillHeavyArmor    = SkillID(oapi.SkillIdHeavyArmor)    // 重装
	SkillFireResist    = SkillID(oapi.SkillIdFireResist)    // 耐火
	SkillThunderResist = SkillID(oapi.SkillIdThunderResist) // 耐電
	SkillChillResist   = SkillID(oapi.SkillIdChillResist)   // 耐冷
	SkillPhotonResist  = SkillID(oapi.SkillIdPhotonResist)  // 耐光
)

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

// Get は指定されたスキルIDのSkillを返す。まだ無ければ値0のスキルを作って返す
func (s *Skills) Get(id SkillID) *Skill {
	if s.Data == nil {
		s.Data = make(map[SkillID]*Skill)
	}
	sk, ok := s.Data[id]
	if !ok {
		sk = &Skill{Exp: IntPool{Max: LevelUpExp}}
		s.Data[id] = sk
	}
	return sk
}

// LevelUpExp はスキルアップに必要な経験値
const LevelUpExp = 100

// NewSkills はスキルを持たない Skills を返す。スキルは Get で初めて読んだときに値0で作る
func NewSkills() *Skills {
	return &Skills{Data: make(map[SkillID]*Skill)}
}

// WeaponSkillID は武器種別に対応するスキルIDを返す。武器種別はそれと同じ名前の武器スキルで上達する
func WeaponSkillID(at AttackType) (SkillID, bool) {
	id := SkillID(at.Type)
	return id, slices.Contains(WeaponSkillIDs, id)
}
