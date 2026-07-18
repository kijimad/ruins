package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBook_IsCompleted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		book     Book
		expected bool
	}{
		{"未読", Book{Effort: IntPool{Max: 10, Current: 0}}, false},
		{"途中", Book{Effort: IntPool{Max: 10, Current: 5}}, false},
		{"読了", Book{Effort: IntPool{Max: 10, Current: 10}}, true},
		{"超過", Book{Effort: IntPool{Max: 10, Current: 15}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.book.IsCompleted())
		})
	}
}

func TestBook_CanRead(t *testing.T) {
	t.Parallel()

	t.Run("読了済みの本は読めない", func(t *testing.T) {
		t.Parallel()
		b := &Book{Effort: IntPool{Max: 10, Current: 10}}
		assert.Error(t, b.CanRead(nil))
	})

	t.Run("スキル条件なしの本は誰でも読める", func(t *testing.T) {
		t.Parallel()
		b := &Book{Effort: IntPool{Max: 10, Current: 0}}
		assert.NoError(t, b.CanRead(nil))
	})

	t.Run("RequiredLevel=0の本は誰でも読める", func(t *testing.T) {
		t.Parallel()
		b := &Book{
			Effort: IntPool{Max: 10, Current: 0},
			Skill:  &SkillBookEffect{TargetSkill: SkillSword, RequiredLevel: 0},
		}
		assert.NoError(t, b.CanRead(nil))
	})

	t.Run("スキルが足りないと読めない", func(t *testing.T) {
		t.Parallel()
		b := &Book{
			Effort: IntPool{Max: 10, Current: 0},
			Skill:  &SkillBookEffect{TargetSkill: SkillSword, RequiredLevel: 3},
		}
		skills := NewSkills()
		assert.Error(t, b.CanRead(skills))
	})

	t.Run("スキルが足りていれば読める", func(t *testing.T) {
		t.Parallel()
		b := &Book{
			Effort: IntPool{Max: 10, Current: 0},
			Skill:  &SkillBookEffect{TargetSkill: SkillSword, RequiredLevel: 1},
		}
		skills := NewSkills()
		skills.Get(SkillSword).Value = 5
		assert.NoError(t, b.CanRead(skills))
	})
}

func TestReadingEfficiency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		playerLevel int
		bookLevel   int
		expected    int
	}{
		// 最適（同レベル）
		{"同レベル", 3, 3, 100},

		// 本が難しい側
		{"本が1レベル上", 3, 4, 90},
		{"本が3レベル上", 3, 6, 70},
		{"本が5レベル上", 3, 8, 50},
		{"本が6レベル上で理解不能", 3, 9, 0},
		{"本が10レベル上で理解不能", 3, 13, 0},

		// 本が易しい側
		{"本が1レベル下", 3, 2, 82},
		{"本が3レベル下", 3, 0, 46},
		{"本が5レベル下で最低効率", 5, 0, 10},
		{"本が10レベル下で最低効率", 10, 0, 10},

		// エッジケース
		{"両方0", 0, 0, 100},
		{"プレイヤー0で本が5", 0, 5, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ReadingEfficiency(tt.playerLevel, tt.bookLevel)
			assert.Equal(t, tt.expected, int(result))
		})
	}
}
