package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestNeedDefs_FreshnessAt_経過した段階数で段階を選ぶ(t *testing.T) {
	t.Parallel()
	const shelf consts.Turn = 100
	p := Perishable{StageLength: shelf}
	defs := NeedDefs{Freshness: []FreshnessDef{
		{Level: rawschema.Fresh, From: 0},
		{Level: rawschema.Stale, From: 1},
		{Level: rawschema.Rotten, From: 2},
	}}

	tests := []struct {
		name string
		rot  consts.Turn
		want rawschema.FreshnessStageId
	}{
		{"劣化ゼロは新鮮", 0, rawschema.Fresh},
		{"保存期間直前は新鮮", shelf - 1, rawschema.Fresh},
		{"保存期間ちょうどで劣化", shelf, rawschema.Stale},
		{"2倍直前は劣化", 2*shelf - 1, rawschema.Stale},
		{"2倍で腐敗", 2 * shelf, rawschema.Rotten},
		{"十分な劣化は腐敗のまま", 10 * shelf, rawschema.Rotten},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, defs.FreshnessAt(p.StagesElapsed(tt.rot)).Level)
		})
	}
}
