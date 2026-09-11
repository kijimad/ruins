package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugStarveHunger は満腹度を0にすること、プレイヤー不在ではエラーを返すことを検証する。
func TestDebugStarveHunger(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 事前に満腹にしておく
	hunger := world.Components.Hunger.Get(player)
	hunger.Current = hunger.Max
	require.Positive(t, hunger.Current)

	require.NoError(t, debugStarveHunger(world))
	assert.Zero(t, world.Components.Hunger.Get(player).Current, "満腹度は0になる")

	// プレイヤー不在ではエラー
	empty := testutil.InitTestWorld(t)
	assert.Error(t, debugStarveHunger(empty))
}

// TestDebugExhaustFatigue は疲労を上限まで上げること、プレイヤー不在ではエラーを返すことを検証する。
func TestDebugExhaustFatigue(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	require.NoError(t, debugExhaustFatigue(world))
	fatigue := world.Components.Fatigue.Get(player)
	assert.Equal(t, fatigue.Max, fatigue.Current, "疲労は上限まで上がる")

	empty := testutil.InitTestWorld(t)
	assert.Error(t, debugExhaustFatigue(empty))
}

// TestDebugInflictConditions は主要部位へ状態異常を付与すること、プレイヤー不在ではエラーを返すことを検証する。
func TestDebugInflictConditions(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	require.NoError(t, debugInflictConditions(world))

	hs := world.Components.HealthStatus.Get(player)
	assert.True(t, hasCondition(hs, gc.BodyPartArms, gc.ConditionFracture), "腕に骨折を付与する")
	assert.True(t, hasCondition(hs, gc.BodyPartTorso, gc.ConditionLiverIllness), "胴に肝疾患を付与する")

	empty := testutil.InitTestWorld(t)
	assert.Error(t, debugInflictConditions(empty))
}

// TestDebugTreatAllConditions は付与済みの全状態異常へ良質な治療を適用することを検証する。
func TestDebugTreatAllConditions(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 先に状態異常を付与してから治療する
	require.NoError(t, debugInflictConditions(world))
	require.NoError(t, debugTreatAllConditions(world))

	hs := world.Components.HealthStatus.Get(player)
	treated := 0
	for p := range hs.Parts {
		for _, c := range hs.Parts[p].Conditions {
			assert.EqualValues(t, 150, c.TendQuality, "全条件に良質な治療が入る")
			treated++
		}
	}
	assert.Positive(t, treated, "治療対象の条件が存在する")

	empty := testutil.InitTestWorld(t)
	assert.Error(t, debugTreatAllConditions(empty))
}

// TestPlayerGridElement はプレイヤーの座標を返すこと、プレイヤー不在ではエラーを返すことを検証する。
func TestPlayerGridElement(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 7}, "ash")
	require.NoError(t, err)

	grid, err := playerGridElement(world)
	require.NoError(t, err)
	require.NotNil(t, grid)
	assert.Equal(t, consts.Tile(5), grid.X)
	assert.Equal(t, consts.Tile(7), grid.Y)

	empty := testutil.InitTestWorld(t)
	_, err = playerGridElement(empty)
	assert.Error(t, err)
}

// hasCondition は指定部位に指定種別の状態異常があるかを返す。
func hasCondition(hs *gc.HealthStatus, part gc.BodyPart, ct gc.ConditionType) bool {
	for _, c := range hs.Parts[part].Conditions {
		if c.Type == ct {
			return true
		}
	}
	return false
}
