package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestMaxHP(t *testing.T) {
	t.Parallel()
	t.Run("calculate max HP with base stats", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     10,
				Modifier: 0,
				Total:    10,
			},
			Strength: gc.Attribute{
				Base:     5,
				Modifier: 0,
				Total:    5,
			},
			Sensation: gc.Attribute{
				Base:     3,
				Modifier: 0,
				Total:    3,
			},
		}
		result := maxHP(attrs)
		// 30 + (10*8 + 5 + 3) = 30 + 88 = 118
		expected := 118
		assert.Equal(t, expected, result, "maxHPの計算が正しくない")
	})

	t.Run("calculate max HP with level bonus", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     10,
				Modifier: 0,
				Total:    10,
			},
			Strength: gc.Attribute{
				Base:     5,
				Modifier: 0,
				Total:    5,
			},
			Sensation: gc.Attribute{
				Base:     3,
				Modifier: 0,
				Total:    3,
			},
		}
		result := maxHP(attrs)
		// 30 + (10*8 + 5 + 3) = 30 + 88 = 118
		expected := 118
		assert.Equal(t, expected, result, "レベルボーナス込みのmaxHPの計算が正しくない")
	})

	t.Run("calculate max HP with high stats", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     20,
				Modifier: 5,
				Total:    25,
			},
			Strength: gc.Attribute{
				Base:     15,
				Modifier: 3,
				Total:    18,
			},
			Sensation: gc.Attribute{
				Base:     10,
				Modifier: 2,
				Total:    12,
			},
		}
		result := maxHP(attrs)
		// 30 + (25*8 + 18 + 12) = 30 + 230 = 260
		expected := 260
		assert.Equal(t, expected, result, "高ステータスでのmaxHPの計算が正しくない")
	})
}

func TestMaxSP(t *testing.T) {
	t.Parallel()
	t.Run("calculate max SP with base stats", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     10,
				Modifier: 0,
				Total:    10,
			},
			Dexterity: gc.Attribute{
				Base:     8,
				Modifier: 0,
				Total:    8,
			},
			Agility: gc.Attribute{
				Base:     6,
				Modifier: 0,
				Total:    6,
			},
		}
		result := maxSP(attrs)
		// 10*2 + 8 + 6 = 20 + 8 + 6 = 34
		expected := 34
		assert.Equal(t, expected, result, "maxSPの計算が正しくない")
	})

	t.Run("calculate max SP with level bonus", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     10,
				Modifier: 0,
				Total:    10,
			},
			Dexterity: gc.Attribute{
				Base:     8,
				Modifier: 0,
				Total:    8,
			},
			Agility: gc.Attribute{
				Base:     6,
				Modifier: 0,
				Total:    6,
			},
		}
		result := maxSP(attrs)
		// 10*2 + 8 + 6 = 20 + 8 + 6 = 34
		expected := 34
		assert.Equal(t, expected, result, "maxSPの計算が正しくない")
	})

	t.Run("calculate max SP with high stats", func(t *testing.T) {
		t.Parallel()
		attrs := &gc.Attributes{
			Vitality: gc.Attribute{
				Base:     20,
				Modifier: 5,
				Total:    25,
			},
			Dexterity: gc.Attribute{
				Base:     15,
				Modifier: 3,
				Total:    18,
			},
			Agility: gc.Attribute{
				Base:     12,
				Modifier: 2,
				Total:    14,
			},
		}
		result := maxSP(attrs)
		// 25*2 + 18 + 14 = 50 + 18 + 14 = 82
		expected := 82
		assert.Equal(t, expected, result, "高ステータスでのmaxSPの計算が正しくない")
	})
}
