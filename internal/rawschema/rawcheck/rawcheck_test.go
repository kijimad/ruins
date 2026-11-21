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

func TestValidate_コードだけが組む描画の段を書くと位置で返す(t *testing.T) {
	t.Parallel()
	for _, depth := range []rawschema.SpriteDepth{rawschema.SpriteDepthSnow, rawschema.SpriteDepthRemains, rawschema.SpriteDepthItem, rawschema.SpriteDepthPlayer} {
		raws := makeRefRaws(func(r *rawschema.Raws) { (*r.Props)[0].SpriteRender.Depth = depth })
		var ce *rawcheck.Error
		require.ErrorAs(t, rawcheck.Validate(raws), &ce, depth)
		assert.Equal(t, []string{"props.0.spriteRender.depth"}, ce.Paths(), depth)
	}
}

func TestValidate_効果を持つ物が消費物でないと位置で返す(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*rawschema.Item)
	}{
		{name: "治療を持つ物", mutate: func(it *rawschema.Item) { it.Consumable = nil }},
		{name: "栄養だけを持つ物", mutate: func(it *rawschema.Item) {
			it.Consumable, it.Remedy, it.ProvidesNutrition = nil, nil, new(10)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raws := makeRefRaws(func(r *rawschema.Raws) { tt.mutate(&(*r.Items)[0]) })
			var ce *rawcheck.Error
			require.ErrorAs(t, rawcheck.Validate(raws), &ce)
			assert.Equal(t, []string{"items.0.consumable"}, ce.Paths())
		})
	}
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

func TestValidate_携行光源を持つアイテムは残量を持つ(t *testing.T) {
	t.Parallel()
	raws := makeRefRaws(func(r *rawschema.Raws) {
		(*r.Items)[0].LightSource = &rawschema.LightSource{Radius: 4, Color: rawschema.RGBAColor{A: 255}}
	})
	var ce *rawcheck.Error
	require.ErrorAs(t, rawcheck.Validate(raws), &ce)
	assert.Equal(t, []string{"items.0.charge"}, ce.Paths())

	(*raws.Items)[0].Charge = &rawschema.Charge{Max: 600}
	require.NoError(t, rawcheck.Validate(raws), "残量を持てば通る")
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
