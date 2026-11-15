package rawcheck_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/rawschema/rawcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_実データが通る(t *testing.T) {
	t.Parallel()
	raws, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	require.NoError(t, rawcheck.Validate(raws))
}

func TestValidate_違反を位置で返す(t *testing.T) {
	t.Parallel()
	raws := rawschema.Raws{Items: &[]rawschema.Item{{Id: "x", Name: "x", Description: "d", SpriteKey: "Bad Key", SpriteSheetName: "s"}}}
	err := rawcheck.Validate(raws)
	var ce *rawcheck.Error
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, []string{"items.0.spriteKey"}, ce.Paths())
}

func TestValidate_ダイス表記の違反を位置で返す(t *testing.T) {
	t.Parallel()
	// 個数0の 0d6 は Dice の正規表現が弾く
	raws := makeRefRaws(func(r *rawschema.Raws) { (*r.EnemyTables)[0].Entries[0].Pack = "0d6" })
	var ce *rawcheck.Error
	require.ErrorAs(t, rawcheck.Validate(raws), &ce)
	assert.Equal(t, []string{"enemyTables.0.entries.0.pack"}, ce.Paths())
}

func TestValidate_小屋の形は小屋の目印だけが持ち大きさに上下限がある(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*rawschema.Raws)
		want   []string
	}{
		{"小屋の目印に形が無い", func(r *rawschema.Raws) { (*r.Landmarks)[1].Hut = nil }, []string{"landmarks.1.hut"}},
		{"露天の目印に形がある", func(r *rawschema.Raws) {
			(*r.Landmarks)[0].Hut = &rawschema.LandmarkHut{W: 4, H: 4, Floor: "floor", Wall: "wall"}
		}, []string{"landmarks.0.hut"}},
		{"小屋の幅が出入口を置けないほど狭い", func(r *rawschema.Raws) { (*r.Landmarks)[1].Hut.W = 2 }, []string{"landmarks.1.hut.w"}},
		{"小屋の高さがチャンクの内側に収まらない", func(r *rawschema.Raws) { (*r.Landmarks)[1].Hut.H = 8 }, []string{"landmarks.1.hut.h"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var ce *rawcheck.Error
			require.ErrorAs(t, rawcheck.Validate(makeRefRaws(tt.mutate)), &ce)
			assert.Equal(t, tt.want, ce.Paths())
		})
	}
}

func TestError_Paths_位置を並べて重複を除く(t *testing.T) {
	t.Parallel()

	e := &rawcheck.Error{Violations: []rawcheck.Violation{
		{Path: "items.1.id", Message: "x"},
		{Path: "items.0.id", Message: "y"},
		{Path: "items.1.id", Message: "z"},
	}}
	assert.Equal(t, []string{"items.0.id", "items.1.id"}, e.Paths())
}
