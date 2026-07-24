package overworld

import (
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findCityAnchor は幅 wantWidth の市街地アンカーを持つ seed を探す。大都市(幅3)は
// 骨董品店・診療所・研究施設のレア施設を含むので、俯瞰図の確認に向く。
func findCityAnchor(t *testing.T, wantWidth consts.Chunk) (uint64, worldstream.ChunkCoord) {
	t.Helper()
	const rows = 1
	for s := uint64(1); s < 2000; s++ {
		for x := consts.Chunk(0); x < 12; x++ {
			c := worldstream.ChunkCoord{X: x}
			if !urbanPlacement.At(s, c, rows) {
				continue
			}
			if urbanWidthOf(ChunkSeed2D(s^urbanSalt, c.X, c.Y)) == wantWidth {
				return s, c
			}
		}
	}
	require.Failf(t, "seed未発見", "幅%dの市街地を持つ seed が見つからない", wantWidth)
	return 0, worldstream.ChunkCoord{}
}

func TestChunkSchematic_市街地の建物が施設種別の文字で塗られる(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 50, 50
	seed, anchor := findCityAnchor(t, 3)

	region := RegionSchematic(seed, anchor, 3, 1, chunkW, chunkH)
	joined := strings.Join(region, "\n")

	// 施設の文字が少なくとも数種類は現れる。原野と建物が塗り分けられている
	assert.Contains(t, joined, string(glyphField), "原野のマスがある")
	distinct := 0
	for k := range facilityGlyphs {
		if strings.ContainsRune(joined, facilityGlyphs[k].label) {
			distinct++
		}
	}
	assert.GreaterOrEqualf(t, distinct, 2, "複数種別の施設が種別文字で現れる。実際 %d 種", distinct)

	// 俯瞰図と凡例を目視できるようログに出す
	t.Logf("seed=%d 市街地アンカー=%v 幅3\n%s\n\n凡例: %s", seed, anchor, joined, SchematicLegend())
}

func TestChunkSchematic_純関数で決定的(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 50, 50
	seed, anchor := findCityAnchor(t, 3)
	a := RegionSchematic(seed, anchor, 3, 1, chunkW, chunkH)
	b := RegionSchematic(seed, anchor, 3, 1, chunkW, chunkH)
	assert.Equal(t, a, b, "同じ seed と座標なら俯瞰図は完全に一致する")
}

func TestChunkSchematic_遺跡入口と集落が地物の文字で出る(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 50, 50
	const rows = 1

	foundRuin, foundSettlement := false, false
	for s := uint64(1); s < 200 && (!foundRuin || !foundSettlement); s++ {
		for x := consts.Chunk(0); x < 8; x++ {
			c := worldstream.ChunkCoord{X: x}
			// 市街地に上書きされないチャンクだけ見る
			if _, _, ok := urbanAnchorOf(s, c, rows); ok {
				continue
			}
			joined := strings.Join(ChunkSchematic(s, c, rows, chunkW, chunkH), "\n")
			if ruinPlacement.At(s, c, rows) {
				assert.Contains(t, joined, string(glyphRuin), "遺跡入口チャンクに > が出る")
				foundRuin = true
			}
			if settlementPlacement.At(s, c, rows) {
				hasVillage := strings.ContainsRune(joined, glyphVillage)
				hasHamlet := strings.ContainsRune(joined, glyphHamlet)
				assert.Truef(t, hasVillage || hasHamlet, "集落チャンクに村か一軒家の文字が出る")
				foundSettlement = true
			}
		}
	}
	assert.True(t, foundRuin, "前提: 遺跡入口チャンクが見つかる")
	assert.True(t, foundSettlement, "前提: 集落チャンクが見つかる")
}
