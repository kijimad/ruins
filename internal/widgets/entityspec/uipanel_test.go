package entityspec_test

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
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
	labels := ui.CollectLabels(panel)

	assert.Contains(t, labels, "Attack", "見出しラベルが出る")
	assert.Contains(t, labels, "Damage", "データ行のラベルが出る")
	assert.Contains(t, labels, "25", "データ行の値が出る")
	assert.Contains(t, labels, "Element", "色付き行のラベルが出る")
	assert.Contains(t, labels, "Fire", "色付き行の値が出る")
}

func TestBuildSpecPanel_空でも落ちない(t *testing.T) {
	t.Parallel()
	panel := entityspec.BuildSpecPanel(nil, nil)
	assert.Empty(t, ui.CollectLabels(panel), "行が無ければラベルも無い")
}
