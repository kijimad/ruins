package overworld

import (
	"image/color"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGlyphInfo_凡例の全記号に色が設定されている は、記号を足して色を付け忘れるとゼロ値の透明黒に
// なる退行を定義元で捕らえる。色は GlyphInfo が記号と同居して持ち、UI 側はこれを引き写すだけ。
func TestGlyphInfo_凡例の全記号に色が設定されている(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	glyphs := LegendGlyphs(raws)
	require.NotEmpty(t, glyphs, "凡例記号が定義されている")
	for _, g := range glyphs {
		assert.NotEqualf(t, color.RGBA{}, g.Color, "%s(%c) に色が設定されている", g.Name, g.Label)
	}
}

// findUrbanChunk は市街地の建物チャンクになる seed と座標を探す。
func findUrbanChunk(t *testing.T, rows consts.Chunk) (uint64, consts.Coord[consts.Chunk]) {
	t.Helper()
	for s := uint64(1); s < 500; s++ {
		for y := range rows {
			for x := range consts.Chunk(12) {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				if _, ok := urbanChunkAt(s, c, rows); ok {
					return s, c
				}
			}
		}
	}
	require.Fail(t, "前提: 市街地チャンクを持つ seed が見つかる")
	return 0, consts.Coord[consts.Chunk]{}
}

func TestChunkPlace_市街地の建物チャンクは施設種別の文字を返す(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := ZoneCatalogFrom(raws)
	const rows consts.Chunk = 9
	seed, c := findUrbanChunk(t, rows)
	kind, ok := urbanFacilityAt(cat, seed, c, rows)
	require.True(t, ok, "前提: 市街地チャンク")

	g, ok := glyphByID(raws, kind)
	require.True(t, ok, "施設 %q の記号が mapGlyphs にある", kind)
	assert.Equal(t, g.Label, ChunkPlace(raws, cat, seed, c, rows), "建物チャンクは施設種別の文字を返す")
}

func TestChunkPlace_純関数で決定的(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := ZoneCatalogFrom(raws)
	const rows consts.Chunk = 9
	seed, c := findUrbanChunk(t, rows)
	first := ChunkPlace(raws, cat, seed, c, rows)
	for range 5 {
		assert.Equal(t, first, ChunkPlace(raws, cat, seed, c, rows), "同じ引数なら毎回同じ文字")
	}
}

func TestChunkPlace_遺跡入口と集落が地物の文字で出る(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := ZoneCatalogFrom(raws)
	const rows consts.Chunk = 9

	dungeonGlyph := placeGlyph(raws, placeDungeonEntrance)
	villageGlyph := placeGlyph(raws, placeVillage)
	hamletGlyph := placeGlyph(raws, placeHamlet)
	foundDungeonEntrance, foundVillage, foundHamlet := false, false, false
	for s := uint64(1); s < 400 && (!foundDungeonEntrance || !foundVillage || !foundHamlet); s++ {
		for y := range rows {
			for x := range consts.Chunk(8) {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				// 市街地に上書きされないチャンクだけ見る
				if _, ok := urbanChunkAt(s, c, rows); ok {
					continue
				}
				switch ChunkPlace(raws, cat, s, c, rows) {
				case dungeonGlyph:
					foundDungeonEntrance = true
				case villageGlyph:
					foundVillage = true
				case hamletGlyph:
					foundHamlet = true
				}
			}
		}
	}
	assert.True(t, foundDungeonEntrance, "遺跡入口チャンクが > で出る")
	assert.True(t, foundVillage, "村の集落が村の文字で出る")
	assert.True(t, foundHamlet, "一軒家の集落が一軒家の文字で出る。開始特例で常に村になる退行の検知")
}

func TestSchematicLegend_全ての記号を含む(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	legend := SchematicLegend(raws)
	for _, g := range LegendGlyphs(raws) {
		assert.Truef(t, strings.ContainsRune(legend, g.Label), "凡例に %s(%c) がある", g.Name, g.Label)
	}
}
