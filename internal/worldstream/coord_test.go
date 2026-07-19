package worldstream_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/stretchr/testify/assert"
)

func TestBandOriginX(t *testing.T) {
	t.Parallel()

	assert.Equal(t, consts.AbsTileX(0), worldstream.BandOriginX(0, 100), "eastIndex=0 は原点0")
	assert.Equal(t, consts.AbsTileX(300), worldstream.BandOriginX(3, 100), "eastIndex*chunkW")
}

func TestAbsLocalRoundTrip(t *testing.T) {
	t.Parallel()

	origin := worldstream.BandOriginX(2, 100) // 絶対原点 200

	abs := worldstream.ToAbs(origin, 37) // 200 + 37
	assert.Equal(t, consts.AbsTileX(237), abs, "ローカル→絶対はオフセット加算")

	local := worldstream.ToLocal(origin, abs)
	assert.Equal(t, consts.Tile(37), local, "絶対→ローカルで元に戻る")
}

// TestToLocal_前線を帯ローカルへ は「絶対 X の前線を帯内のローカル X に落とす」用途を固定する。
func TestToLocal_前線を帯ローカルへ(t *testing.T) {
	t.Parallel()

	origin := worldstream.BandOriginX(5, 100) // 帯ローカル0 = 絶対500
	frontEast := consts.AbsTileX(540)         // 帯の40タイル目に前線がある

	assert.Equal(t, consts.Tile(40), worldstream.ToLocal(origin, frontEast),
		"絶対540は帯ローカル40に写る")
}
