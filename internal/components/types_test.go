package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/oapi"
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
			ID:     "鉄",
			Amount: 3,
		}
		assert.Equal(t, "鉄", input.ID, "素材名が正しく設定されない")
		assert.Equal(t, 3, input.Amount, "必要量が正しく設定されない")
	})

	t.Run("empty name", func(t *testing.T) {
		t.Parallel()
		input := RecipeInput{
			ID:     "",
			Amount: 1,
		}
		assert.Empty(t, input.ID, "空の素材名が正しく設定されない")
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
			TargetGroup: oapi.TargetGroupENEMY,
			TargetNum:   oapi.SINGLE,
		}
		assert.Equal(t, oapi.TargetGroupENEMY, target.TargetGroup, "対象グループが正しく設定されない")
		assert.Equal(t, oapi.SINGLE, target.TargetNum, "対象数が正しく設定されない")
	})

	t.Run("various target combinations", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name        string
			targetGroup oapi.TargetGroup
			targetNum   oapi.TargetNum
		}{
			{"single enemy", oapi.TargetGroupENEMY, oapi.SINGLE},
			{"all enemies", oapi.TargetGroupENEMY, oapi.ALL},
			{"single ally", oapi.TargetGroupALLY, oapi.SINGLE},
			{"all allies", oapi.TargetGroupALLY, oapi.ALL},
			{"single weapon", oapi.TargetGroupWEAPON, oapi.SINGLE},
			{"none target", oapi.TargetGroupNONE, oapi.SINGLE},
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
			expected i18n.Msgid
		}{
			{SlotHead, "Head"},
			{SlotTorso, "Torso"},
			{SlotArms, "Arms"},
			{SlotHands, "Hands"},
			{SlotLegs, "Legs"},
			{SlotFeet, "Feet"},
			{SlotJewelry, "Accessory"},
			{SlotWeapon1, "Weapon 1"},
			{SlotWeapon2, "Weapon 2"},
			{SlotWeapon3, "Weapon 3"},
			{SlotWeapon4, "Weapon 4"},
			{SlotWeapon5, "Weapon 5"},
		}

		for _, tt := range tests {
			t.Run(string(tt.expected), func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.expected, tt.slot.Name())
			})
		}
	})

	t.Run("未定義のスロット番号はパニックする", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { _ = EquipmentSlotNumber(99).Name() })
	})
}

func TestParseEquipmentSlot(t *testing.T) {
	t.Parallel()

	slots := []oapi.EquipSlot{
		oapi.EquipSlotHEAD, oapi.EquipSlotTORSO, oapi.EquipSlotARMS, oapi.EquipSlotHANDS, oapi.EquipSlotLEGS, oapi.EquipSlotFEET,
		oapi.EquipSlotWEAPON1, oapi.EquipSlotWEAPON2, oapi.EquipSlotWEAPON3, oapi.EquipSlotWEAPON4, oapi.EquipSlotWEAPON5,
	}
	got := map[oapi.EquipSlot]EquipmentSlotNumber{}
	for _, slot := range slots {
		n, ok := ParseEquipmentSlot(slot)
		assert.True(t, ok, slot)
		got[slot] = n
	}
	assert.Equal(t, map[oapi.EquipSlot]EquipmentSlotNumber{
		oapi.EquipSlotHEAD: SlotHead, oapi.EquipSlotTORSO: SlotTorso, oapi.EquipSlotARMS: SlotArms,
		oapi.EquipSlotHANDS: SlotHands, oapi.EquipSlotLEGS: SlotLegs, oapi.EquipSlotFEET: SlotFeet,
		oapi.EquipSlotWEAPON1: SlotWeapon1, oapi.EquipSlotWEAPON2: SlotWeapon2, oapi.EquipSlotWEAPON3: SlotWeapon3,
		oapi.EquipSlotWEAPON4: SlotWeapon4, oapi.EquipSlotWEAPON5: SlotWeapon5,
	}, got)

	_, ok := ParseEquipmentSlot("INVALID")
	assert.False(t, ok, "未知の値は ok=false")
}

// Amounter インターフェース（RatioAmount/NumeralAmount）は撤去され、
// ProvidesHealing のタグ付きデータに平坦化された。Calc の挙動は
// types_amount_test.go の TestProvidesHealing_Calc_* で検証する。
