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

// TestToLocal_絶対Xを帯ローカルへ は「絶対 X を帯内のローカル X に落とす」変換を固定する。
func TestToLocal_絶対Xを帯ローカルへ(t *testing.T) {
	t.Parallel()

	origin := worldstream.BandOriginX(5, 100) // 帯ローカル0 = 絶対500
	absX := consts.AbsTileX(540)              // 帯の40タイル目

	assert.Equal(t, consts.Tile(40), worldstream.ToLocal(origin, absX),
		"絶対540は帯ローカル40に写る")
}
