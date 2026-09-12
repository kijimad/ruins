package worldstream_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/stretchr/testify/assert"
)

// TestBandOriginY_SeamlessBandと一致する は worldstream.BandOriginY と
// components.SeamlessBand.BandOriginY が同一値であることを固定する。依存方向の都合で同じ式が
// 2箇所にあるので、片方だけ変えた乖離をここで機械的に検知する。
func TestBandOriginY_SeamlessBandと一致する(t *testing.T) {
	t.Parallel()

	cases := []struct {
		north  consts.Chunk
		chunkH consts.Tile
	}{
		{0, 100}, {1, 24}, {5, 30}, {9, 20},
	}
	for _, c := range cases {
		sb := gc.SeamlessBand{NorthIndex: c.north, ChunkH: c.chunkH}
		assert.Equalf(t, worldstream.BandOriginY(c.north, c.chunkH), sb.BandOriginY(),
			"north=%d chunkH=%d で両実装が一致する", c.north, c.chunkH)
	}
}

func TestBandOriginY(t *testing.T) {
	t.Parallel()

	assert.Equal(t, consts.AbsTileY(0), worldstream.BandOriginY(0, 100), "northIndex=0 は原点0")
	assert.Equal(t, consts.AbsTileY(-300), worldstream.BandOriginY(3, 100), "北は -Y なので -northIndex*chunkH")
}

func TestAbsLocalRoundTrip(t *testing.T) {
	t.Parallel()

	origin := worldstream.BandOriginY(2, 100) // 絶対原点 -200

	abs := worldstream.ToAbsY(origin, 37) // -200 + 37
	assert.Equal(t, consts.AbsTileY(-163), abs, "ローカル→絶対はオフセット加算")

	local := worldstream.ToLocalY(origin, abs)
	assert.Equal(t, consts.Tile(37), local, "絶対→ローカルで元に戻る")
}

// TestToLocalY_絶対Yを帯ローカルへ は「絶対 Y を帯内のローカル Y に落とす」変換を固定する。
func TestToLocalY_絶対Yを帯ローカルへ(t *testing.T) {
	t.Parallel()

	origin := worldstream.BandOriginY(5, 100) // 帯ローカル0 = 絶対-500
	absY := consts.AbsTileY(-460)             // 帯の40タイル目

	assert.Equal(t, consts.Tile(40), worldstream.ToLocalY(origin, absY),
		"絶対-460は帯ローカル40に写る")
}
