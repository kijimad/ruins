package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugInvincibleSystem_致死の一撃で倒れたプレイヤーを満たして起こす(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Config.DebugInvincible = true
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "bat")
	require.NoError(t, err)
	world.Components.HealthStatus.Get(player).BodyTempOffset = -3
	world.Components.HealthStatus.Get(player).Parts[gc.BodyPartArms].AddCondition(gc.HealthCondition{Type: gc.ConditionType("fracture"), Severity: gc.SeveritySevere})
	world.Components.Hunger.Get(player).Current = 0
	fatigue := world.Components.Fatigue.Get(player)
	fatigue.Current = fatigue.Max

	gameaction.ApplyDamage(world, player, world.Components.HP.Get(player).Max+100, enemy)
	gameaction.ApplyDamage(world, enemy, 1, player)
	require.True(t, world.Components.Dead.Has(player))

	if world.Components.StatsChanged.Has(player) {
		world.Components.StatsChanged.Remove(player)
	}

	require.NoError(t, (&DebugInvincibleSystem{}).Update(world))
	hp := world.Components.HP.Get(player)
	hunger := world.Components.Hunger.Get(player)
	assert.False(t, world.Components.Dead.Has(player))
	assert.Equal(t, hp.Max, hp.Current)
	assert.Zero(t, world.Components.HealthStatus.Get(player).BodyTempOffset)
	for _, part := range world.Components.HealthStatus.Get(player).Parts {
		assert.Empty(t, part.Conditions, "怪我も治す")
	}
	assert.True(t, world.Components.StatsChanged.Has(player), "不調を消したら能力値を計算し直させる")
	assert.Equal(t, hunger.Max, hunger.Current)
	assert.Zero(t, world.Components.Fatigue.Get(player).Current)
	enemyHP := world.Components.HP.Get(enemy)
	assert.Equal(t, enemyHP.Max-1, enemyHP.Current, "敵は満たさない")
}

func TestDebugInvincibleSystem_無敵でなければ何もしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	world.Components.HP.Get(player).Current = 1
	world.Components.Dead.Add(player, &gc.Dead{})

	require.NoError(t, (&DebugInvincibleSystem{}).Update(world))
	assert.True(t, world.Components.Dead.Has(player))
	assert.Equal(t, 1, world.Components.HP.Get(player).Current)
}
