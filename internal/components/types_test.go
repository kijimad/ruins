package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {
	t.Parallel()
	t.Run("create pool", func(t *testing.T) {
		t.Parallel()
		pool := Pool{
			Max:     100,
			Current: 75,
		}
		assert.Equal(t, 100, pool.Max, "最大値が正しく設定されない")
		assert.Equal(t, 75, pool.Current, "現在値が正しく設定されない")
	})

	t.Run("empty pool", func(t *testing.T) {
		t.Parallel()
		pool := Pool{
			Max:     50,
			Current: 0,
		}
		assert.Equal(t, 50, pool.Max, "最大値が正しく設定されない")
		assert.Equal(t, 0, pool.Current, "空のプールの現在値が正しくない")
	})

	t.Run("full pool", func(t *testing.T) {
		t.Parallel()
		pool := Pool{
			Max:     200,
			Current: 200,
		}
		assert.Equal(t, pool.Max, pool.Current, "満タンのプールで最大値と現在値が一致しない")
	})
}

func TestAttribute(t *testing.T) {
	t.Parallel()
	t.Run("create attribute", func(t *testing.T) {
		t.Parallel()
		attr := Attribute{
			Base:     10,
			Modifier: 5,
			Total:    15,
		}
		assert.Equal(t, 10, attr.Base, "基本値が正しく設定されない")
		assert.Equal(t, 5, attr.Modifier, "修正値が正しく設定されない")
		assert.Equal(t, 15, attr.Total, "合計値が正しく設定されない")
	})

	t.Run("negative modifier", func(t *testing.T) {
		t.Parallel()
		attr := Attribute{
			Base:     20,
			Modifier: -5,
			Total:    15,
		}
		assert.Equal(t, 20, attr.Base, "基本値が正しく設定されない")
		assert.Equal(t, -5, attr.Modifier, "負の修正値が正しく設定されない")
		assert.Equal(t, 15, attr.Total, "負の修正値を含む合計値が正しくない")
	})

	t.Run("zero values", func(t *testing.T) {
		t.Parallel()
		attr := Attribute{
			Base:     0,
			Modifier: 0,
			Total:    0,
		}
		assert.Equal(t, 0, attr.Base, "ゼロの基本値が正しく設定されない")
		assert.Equal(t, 0, attr.Modifier, "ゼロの修正値が正しく設定されない")
		assert.Equal(t, 0, attr.Total, "ゼロの合計値が正しく設定されない")
	})
}

func TestRecipeInput(t *testing.T) {
	t.Parallel()
	t.Run("create recipe input", func(t *testing.T) {
		t.Parallel()
		input := RecipeInput{
			Name:   "鉄",
			Amount: 3,
		}
		assert.Equal(t, "鉄", input.Name, "素材名が正しく設定されない")
		assert.Equal(t, 3, input.Amount, "必要量が正しく設定されない")
	})

	t.Run("empty name", func(t *testing.T) {
		t.Parallel()
		input := RecipeInput{
			Name:   "",
			Amount: 1,
		}
		assert.Equal(t, "", input.Name, "空の素材名が正しく設定されない")
		assert.Equal(t, 1, input.Amount, "必要量が正しく設定されない")
	})
}

func TestEquipBonus(t *testing.T) {
	t.Parallel()
	t.Run("create equip bonus", func(t *testing.T) {
		t.Parallel()
		bonus := EquipBonus{
			Vitality:  5,
			Strength:  3,
			Sensation: 2,
			Dexterity: 1,
			Agility:   4,
		}
		assert.Equal(t, 5, bonus.Vitality, "体力ボーナスが正しく設定されない")
		assert.Equal(t, 3, bonus.Strength, "筋力ボーナスが正しく設定されない")
		assert.Equal(t, 2, bonus.Sensation, "感覚ボーナスが正しく設定されない")
		assert.Equal(t, 1, bonus.Dexterity, "器用ボーナスが正しく設定されない")
		assert.Equal(t, 4, bonus.Agility, "敏捷ボーナスが正しく設定されない")
	})

	t.Run("negative bonuses", func(t *testing.T) {
		t.Parallel()
		bonus := EquipBonus{
			Vitality:  -2,
			Strength:  -1,
			Sensation: 0,
			Dexterity: 3,
			Agility:   -4,
		}
		assert.Equal(t, -2, bonus.Vitality, "負の体力ボーナスが正しく設定されない")
		assert.Equal(t, -1, bonus.Strength, "負の筋力ボーナスが正しく設定されない")
		assert.Equal(t, 0, bonus.Sensation, "ゼロの感覚ボーナスが正しく設定されない")
		assert.Equal(t, 3, bonus.Dexterity, "正の器用ボーナスが正しく設定されない")
		assert.Equal(t, -4, bonus.Agility, "負の敏捷ボーナスが正しく設定されない")
	})
}

