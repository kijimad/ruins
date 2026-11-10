package components

import (
	"github.com/kijimaD/ruins/internal/oapi"
)

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

// NewSkills はスキルを持たない Skills を返す
func NewSkills() *Skills {
	return &Skills{Data: make(map[oapi.SkillId]*Skill)}
}
