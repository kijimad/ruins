package overworld

import (
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findCityChunk は市街地の建物チャンクになる seed と座標を探す。
func findCityChunk(t *testing.T, rows consts.Chunk) (uint64, worldstream.ChunkCoord) {
	t.Helper()
	for s := uint64(1); s < 500; s++ {
		for y := range rows {
			for x := range consts.Chunk(12) {
				c := worldstream.ChunkCoord{X: x, Y: y}
				if _, _, ok := cityChunkInfo(s, c, rows); ok {
					return s, c
				}
			}
		}
	}
	require.Fail(t, "前提: 市街地チャンクを持つ seed が見つかる")
	return 0, worldstream.ChunkCoord{}
}

func TestChunkPlace_市街地の建物チャンクは施設種別の文字を返す(t *testing.T) {
	t.Parallel()

	const rows consts.Chunk = 9
	seed, c := findCityChunk(t, rows)
	facility, _, ok := cityChunkInfo(seed, c, rows)
	require.True(t, ok, "前提: 市街地チャンク")

	want := facilityGlyphs[facilityCatalog[facility].kind].Label
	assert.Equal(t, want, ChunkPlace(seed, c, rows), "建物チャンクは施設種別の文字を返す")
}

func TestChunkPlace_純関数で決定的(t *testing.T) {
	t.Parallel()

	const rows consts.Chunk = 9
	seed, c := findCityChunk(t, rows)
	first := ChunkPlace(seed, c, rows)
	for range 5 {
		assert.Equal(t, first, ChunkPlace(seed, c, rows), "同じ引数なら毎回同じ文字")
	}
}

func TestChunkPlace_遺跡入口と集落が地物の文字で出る(t *testing.T) {
	t.Parallel()

	const rows consts.Chunk = 9

	foundRuin, foundVillage, foundHamlet := false, false, false
	for s := uint64(1); s < 400 && (!foundRuin || !foundVillage || !foundHamlet); s++ {
		for y := range rows {
			for x := range consts.Chunk(8) {
				c := worldstream.ChunkCoord{X: x, Y: y}
				// 市街地に上書きされないチャンクだけ見る
				if _, _, ok := cityChunkInfo(s, c, rows); ok {
					continue
				}
				switch ChunkPlace(s, c, rows) {
				case GlyphRuin:
					foundRuin = true
				case GlyphVillage:
					foundVillage = true
				case GlyphHamlet:
					foundHamlet = true
				}
			}
		}
	}
	assert.True(t, foundRuin, "遺跡入口チャンクが > で出る")
	assert.True(t, foundVillage, "村の集落が村の文字で出る")
	assert.True(t, foundHamlet, "一軒家の集落が一軒家の文字で出る。開始特例で常に村になる退行の検知")
}

func TestSchematicLegend_全ての施設種別を含む(t *testing.T) {
	t.Parallel()

	legend := SchematicLegend()
	for _, g := range FacilityGlyphs() {
		assert.Truef(t, strings.ContainsRune(legend, g.Label), "凡例に %s(%c) がある", g.Name, g.Label)
	}
}
