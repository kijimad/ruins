package components

import (
	"github.com/kijimaD/ruins/internal/rawschema"
)

// Skill は個別のスキル
type Skill struct {
	Value int     // スキル値
	Exp   IntPool // 蓄積経験値。Maxに達するとスキルアップする
}

// Skills はエンティティが持つスキルセット
type Skills struct {
	Data map[rawschema.SkillId]*Skill
}

// Level はスキルの値を返す。まだ無いスキルは0。読むだけで Data へ書き込まないので、並行に読んでよい
func (s *Skills) Level(id rawschema.SkillId) int {
	if sk, ok := s.Data[id]; ok {
		return sk.Value
	}
	return 0
}

// Get は指定されたスキルIDのSkillを返す。まだ無ければ値0のスキルを作って Data に足す。書き換えるときに使う
func (s *Skills) Get(id rawschema.SkillId) *Skill {
	if s.Data == nil {
		s.Data = make(map[rawschema.SkillId]*Skill)
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
	return &Skills{Data: make(map[rawschema.SkillId]*Skill)}
}
