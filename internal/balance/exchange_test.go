package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exchangeCell は交換レート表から、基準つまみ a を+10%したときの相手 b の補償率を引くヘルパー。
func exchangeCell(t *testing.T, rows []MetricExchange, metric, a, b string) ExchangeCell {
	t.Helper()
	for _, mx := range rows {
		if mx.Metric != metric {
			continue
		}
		for _, row := range mx.Rows {
			if row.Knob != a {
				continue
			}
			for _, c := range row.Cells {
				if c.Compensator == b {
					return c
				}
			}
		}
	}
	require.FailNow(t, "cell not found", "%s: %s -> %s", metric, a, b)
	return ExchangeCell{}
}

func TestExchangeRates_単独つまみのメトリクスは表を作らない(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	rows := ExchangeRates(master)
	// OIL航続と loot手取りと Lv30攻撃数は動かすつまみが1つ以下なので交換の余地がなく表に出ない。
	for _, mx := range rows {
		assert.NotEqual(t, "OIL航続", mx.Metric, "単独つまみのメトリクスは表を作らない")
		assert.NotEqual(t, "loot手取り", mx.Metric)
		assert.NotEqual(t, "Lv30攻撃数", mx.Metric)
		assert.GreaterOrEqual(t, len(mx.Knobs), 2, "表に出るのは2つ以上動かすメトリクスだけ")
	}
}

func TestExchangeRates_線形メトリクスの補償が手計算と一致(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	rows := ExchangeRates(master)

	// 飢餓日数は満腹度・減耗ターンに線形なので、片方+10%は他方 1/1.1-1=-9.1% で打ち消せる。
	c := exchangeCell(t, rows, "飢餓まで日数", "最大満腹度", "空腹減耗ターン")
	require.True(t, c.OK, "線形メトリクスは範囲内で解ける")
	assert.InDelta(t, -9.1, c.PctChange, 0.3, "満腹度+10%%は減耗ターン-9.1%%で打ち消せる")

	// 睡眠時間割合は蓄積と回復が対称なので、蓄積+10%%は回復+10%%で打ち消せる。
	s := exchangeCell(t, rows, "睡眠時間割合", "疲労蓄積量", "疲労回復量")
	require.True(t, s.OK)
	assert.InDelta(t, 10.0, s.PctChange, 0.3, "蓄積+10%%は回復+10%%で打ち消せる")
}

func TestExchangeRates_戦闘の補償は正で両立可能(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	rows := ExchangeRates(master)
	// 敵を+10%強くしたら味方の筋力を増やして戻す。符号は正で、範囲内で解ける。
	c := exchangeCell(t, rows, "戦力比d20", "敵武器ダメージ(bite)", "プレイヤー筋力")
	require.True(t, c.OK, "戦闘の補償は範囲内で解ける")
	assert.Positive(t, c.PctChange, "敵強化は味方強化で打ち消す")
}
