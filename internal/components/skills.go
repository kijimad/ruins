package components

import (
	"slices"

	"github.com/kijimaD/ruins/internal/oapi"
)

// WeaponSkillIDs は武器に対応するスキルIDのリスト
var WeaponSkillIDs = []oapi.SkillId{
	oapi.SkillIdSword, oapi.SkillIdSpear, oapi.SkillIdFist,
	oapi.SkillIdBow, oapi.SkillIdHandgun, oapi.SkillIdRifle, oapi.SkillIdCannon,
}

// Skill は個別のスキル
type Skill struct {
	Value int     // スキル値
	Exp   IntPool // 蓄積経験値。Maxに達するとスキルアップする
}

// Skills はエンティティが持つスキルセット
type Skills struct {
	Data map[oapi.SkillId]*Skill
}

// Get は指定されたスキルIDのSkillを返す。まだ無ければ値0のスキルを作って返す
func (s *Skills) Get(id oapi.SkillId) *Skill {
	if s.Data == nil {
		s.Data = make(map[oapi.SkillId]*Skill)
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
	return &Skills{Data: make(map[oapi.SkillId]*Skill)}
}

// WeaponSkillID は武器種別に対応するスキルIDを返す。武器種別はそれと同じ名前の武器スキルで上達する
func WeaponSkillID(at AttackType) (oapi.SkillId, bool) {
	id := oapi.SkillId(at.Type)
	return id, slices.Contains(WeaponSkillIDs, id)
}
