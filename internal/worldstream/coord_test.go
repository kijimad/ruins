package worldstream_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/stretchr/testify/assert"
)

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
