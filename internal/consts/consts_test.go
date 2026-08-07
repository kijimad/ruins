package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	t.Parallel()
	// 定数の値をテスト
	assert.Equal(t, 960, GameWidth, "GameWidthの値が正しくない")
	assert.Equal(t, 720, GameHeight, "GameHeightの値が正しくない")
	assert.Equal(t, 32, int(TileSize), "TileSizeの値が正しくない")

	// ラベルの値をテスト
	assert.Equal(t, "HP", HPLabel, "HPLabelの値が正しくない")
	assert.Equal(t, "Vitality", VitalityLabel, "VitalityLabelの値が正しくない")
	assert.Equal(t, "Strength", StrengthLabel, "StrengthLabelの値が正しくない")
	assert.Equal(t, "Sensation", SensationLabel, "SensationLabelの値が正しくない")
	assert.Equal(t, "Dexterity", DexterityLabel, "DexterityLabelの値が正しくない")
	assert.Equal(t, "Agility", AgilityLabel, "AgilityLabelの値が正しくない")
	assert.Equal(t, "Defense", DefenseLabel, "DefenseLabelの値が正しくない")
	assert.Equal(t, "Accuracy", AccuracyLabel, "AccuracyLabelの値が正しくない")
	assert.Equal(t, "Attack power", DamageLabel, "DamageLabelの値が正しくない")
	assert.Equal(t, "Hits", AttackCountLabel, "AttackCountLabelの値が正しくない")
	assert.Equal(t, "Slot", EquimentCategoryLabel, "EquimentCategoryLabelの値が正しくない")
}
