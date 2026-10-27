package hud

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGameInfo_bodyTempGauge(t *testing.T) {
	t.Parallel()
	info := newTestGameInfo(t, ebiten.NewImage(1, 1))
	rect := image.Rect(0, 0, 100, gaugeHeight)

	t.Run("非表示なら枠も塗りも描かない", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		wgt := info.bodyTempGauge(GameInfoData{BodyTempVisible: false})
		wgt.Layout(rect)
		wgt.Draw(cv)
		assert.Zero(t, cv.roundedStrokes)
		assert.Empty(t, cv.tintedRects)
	})

	t.Run("表示なら比率ぶんの幅で塗り枠も描く", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		wgt := info.bodyTempGauge(GameInfoData{BodyTempVisible: true, BodyTempRatio: 0.5})
		wgt.Layout(rect)
		wgt.Draw(cv)
		require.Equal(t, 1, cv.roundedStrokes, "枠は常に描く")
		require.Len(t, cv.tintedRects, 1)
		assert.Equal(t, 50, cv.tintedRects[0].Dx(), "比率0.5なら幅は半分")
	})
}

func TestGameInfo_healthGauge(t *testing.T) {
	t.Parallel()
	info := newTestGameInfo(t, ebiten.NewImage(1, 1))
	rect := image.Rect(0, 0, 100, gaugeHeight)

	tests := []struct {
		name      string
		hp, maxHP int
		wantWidth int
	}{
		{"最大HPが0以下なら比率0で塗らない", 10, 0, 0},
		{"HPが最大の半分なら半分塗る", 50, 100, 50},
		{"HPが最大を超えても比率は1で頭打ち", 150, 100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cv := &fakeCanvas{}
			wgt := info.healthGauge(GameInfoData{PlayerHP: tt.hp, PlayerMaxHP: tt.maxHP})
			wgt.Layout(rect)
			wgt.Draw(cv)
			require.Equal(t, 1, cv.roundedStrokes, "枠は常に描く")
			if tt.wantWidth == 0 {
				assert.Empty(t, cv.tintedRects, "比率0なら塗りは描かない")
			} else {
				require.Len(t, cv.tintedRects, 1)
				assert.Equal(t, tt.wantWidth, cv.tintedRects[0].Dx())
			}
		})
	}
}

func TestGameInfo_fuelGauge(t *testing.T) {
	t.Parallel()
	info := newTestGameInfo(t, ebiten.NewImage(1, 1))
	rect := image.Rect(0, 0, 100, gaugeHeight)

	tests := []struct {
		name      string
		ratio     float64
		wantWidth int
	}{
		{"充填率ぶんの幅で塗る", 0.25, 25},
		{"満タンなら全幅で塗る", 1.0, 100},
		{"空なら塗りは描かない", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cv := &fakeCanvas{}
			wgt := info.fuelGauge(GameInfoData{FuelRatio: tt.ratio})
			wgt.Layout(rect)
			wgt.Draw(cv)
			require.Equal(t, 1, cv.roundedStrokes, "枠は常に描く")
			if tt.wantWidth == 0 {
				assert.Empty(t, cv.tintedRects, "比率0なら塗りは描かない")
			} else {
				require.Len(t, cv.tintedRects, 1)
				assert.Equal(t, tt.wantWidth, cv.tintedRects[0].Dx())
			}
		})
	}
}

func TestWeightColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		weight, maxWeight consts.Milligram
		want              color.RGBA
	}{
		{"最大重量が0以下なら通常色", 100, 0, theme.TextPrimary},
		{"8割以下なら通常色", 80, 100, theme.TextPrimary},
		{"8割超なら警告色", 81, 100, theme.HUDWeightWarning},
		{"超過なら危険色", 101, 100, theme.HUDWeightDanger},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := weightColor(GameInfoData{PlayerWeight: tt.weight, PlayerMaxWeight: tt.maxWeight})
			assert.Equal(t, tt.want, got)
		})
	}
}
