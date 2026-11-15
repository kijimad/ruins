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

func TestValidate_下地を使う間取りを持つダンジョンは下地を持つ(t *testing.T) {
	t.Parallel()
	raws := makeRefRaws(func(r *rawschema.Raws) {
		(*r.Layouts)[0].Tiles.Floor = "ground"
	})
	var ce *rawcheck.Error
	require.ErrorAs(t, rawcheck.Validate(raws), &ce)
	assert.Equal(t, []string{"dungeons.0.ground"}, ce.Paths())

	(*raws.Dungeons)[0].Ground = new(rawschema.EntityID("floor"))
	require.NoError(t, rawcheck.Validate(raws), "下地を持てば通る")
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

func TestValidate_タイルのidに下地を残す地形の予約語を使わない(t *testing.T) {
	t.Parallel()
	raws := makeRefRaws(func(r *rawschema.Raws) {
		g := (*r.Tiles)[1]
		g.Id = "ground"
		*r.Tiles = append(*r.Tiles, g)
	})
	var ce *rawcheck.Error
	require.ErrorAs(t, rawcheck.Validate(raws), &ce)
	assert.Equal(t, []string{"tiles.2.id"}, ce.Paths())
}

func TestValidate_壁のタイルは通れない(t *testing.T) {
	t.Parallel()
	raws := makeRefRaws(func(r *rawschema.Raws) {
		(*r.Tiles)[0].Wall = new(true)
		(*r.Tiles)[0].BlockPass = false
	})
	var ce *rawcheck.Error
	require.ErrorAs(t, rawcheck.Validate(raws), &ce)
	assert.Equal(t, []string{"tiles.0.blockPass"}, ce.Paths())

	(*raws.Tiles)[0].BlockPass = true
	require.NoError(t, rawcheck.Validate(raws), "通れない壁は通る")
}
