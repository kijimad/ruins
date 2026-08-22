package components

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRotateScreenDir(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		base Direction
		yaw  float64
		want Direction
	}{
		{"回転なしは上キーで北", DirectionUp, 0, DirectionUp},
		{"回転なしは右キーで東", DirectionRight, 0, DirectionRight},
		{"180度で上キーは南へ反転", DirectionUp, math.Pi, DirectionDown},
		{"180度で右キーは西へ反転", DirectionRight, math.Pi, DirectionLeft},
		{"90度で上キーは西へ", DirectionUp, math.Pi / 2, DirectionLeft},
		{"270度で上キーは東へ", DirectionUp, 3 * math.Pi / 2, DirectionRight},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, RotateScreenDir(tc.base, tc.yaw))
		})
	}
}

func TestSnapWorldVec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		wx, wy float64
		want   Direction
	}{
		{"北", 0, -1, DirectionUp},
		{"南", 0, 1, DirectionDown},
		{"東", 1, 0, DirectionRight},
		{"西", -1, 0, DirectionLeft},
		{"北東", 1, -1, DirectionUpRight},
		{"南西", -1, 1, DirectionDownLeft},
		{"北寄りは北へ丸める", 0.2, -1, DirectionUp},
		{"ゼロは方向なし", 0, 0, DirectionNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, SnapWorldVec(tc.wx, tc.wy))
		})
	}
}

func TestDirection_ScreenIntent(t *testing.T) {
	t.Parallel()

	su, sr := DirectionUp.ScreenIntent()
	assert.Equal(t, 1.0, su)
	assert.Equal(t, 0.0, sr)

	su, sr = DirectionDownLeft.ScreenIntent()
	assert.Equal(t, -1.0, su)
	assert.Equal(t, -1.0, sr)

	su, sr = DirectionNone.ScreenIntent()
	assert.Equal(t, 0.0, su)
	assert.Equal(t, 0.0, sr)
}
