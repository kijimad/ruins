package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoundaryMargins_崖まで動かすと帯端に届く(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	margins := BoundaryMargins(master)
	require.NotEmpty(t, margins)

	bands := metricBands()
	nameToIdx := map[string]int{}
	for i, n := range SensitivityMetricNames {
		nameToIdx[n] = i
	}
	knobByName := map[string]Knob{}
	for _, k := range knobRegistry() {
		knobByName[k.Name] = k
	}

	// 帯を持たないメトリクスは境界監視の対象外。
	for _, m := range margins {
		assert.True(t, bands[nameToIdx[m.Metric]].has, m.Metric+" は帯を持つ")
	}

	// 連続メトリクス、すなわち生存・疲労・物流は導出が滑らかなので、余白ぶん動かすと帯端へ厳密に届く。
	// 戦闘と成長は整数丸めで階段状になり端ちょうどに載らないので、この厳密照合からは除く。
	continuous := map[int]bool{
		nameToIdx["飢餓まで日数"]: true,
		nameToIdx["睡眠時間割合"]: true,
		nameToIdx["OIL航続"]:  true,
	}
	checked := 0
	for _, m := range margins {
		idx := nameToIdx[m.Metric]
		if !m.OK || !continuous[idx] {
			continue
		}
		p := DefaultParams()
		*knobByName[m.Knob].Ptr(&p) *= 1 + m.NearestPct/100
		got := metricValue(master, p, idx)
		edge := bands[idx].hi
		if m.Edge == "下限" {
			edge = bands[idx].lo
		}
		assert.InDelta(t, edge, got, edge*0.02+1e-6, m.Metric+"/"+m.Knob+" は余白ぶん動かすと帯端に届く")
		checked++
	}
	assert.Positive(t, checked, "連続メトリクスの余白が解けている")
}

func TestBoundaryMargins_余白の外は帯を割る(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	margins := BoundaryMargins(master)
	bands := metricBands()
	nameToIdx := map[string]int{}
	for i, n := range SensitivityMetricNames {
		nameToIdx[n] = i
	}
	knobByName := map[string]Knob{}
	for _, k := range knobRegistry() {
		knobByName[k.Name] = k
	}

	// 量子化を含めた全メトリクスで、余白の2倍まで動かせば帯を外れる。崖の位置が正しいことの確認。
	checked := 0
	for _, m := range margins {
		if !m.OK || m.NearestPct == 0 {
			continue
		}
		idx := nameToIdx[m.Metric]
		p := DefaultParams()
		*knobByName[m.Knob].Ptr(&p) *= 1 + (m.NearestPct/100)*2
		got := metricValue(master, p, idx)
		outside := got < bands[idx].lo-1e-9 || got > bands[idx].hi+1e-9
		assert.True(t, outside, m.Metric+"/"+m.Knob+" は余白の2倍で帯を外れる")
		checked++
	}
	assert.Positive(t, checked)
}
