package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dayTurn は1から数える日付 day のその日の中のターン offset を、経過ターンにする
func dayTurn(day int, offset consts.Turn) consts.Turn {
	return consts.Turn(day-1)*gc.TurnsPerDay + offset
}

func TestRaidWarnsAt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		turn consts.Turn
		want bool
	}{
		{"7日目の夜明け", dayTurn(7, 0), true},
		{"14日目の夜明け", dayTurn(14, 0), true},
		{"7日目の夜明けの次のターン", dayTurn(7, 1), false},
		{"6日目の夜明け", dayTurn(6, 0), false},
		{"1日目の夜明け", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, query.RaidWarnsAt(tt.turn))
		})
	}
}

func TestRaidWaveAt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		turn   consts.Turn
		wave   int
		isWave bool
	}{
		{"7日目の1000は1つ目", dayTurn(7, 1000), 0, true},
		{"7日目の1150は2つ目", dayTurn(7, 1150), 1, true},
		{"7日目の1300は3つ目", dayTurn(7, 1300), 2, true},
		{"14日目の1000は1つ目", dayTurn(14, 1000), 0, true},
		{"7日目の999は波でない", dayTurn(7, 999), 0, false},
		{"7日目の1001は波でない", dayTurn(7, 1001), 0, false},
		{"7日目の0は波でない", dayTurn(7, 0), 0, false},
		{"6日目の1000は波でない", dayTurn(6, 1000), 0, false},
		{"8日目の1000は波でない", dayTurn(8, 1000), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wave, ok := query.RaidWaveAt(tt.turn)
			assert.Equal(t, tt.isWave, ok)
			assert.Equal(t, tt.wave, wave)
		})
	}
}

func TestNextRaidWaveTurn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		turn consts.Turn
		want consts.Turn
	}{
		{"1日目からは7日目の夜", 0, dayTurn(7, 1000)},
		{"7日目の昼からは今夜", dayTurn(7, 500), dayTurn(7, 1000)},
		{"7日目の最初の波のちょうどは今夜", dayTurn(7, 1000), dayTurn(7, 1000)},
		{"7日目の最初の波を過ぎたら14日目の夜", dayTurn(7, 1001), dayTurn(14, 1000)},
		{"8日目からは14日目の夜", dayTurn(8, 0), dayTurn(14, 1000)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, query.NextRaidWaveTurn(tt.turn))
		})
	}
}

func TestRaidTargets(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}

	t.Run("キューブは畳んでいても的にする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, at(5, 5), "ash")
		require.NoError(t, err)
		_, err = lifecycle.SpawnCube(world, at(10, 10))
		require.NoError(t, err)
		assert.Equal(t, []consts.Coord[consts.Tile]{at(10, 10)}, query.RaidTargets(world))
	})

	t.Run("キューブが無ければプレイヤーを的にする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, at(5, 5), "ash")
		require.NoError(t, err)
		assert.Equal(t, []consts.Coord[consts.Tile]{at(5, 5)}, query.RaidTargets(world))
	})

	t.Run("どちらも無ければ空", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		assert.Empty(t, query.RaidTargets(world))
	})
}
