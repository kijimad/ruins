package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
