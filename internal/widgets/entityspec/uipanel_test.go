package entityspec_test

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/internal/uicore"
)

// BuildSpecPanel の検証はツリー構造で行う。CollectLabels は描画せず Value を集めるだけなので
// フェイスも ebiten も要らず、完全に並列でよい。
func TestBuildSpecPanel_見出しとデータ行を組む(t *testing.T) {
	t.Parallel()
	rows := []entityspec.SpecRow{
		{Label: "Attack", Header: true},
		{Label: "Damage", Value: "25"},
		{Label: "Element", Value: "Fire", Color: &color.RGBA{R: 0xff, A: 0xff}},
	}
	panel := entityspec.BuildSpecPanel(rows, nil)
	labels := uicore.CollectLabels(panel)

	// 見出し・データ行のラベルと値・色付き行が、この並びどおりに出る
	assert.Equal(t, []string{"Attack", "Damage", "25", "Element", "Fire"}, labels)
}

func TestBuildSpecPanel_空でも落ちない(t *testing.T) {
	t.Parallel()
	panel := entityspec.BuildSpecPanel(nil, nil)
	assert.Empty(t, uicore.CollectLabels(panel), "行が無ければラベルも無い")
}
