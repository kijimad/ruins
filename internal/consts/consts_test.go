package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstants(t *testing.T) {
	t.Parallel()
	// 定数の値をテスト
	assert.Equal(t, 960, MinGameWidth, "MinGameWidthの値が正しくない")
	assert.Equal(t, 720, MinGameHeight, "MinGameHeightの値が正しくない")
	assert.Equal(t, 32, int(TileSize), "TileSizeの値が正しくない")

	// ラベルの値をテスト
	assert.Equal(t, "HP", HPLabel, "HPLabelの値が正しくない")
	assert.Equal(t, "SP", SPLabel, "SPLabelの値が正しくない")
	assert.Equal(t, "体力", VitalityLabel, "VitalityLabelの値が正しくない")
	assert.Equal(t, "筋力", StrengthLabel, "StrengthLabelの値が正しくない")
	assert.Equal(t, "感覚", SensationLabel, "SensationLabelの値が正しくない")
	assert.Equal(t, "器用", DexterityLabel, "DexterityLabelの値が正しくない")
	assert.Equal(t, "敏捷", AgilityLabel, "AgilityLabelの値が正しくない")
	assert.Equal(t, "防御", DefenseLabel, "DefenseLabelの値が正しくない")
	assert.Equal(t, "命中", AccuracyLabel, "AccuracyLabelの値が正しくない")
	assert.Equal(t, "攻撃力", DamageLabel, "DamageLabelの値が正しくない")
	assert.Equal(t, "回数", AttackCountLabel, "AttackCountLabelの値が正しくない")
	assert.Equal(t, "部位", EquimentCategoryLabel, "EquimentCategoryLabelの値が正しくない")
}
