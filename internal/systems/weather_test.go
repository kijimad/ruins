package systems

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

// TestWeatherSystem_スペルが尽きると次の天候へ遷移する は、runTurnEndSystems から呼ばれる WeatherSystem が
// 実際にスペルを進めることを固定する。登録漏れで天候が一度も動かない回帰を捕まえる。
func TestWeatherSystem_スペルが尽きると次の天候へ遷移する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	query.EnsureSeamlessBand(world)

	weather := query.GetWeather(world)
	weather.Current = gc.WeatherClear
	weather.UntilTurn = 0
	query.GetGameTime(world).TotalTurns = 5000

	require.NoError(t, (&WeatherSystem{}).Update(world))

	assert.Greater(t, query.GetWeather(world).UntilTurn, consts.Turn(5000), "尽きたスペルを次の長さへ進める")
}

// TestWeatherSystem_スペルが続く間は遷移しない は、UntilTurn に達していなければ天候を保つことを固定する。
func TestWeatherSystem_スペルが続く間は遷移しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	query.EnsureSeamlessBand(world)

	weather := query.GetWeather(world)
	weather.Current = gc.WeatherSnow
	weather.UntilTurn = 10000
	query.GetGameTime(world).TotalTurns = 5000

	require.NoError(t, (&WeatherSystem{}).Update(world))

	assert.Equal(t, gc.WeatherSnow, query.GetWeather(world).Current, "スペルが続く間は天候を保つ")
	assert.Equal(t, consts.Turn(10000), query.GetWeather(world).UntilTurn, "残り終端も変えない")
}
