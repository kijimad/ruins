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
	// 能力値0、スキル値0のとき、1回あたり10exp獲得する
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

func TestGainExp_HighAbilityGrowsFaster(t *testing.T) {
	t.Parallel()

	// 能力値8のとき: 10 * 140 / 100 = 14exp
	// 100 / 14 ≈ 8回でスキルアップ
	s := newTestSkill(0, 0)
	for i := 0; i < 8; i++ {
		GainExp(s, 8)
	}
	assert.Equal(t, 1, s.Value, "能力値8では8回以内にスキルアップするはず")
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

	// MaxLevel=100がデフォルトなので、上限なしテストにはgainExpを直接使う
	s := newTestSkill(50, 0)
	before := s.Exp.Current
	GainExp(s, 0)
	assert.Greater(t, s.Exp.Current, before, "最低1expは獲得するはず")
}

func TestGainExp_CombinedAbilityAndLevel(t *testing.T) {
	t.Parallel()

	// 能力値5、スキル値10のとき:
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

func TestGainExp_MaxLevel(t *testing.T) {
	t.Parallel()

	s := newTestSkill(100, 99)
	result := GainExp(s, 0)
	assert.False(t, result, "最大レベルに達したら経験値を獲得しない")
	assert.Equal(t, 100, s.Value, "スキル値は変わらない")
	assert.Equal(t, 99, s.Exp.Current, "経験値は変わらない")
}

func TestGainExpScaled_HalfEfficiency(t *testing.T) {
	t.Parallel()

	// efficiency=50%のとき、BaseExp=10*50/100=5exp
	s := newTestSkill(0, 0)
	GainExpScaled(s, 0, 50)
	assert.Equal(t, 5, s.Exp.Current)
}

func TestGainExpScaled_ZeroEfficiency(t *testing.T) {
	t.Parallel()

	// efficiency=0%のとき、最低1expは獲得する
	s := newTestSkill(0, 0)
	GainExpScaled(s, 0, 0)
	assert.Equal(t, 1, s.Exp.Current, "最低1expは獲得するはず")
}

func TestGainExpScaled_FullEfficiency(t *testing.T) {
	t.Parallel()

	// efficiency=100%のとき、GainExpと同じ結果になる
	s1 := newTestSkill(0, 0)
	s2 := newTestSkill(0, 0)
	GainExp(s1, 5)
	GainExpScaled(s2, 5, 100)
	assert.Equal(t, s1.Exp.Current, s2.Exp.Current)
}
