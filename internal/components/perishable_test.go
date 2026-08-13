package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestPerishable_Stage(t *testing.T) {
	t.Parallel()
	const shelf consts.Turn = 100
	p := Perishable{SpawnedAtTurn: 0, ShelfLife: shelf}

	tests := []struct {
		name string
		now  consts.Turn
		want FreshnessStage
	}{
		{"生成直後は新鮮", 0, FreshnessFresh},
		{"保存期間直前は新鮮", shelf - 1, FreshnessFresh},
		{"保存期間ちょうどで劣化", shelf, FreshnessStale},
		{"2倍直前は劣化", 2*shelf - 1, FreshnessStale},
		{"2倍で腐敗", 2 * shelf, FreshnessRotten},
		{"十分経過は腐敗のまま", 10 * shelf, FreshnessRotten},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Stage(tt.now))
		})
	}
}

func TestPerishable_Stage_生成時刻からの経過で判定する(t *testing.T) {
	t.Parallel()
	const shelf consts.Turn = 100
	p := Perishable{SpawnedAtTurn: 500, ShelfLife: shelf}

	assert.Equal(t, FreshnessFresh, p.Stage(599), "生成+99は新鮮")
	assert.Equal(t, FreshnessStale, p.Stage(600), "生成+100で劣化")
}
