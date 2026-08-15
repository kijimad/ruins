package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestPerishable_Stage(t *testing.T) {
	t.Parallel()
	const shelf consts.Turn = 100
	p := Perishable{StageLength: shelf}

	tests := []struct {
		name string
		rot  consts.Turn
		want FreshnessStage
	}{
		{"劣化ゼロは新鮮", 0, FreshnessFresh},
		{"保存期間直前は新鮮", shelf - 1, FreshnessFresh},
		{"保存期間ちょうどで劣化", shelf, FreshnessStale},
		{"2倍直前は劣化", 2*shelf - 1, FreshnessStale},
		{"2倍で腐敗", 2 * shelf, FreshnessRotten},
		{"十分な劣化は腐敗のまま", 10 * shelf, FreshnessRotten},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Stage(tt.rot))
		})
	}
}

func TestPerishable_MergeRot(t *testing.T) {
	t.Parallel()

	t.Run("個数で加重平均する", func(t *testing.T) {
		t.Parallel()
		target := Perishable{RotAccrued: 100, StageLength: 1000}
		target.MergeRot(3, Perishable{RotAccrued: 500, StageLength: 1000}, 1)
		// (100*3 + 500*1) / 4 = 200
		assert.Equal(t, consts.Turn(200), target.RotAccrued)
	})

	t.Run("総数ゼロなら変えない", func(t *testing.T) {
		t.Parallel()
		target := Perishable{RotAccrued: 42, StageLength: 1000}
		target.MergeRot(0, Perishable{RotAccrued: 500, StageLength: 1000}, 0)
		assert.Equal(t, consts.Turn(42), target.RotAccrued)
	})
}
