package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {
	t.Parallel()
	t.Run("create pool", func(t *testing.T) {
		t.Parallel()
		pool := IntPool{
			Max:     100,
			Current: 75,
		}
		assert.Equal(t, 100, pool.Max, "最大値が正しく設定されない")
		assert.Equal(t, 75, pool.Current, "現在値が正しく設定されない")
	})

	t.Run("empty pool", func(t *testing.T) {
		t.Parallel()
		pool := IntPool{
			Max:     50,
			Current: 0,
		}
		assert.Equal(t, 50, pool.Max, "最大値が正しく設定されない")
		assert.Equal(t, 0, pool.Current, "空のプールの現在値が正しくない")
	})

	t.Run("full pool", func(t *testing.T) {
		t.Parallel()
		pool := IntPool{
			Max:     200,
			Current: 200,
		}
		assert.Equal(t, pool.Max, pool.Current, "満タンのプールで最大値と現在値が一致しない")
	})
}

func TestAbility(t *testing.T) {
	t.Parallel()
	t.Run("create ability", func(t *testing.T) {
		t.Parallel()
		abil := Ability{
			Base:     10,
			Modifier: 5,
			Total:    15,
		}
		assert.Equal(t, 10, abil.Base, "基本値が正しく設定されない")
		assert.Equal(t, 5, abil.Modifier, "修正値が正しく設定されない")
		assert.Equal(t, 15, abil.Total, "合計値が正しく設定されない")
	})

	t.Run("negative modifier", func(t *testing.T) {
		t.Parallel()
		abil := Ability{
			Base:     20,
			Modifier: -5,
			Total:    15,
		}
		assert.Equal(t, 20, abil.Base, "基本値が正しく設定されない")
		assert.Equal(t, -5, abil.Modifier, "負の修正値が正しく設定されない")
		assert.Equal(t, 15, abil.Total, "負の修正値を含む合計値が正しくない")
	})

	t.Run("zero values", func(t *testing.T) {
		t.Parallel()
		abil := Ability{
			Base:     0,
			Modifier: 0,
			Total:    0,
		}
		assert.Equal(t, 0, abil.Base, "ゼロの基本値が正しく設定されない")
		assert.Equal(t, 0, abil.Modifier, "ゼロの修正値が正しく設定されない")
		assert.Equal(t, 0, abil.Total, "ゼロの合計値が正しく設定されない")
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
		assert.Empty(t, input.Name, "空の素材名が正しく設定されない")
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

	t.Run("String returns correct names", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			slot     EquipmentSlotNumber
			expected string
		}{
			{SlotHead, "頭部"},
			{SlotTorso, "胴体"},
			{SlotArms, "腕部"},
			{SlotHands, "手部"},
			{SlotLegs, "脚部"},
			{SlotFeet, "足部"},
			{SlotJewelry, "装飾"},
			{SlotWeapon1, "武器1"},
			{SlotWeapon2, "武器2"},
			{SlotWeapon3, "武器3"},
			{SlotWeapon4, "武器4"},
			{SlotWeapon5, "武器5"},
		}

		for _, tt := range tests {
			t.Run(tt.expected, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.expected, tt.slot.String())
			})
		}
	})

	t.Run("未定義のスロット番号はパニックする", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { _ = EquipmentSlotNumber(99).String() })
	})
}

func TestParseEquipmentSlot(t *testing.T) {
	t.Parallel()

	t.Run("防具スロットのパース", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			input    string
			expected EquipmentSlotNumber
		}{
			{"HEAD", SlotHead},
			{"TORSO", SlotTorso},
			{"ARMS", SlotArms},
			{"HANDS", SlotHands},
			{"LEGS", SlotLegs},
			{"FEET", SlotFeet},
			{"JEWELRY", SlotJewelry},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				t.Parallel()
				slot, ok := ParseEquipmentSlot(tt.input)
				assert.True(t, ok)
				assert.Equal(t, tt.expected, slot)
			})
		}
	})

	t.Run("武器スロットのパース", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			input    string
			expected EquipmentSlotNumber
		}{
			{"WEAPON1", SlotWeapon1},
			{"WEAPON2", SlotWeapon2},
			{"WEAPON3", SlotWeapon3},
			{"WEAPON4", SlotWeapon4},
			{"WEAPON5", SlotWeapon5},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				t.Parallel()
				slot, ok := ParseEquipmentSlot(tt.input)
				assert.True(t, ok)
				assert.Equal(t, tt.expected, slot)
			})
		}
	})

	t.Run("無効な文字列はfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, ok := ParseEquipmentSlot("INVALID")
		assert.False(t, ok)
	})
}

// Amounter インターフェース（RatioAmount/NumeralAmount）は撤去され、
// ProvidesHealing のタグ付きデータに平坦化された。Calc の挙動は
// types_amount_test.go の TestProvidesHealing_Calc_* で検証する。
