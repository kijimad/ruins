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

func TestFreshnessRank(t *testing.T) {
	t.Parallel()
	// 新鮮→劣化→腐敗の固定順。一覧の並び副キーが依存する
	assert.Equal(t, []int{1, 2, 3}, []int{FreshnessRank(oapi.Fresh), FreshnessRank(oapi.Stale), FreshnessRank(oapi.Rotten)})
}

func TestFreshnessRank_未知の段階はpanicする(t *testing.T) {
	t.Parallel()
	// 段階は算出値で保存されないため、未知の段階は enum 追加漏れのプログラムミスに限る。
	// 黙って握りつぶさず即座に気付けるよう fail-fast で panic する
	assert.Panics(t, func() { FreshnessRank("bogus") })
}
