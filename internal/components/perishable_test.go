package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestPerishable_Stage(t *testing.T) {
	t.Parallel()
	const shelf consts.Turn = 100
	p := Perishable{StageLength: shelf}

	tests := []struct {
		name string
		rot  consts.Turn
		want oapi.FreshnessStageId
	}{
		{"劣化ゼロは新鮮", 0, oapi.Fresh},
		{"保存期間直前は新鮮", shelf - 1, oapi.Fresh},
		{"保存期間ちょうどで劣化", shelf, oapi.Stale},
		{"2倍直前は劣化", 2*shelf - 1, oapi.Stale},
		{"2倍で腐敗", 2 * shelf, oapi.Rotten},
		{"十分な劣化は腐敗のまま", 10 * shelf, oapi.Rotten},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Stage(tt.rot))
		})
	}
}
