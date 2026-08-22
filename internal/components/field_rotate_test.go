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

	cases := []struct {
		name   string
		dir    Direction
		wantSu float64
		wantSr float64
	}{
		{"上", DirectionUp, 1, 0},
		{"下", DirectionDown, -1, 0},
		{"右", DirectionRight, 0, 1},
		{"左", DirectionLeft, 0, -1},
		{"右上", DirectionUpRight, 1, 1},
		{"左上", DirectionUpLeft, 1, -1},
		{"右下", DirectionDownRight, -1, 1},
		{"左下", DirectionDownLeft, -1, -1},
		{"方向なし", DirectionNone, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			su, sr := tc.dir.ScreenIntent()
			assert.Equal(t, tc.wantSu, su)
			assert.Equal(t, tc.wantSr, sr)
		})
	}
}
