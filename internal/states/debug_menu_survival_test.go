package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugToggleInvincible_無敵にするとHPと体温と空腹と疲労を戻す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	world.Components.HP.Get(player).Current = 1
	world.Components.HealthStatus.Get(player).BodyTempOffset = -3
	world.Components.Hunger.Get(player).Current = 0
	fatigue := world.Components.Fatigue.Get(player)
	fatigue.Current = fatigue.Max

	require.NoError(t, debugToggleInvincible(world))
	assert.True(t, world.Resources.Config.DebugInvincible)
	hp := world.Components.HP.Get(player)
	hunger := world.Components.Hunger.Get(player)
	assert.Equal(t, hp.Max, hp.Current)
	assert.Zero(t, world.Components.HealthStatus.Get(player).BodyTempOffset)
	assert.Equal(t, hunger.Max, hunger.Current)
	assert.Zero(t, world.Components.Fatigue.Get(player).Current)

	require.NoError(t, debugToggleInvincible(world))
	assert.False(t, world.Resources.Config.DebugInvincible)
}

func TestDebugKillNearbyEnemies_近くの敵だけを倒す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	near, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10 + debugKillNearbyEnemiesRange, Y: 10}, "bat")
	require.NoError(t, err)
	far, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11 + debugKillNearbyEnemiesRange, Y: 10}, "bat")
	require.NoError(t, err)

	require.NoError(t, debugKillNearbyEnemies(world))
	assert.True(t, world.Components.Dead.Has(near))
	assert.False(t, world.Components.Dead.Has(far))
}
