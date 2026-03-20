package skill

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func newTestSkill(value int, exp int) *gc.Skill {
	return &gc.Skill{Value: value, Exp: gc.Pool{Max: gc.LevelUpExp, Current: exp}}
}

func TestGainExp_SkillLevelUp(t *testing.T) {
	t.Parallel()

	s := newTestSkill(0, 0)
	// 属性値0、スキル値0のとき、1回あたり10exp獲得する
	// 100 / 10 = 10回でスキルアップ
	leveledUp := false
	for i := 0; i < 10; i++ {
		if GainExp(s, 0) {
			leveledUp = true
		}
	}
	assert.True(t, leveledUp, "10回の経験値獲得でスキルアップするはず")
	assert.Equal(t, 1, s.Value)
}

func TestGainExp_HighAttributeGrowsFaster(t *testing.T) {
	t.Parallel()

	// 属性値8のとき: 10 * 140 / 100 = 14exp
	// 100 / 14 ≈ 8回でスキルアップ
	s := newTestSkill(0, 0)
	for i := 0; i < 8; i++ {
		GainExp(s, 8)
	}
	assert.Equal(t, 1, s.Value, "属性値8では8回以内にスキルアップするはず")
}

func TestGainExp_DecaysWithLevel(t *testing.T) {
	t.Parallel()

	// スキル値5のとき: 10 * 100 / 100 * 100 / 200 = 5exp
	// 100 / 5 = 20回でスキルアップ
	s := newTestSkill(5, 0)

	count := 0
	for s.Value == 5 {
		GainExp(s, 0)
		count++
	}
	assert.Equal(t, 20, count, "スキル値5では20回でスキルアップするはず")
}

func TestGainExp_MinimumExpIsOne(t *testing.T) {
	t.Parallel()

	// 非常に高いスキル値でも最低1expは獲得する
	s := newTestSkill(100, 0)
	before := s.Exp.Current
	GainExp(s, 0)
	assert.Greater(t, s.Exp.Current, before, "最低1expは獲得するはず")
}

func TestGainExp_CombinedAttributeAndLevel(t *testing.T) {
	t.Parallel()

	// 属性値5、スキル値10のとき:
	// 10 * 125 / 100 * 100 / 300 = 12 * 100 / 300 = 4exp
	s := newTestSkill(10, 0)
	before := s.Exp.Current
	GainExp(s, 5)
	gained := s.Exp.Current - before
	assert.Equal(t, 4, gained)
}

func TestGainExp_ExpCarriesOver(t *testing.T) {
	t.Parallel()

	// レベルアップ時に余剰expは繰り越される
	s := newTestSkill(0, 95)
	GainExp(s, 0) // +10 → 105 → レベルアップ、残り5
	assert.Equal(t, 1, s.Value)
	assert.Equal(t, 5, s.Exp.Current, "余剰expは繰り越されるはず")
}
