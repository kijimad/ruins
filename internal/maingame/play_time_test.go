package maingame

import (
	"testing"
	"time"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccumulatePlayTime_初回は基準だけ置く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	game := &MainGame{World: world}

	game.accumulatePlayTime()

	assert.Zero(t, query.GetPlayTime(world).Duration, "初回は加算しない")
	assert.False(t, game.lastPlayTick.IsZero(), "基準は置く")
}

func TestAccumulatePlayTime_ラン外は数えない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	game := &MainGame{World: world}
	game.lastPlayTick = time.Now().Add(-time.Hour)

	game.accumulatePlayTime()

	assert.Zero(t, query.GetPlayTime(world).Duration, "プレイヤー不在では加算しない")
}

func TestAccumulatePlayTime_ラン中は加算する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	game := &MainGame{World: world}
	game.lastPlayTick = time.Now().Add(-time.Hour)

	game.accumulatePlayTime()

	assert.Greater(t, query.GetPlayTime(world).Duration, 50*time.Minute, "プレイヤー在では経過実時間を足す")
}
