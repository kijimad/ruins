package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cell はつまみ名とメトリクス名から変化率を引くヘルパー。
func cell(t *testing.T, rows []KnobSensitivity, knob, metric string) float64 {
	t.Helper()
	for _, k := range rows {
		if k.Knob != knob {
			continue
		}
		for _, c := range k.Cells {
			if c.Metric == metric {
				return c.PctChange
			}
		}
	}
	require.FailNow(t, "cell not found", "%s x %s", knob, metric)
	return 0
}

func TestCrossDomainSensitivity_ブロック対角と符号(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	rows := CrossDomainSensitivity(master)
	require.Len(t, rows, len(knobRegistry()), "レジストリの全つまみが行になる")
	for _, k := range rows {
		assert.Len(t, k.Cells, len(SensitivityMetricNames), "各つまみは全メトリクス列を持つ")
	}

	// 全つまみが少なくとも1つのメトリクスを動かす。動かないつまみはレジストリの死角で、
	// Params に成分だけあって metricsAt が読んでいない配線漏れを検知する。
	for _, k := range rows {
		moved := false
		for _, c := range k.Cells {
			if c.PctChange != 0 {
				moved = true
			}
		}
		assert.True(t, moved, k.Knob+" はどのメトリクスも動かしていない")
	}

	// 各ドメインのつまみは自分のメトリクスだけを動かす(ブロック対角)。
	assert.Negative(t, cell(t, rows, "敵武器ダメージ(bite)", "戦力比d20"), "敵武器強化は戦力比を下げる")
	assert.Positive(t, cell(t, rows, "プレイヤー筋力", "戦力比d20"), "筋力は戦力比を上げる")
	assert.InDelta(t, 10, cell(t, rows, "最大満腹度", "飢餓まで日数"), 0.5, "満腹度は飢餓日数に線形")
	assert.Negative(t, cell(t, rows, "飢餓しきい値", "飢餓まで日数"), "しきい値が上がると飢餓に早く入る")
	assert.Positive(t, cell(t, rows, "疲労蓄積量", "睡眠時間割合"), "溜まりが速いほど睡眠時間割合は上がる")
	assert.Negative(t, cell(t, rows, "疲労回復量", "睡眠時間割合"), "回復が速いほど睡眠時間割合は下がる")
	assert.Positive(t, cell(t, rows, "燃料熱量", "OIL航続"), "燃料熱量は航続を伸ばす")
	// 店売りは価値の一定割合なので手取りは loot 価値に線形、弾力性はほぼ1(+10%で+10%前後)。
	// 整数手取りの丸めで操作点ごとに小さく振れるため、1超1未満に厳密固定せず近傍で見る。
	assert.InDelta(t, 10.0, cell(t, rows, "loot価値", "loot手取り"), 2.0, "手取りは loot 価値に線形、弾力性≒1")
	// 横断つまみは日換算メトリクスだけに効く。
	assert.Negative(t, cell(t, rows, "1日ターン数", "飢餓まで日数"), "1日ターンが増えると日数は減る")
	assert.Equal(t, 0.0, cell(t, rows, "1日ターン数", "戦力比d20"), "1日ターンは戦力比に効かない")

	// 戦闘つまみは生存メトリクスに波及しない(ドメイン疎結合の確認)。
	assert.Equal(t, 0.0, cell(t, rows, "敵武器ダメージ(bite)", "飢餓まで日数"), "戦闘つまみは飢餓に効かない")
	// 成長メトリクスはどのつまみでも動かない(成長のつまみは別途必要)。
	for _, k := range rows {
		assert.Equal(t, 0.0, cell(t, rows, k.Knob, "Lv30攻撃数"), k.Knob+" は成長に効かない")
	}
}
