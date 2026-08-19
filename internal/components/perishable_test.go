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

func TestFreshnessStage_Rank(t *testing.T) {
	t.Parallel()
	// 新鮮→劣化→腐敗の固定順。一覧の並び副キーが依存する
	assert.Equal(t, 1, FreshnessFresh.Rank())
	assert.Equal(t, 2, FreshnessStale.Rank())
	assert.Equal(t, 3, FreshnessRotten.Rank())
}

func TestFreshnessStage_Label(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Fresh", FreshnessFresh.Label())
	assert.Equal(t, "Stale", FreshnessStale.Label())
	assert.Equal(t, "Rotten", FreshnessRotten.Label())
}

func TestFreshnessStage_未知の段階はpanicする(t *testing.T) {
	t.Parallel()
	// 段階は算出値で保存されないため、未知の段階は enum 追加漏れのプログラムミスに限る。
	// 黙って握りつぶさず即座に気付けるよう fail-fast で panic する
	assert.Panics(t, func() { FreshnessStage("bogus").Rank() })
	assert.Panics(t, func() { FreshnessStage("bogus").Label() })
}