func TestTargetType(t *testing.T) {
	t.Parallel()
	t.Run("create target type", func(t *testing.T) {
		t.Parallel()
		target := TargetType{
			TargetGroup: TargetGroupEnemy,
			TargetNum:   TargetSingle,
		}
		assert.Equal(t, TargetGroupEnemy, target.TargetGroup, "対象グループが正しく設定されない")
		assert.Equal(t, TargetSingle, target.TargetNum, "対象数が正しく設定されない")
	})

	t.Run("various target combinations", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name        string
			targetGroup TargetGroupType
			targetNum   TargetNumType
		}{
			{"single enemy", TargetGroupEnemy, TargetSingle},
			{"all enemies", TargetGroupEnemy, TargetAll},
			{"single ally", TargetGroupAlly, TargetSingle},
			{"all allies", TargetGroupAlly, TargetAll},
			{"single weapon", TargetGroupWeapon, TargetSingle},
			{"none target", TargetGroupNone, TargetSingle},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				target := TargetType{
					TargetGroup: tt.targetGroup,
					TargetNum:   tt.targetNum,
				}
				assert.Equal(t, tt.targetGroup, target.TargetGroup, "対象グループが一致しない")
				assert.Equal(t, tt.targetNum, target.TargetNum, "対象数が一致しない")
			})
		}
	})
}

func TestEquipmentSlotNumber(t *testing.T) {
	t.Parallel()
	t.Run("create slot numbers", func(t *testing.T) {
		t.Parallel()
		slot0 := EquipmentSlotNumber(0)
		slot1 := EquipmentSlotNumber(1)
		slot7 := EquipmentSlotNumber(7)

		assert.Equal(t, EquipmentSlotNumber(0), slot0, "スロット0が正しく設定されない")
		assert.Equal(t, EquipmentSlotNumber(1), slot1, "スロット1が正しく設定されない")
		assert.Equal(t, EquipmentSlotNumber(7), slot7, "スロット7が正しく設定されない")
	})
}

func TestRatioAmount(t *testing.T) {
	t.Parallel()
	t.Run("create ratio amount", func(t *testing.T) {
		t.Parallel()
		ratio := RatioAmount{Ratio: 0.5}
		assert.Equal(t, 0.5, ratio.Ratio, "倍率が正しく設定されない")

		// Amounterインターフェースを実装していることを確認
		var amounter Amounter = ratio
		assert.NotNil(t, amounter, "Amounterインターフェースを実装していない")
	})

	t.Run("calc with ratio", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			ratio    float64
			base     int
			expected int
		}{
			{"50% of 100", 0.5, 100, 50},
			{"150% of 100", 1.5, 100, 150},
			{"25% of 80", 0.25, 80, 20},
			{"100% of 50", 1.0, 50, 50},
			{"0% of 100", 0.0, 100, 0},
			{"50% of 0", 0.5, 0, 0},
			{"33.3% of 100", 0.333, 100, 33}, // 切り捨て
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ratio := RatioAmount{Ratio: tt.ratio}
				result := ratio.Calc(tt.base)
				assert.Equal(t, tt.expected, result, "計算結果が正しくない")
			})
		}
	})

	t.Run("Amount method exists", func(t *testing.T) {
		t.Parallel()
		ratio := RatioAmount{Ratio: 0.75}
		ratio.Amount() // メソッドが存在することを確認
	})
}

func TestNumeralAmount(t *testing.T) {
	t.Parallel()
	t.Run("create numeral amount", func(t *testing.T) {
		t.Parallel()
		numeral := NumeralAmount{Numeral: 25}
		assert.Equal(t, 25, numeral.Numeral, "絶対量が正しく設定されない")

		// Amounterインターフェースを実装していることを確認
		var amounter Amounter = numeral
		assert.NotNil(t, amounter, "Amounterインターフェースを実装していない")
	})

	t.Run("calc numeral", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			numeral  int
			expected int
		}{
			{"positive value", 100, 100},
			{"zero value", 0, 0},
			{"negative value", -50, -50},
			{"large value", 9999, 9999},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				numeral := NumeralAmount{Numeral: tt.numeral}
				result := numeral.Calc()
				assert.Equal(t, tt.expected, result, "固定値の計算結果が正しくない")
			})
		}
	})

	t.Run("Amount method exists", func(t *testing.T) {
		t.Parallel()
		numeral := NumeralAmount{Numeral: 50}
		numeral.Amount() // メソッドが存在することを確認
	})
}

func TestAmounterInterface(t *testing.T) {
	t.Parallel()
	t.Run("all types implement Amounter", func(t *testing.T) {
		t.Parallel()
		// 両方の型がAmounterインターフェースを実装していることを確認
		var amounters = []Amounter{
			RatioAmount{Ratio: 0.5},
			NumeralAmount{Numeral: 100},
		}

		assert.Len(t, amounters, 2, "すべてのAmounter実装が確認されていない")

		for i, amounter := range amounters {
			assert.NotNil(t, amounter, "Amounter %dがnilである", i)
			amounter.Amount() // Amountメソッドを呼び出せることを確認
		}
	})
}
