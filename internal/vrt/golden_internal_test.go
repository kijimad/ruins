package vrt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToleranceForSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		width   int
		height  int
		wantMin float64
		wantMax float64
	}{
		{
			name:    "小さいウィジェット画像は高いトレランスを返す",
			width:   300,
			height:  30,
			wantMin: 0.052,
			wantMax: 0.054,
		},
		{
			name:    "中程度の画像は中程度のトレランスを返す",
			width:   400,
			height:  120,
			wantMin: 0.022,
			wantMax: 0.024,
		},
		{
			name:    "大きい画像は低いトレランスを返す",
			width:   640,
			height:  480,
			wantMin: 0.009,
			wantMax: 0.010,
		},
		{
			name:    "フルスクリーン画像はさらに低いトレランスを返す",
			width:   960,
			height:  720,
			wantMin: 0.005,
			wantMax: 0.007,
		},
		{
			name:    "ゼロサイズは0を返す",
			width:   0,
			height:  0,
			wantMin: 0,
			wantMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toleranceForSize(tt.width, tt.height)
			assert.GreaterOrEqual(t, got, tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax)
		})
	}
}
