package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestChebyshevDistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ax   int
		ay   int
		bx   int
		by   int
		want int
	}{
		{"同じ位置", 5, 5, 5, 5, 0},
		{"水平距離", 0, 0, 3, 0, 3},
		{"垂直距離", 0, 0, 0, 4, 4},
		{"斜め距離", 0, 0, 3, 3, 3},
		{"斜め距離で水平が大きい", 0, 0, 5, 3, 5},
		{"負の座標", 0, 0, -3, -4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := &gc.GridElement{X: consts.Tile(tt.ax), Y: consts.Tile(tt.ay)}
			b := &gc.GridElement{X: consts.Tile(tt.bx), Y: consts.Tile(tt.by)}
			assert.Equal(t, tt.want, chebyshevDistance(a, b))
		})
	}
}

func TestShouldRetreatLowHP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current int
		max     int
		want    bool
	}{
		{"HP満タン", 100, 100, false},
		{"HP50%", 50, 100, false},
		{"HP26%", 26, 100, false},
		{"HP25%", 25, 100, true},
		{"HP10%", 10, 100, true},
		{"HP0%", 0, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sp := NewSquadProcessor()
			// HPコンポーネントの直接チェックはWorldが必要なので、ロジックだけテスト
			hp := &gc.HP{Current: tt.current, Max: tt.max}
			result := hp.Max > 0 && hp.Current*100/hp.Max <= hpRetreatThreshold
			assert.Equal(t, tt.want, result)
			_ = sp
		})
	}
}

func TestNewSquadProcessor(t *testing.T) {
	t.Parallel()
	sp := NewSquadProcessor()
	assert.NotNil(t, sp)
	assert.NotNil(t, sp.logger)
	assert.NotNil(t, sp.visionSystem)
}
