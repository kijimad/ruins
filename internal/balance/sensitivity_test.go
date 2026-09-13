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
	require.Len(t, rows, 8, "8つのつまみ")
	for _, k := range rows {
		assert.Len(t, k.Cells, len(SensitivityMetricNames), "各つまみは全メトリクス列を持つ")
	}

	// 各ドメインのつまみは自分のメトリクスだけを動かす(ブロック対角)。
	assert.Negative(t, cell(t, rows, "敵武器ダメージ(bite)", "戦力比d20"), "敵武器強化は戦力比を下げる")
	assert.Positive(t, cell(t, rows, "プレイヤー筋力", "戦力比d20"), "筋力は戦力比を上げる")
	assert.InDelta(t, 10, cell(t, rows, "最大満腹度", "飢餓まで日数"), 0.5, "満腹度は飢餓日数に線形")
	assert.Negative(t, cell(t, rows, "疲労回復量", "睡眠時間割合"), "回復が速いほど睡眠時間割合は下がる")
	assert.Positive(t, cell(t, rows, "燃料熱量", "OIL航続"), "燃料熱量は航続を伸ばす")
	// 送料が固定なので loot 価値に対する手取りの弾力性は1を超える(+10%で+10%超)。
	assert.Greater(t, cell(t, rows, "loot価値", "loot手取り"), 10.0, "手取りは loot 価値に対し弾力性>1")
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
