package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spawnTestPlayer はプレイヤーを1体置いた world を返す。
func spawnTestPlayer(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	return world
}

func TestDebugStarveHunger(t *testing.T) {
	t.Parallel()

	t.Run("満腹度を0にする", func(t *testing.T) {
		t.Parallel()
		world := spawnTestPlayer(t)
		player, err := query.GetPlayerEntity(world)
		require.NoError(t, err)

		// 事前に満腹にしておく
		hunger := world.Components.Hunger.Get(player)
		require.Positive(t, hunger.Max)
		hunger.Current = hunger.Max

		require.NoError(t, debugStarveHunger(world))
		assert.Zero(t, world.Components.Hunger.Get(player).Current, "満腹度は0になる")
	})

	t.Run("プレイヤー不在はエラーを返す", func(t *testing.T) {
		t.Parallel()
		assert.ErrorContains(t, debugStarveHunger(testutil.InitTestWorld(t)), "player")
	})
}

func TestDebugExhaustFatigue(t *testing.T) {
	t.Parallel()

	t.Run("疲労を上限まで上げる", func(t *testing.T) {
		t.Parallel()
		world := spawnTestPlayer(t)
		player, err := query.GetPlayerEntity(world)
		require.NoError(t, err)

		require.NoError(t, debugExhaustFatigue(world))
		fatigue := world.Components.Fatigue.Get(player)
		assert.Equal(t, fatigue.Max, fatigue.Current, "疲労は上限まで上がる")
	})

	t.Run("プレイヤー不在はエラーを返す", func(t *testing.T) {
		t.Parallel()
		assert.ErrorContains(t, debugExhaustFatigue(testutil.InitTestWorld(t)), "player")
	})
}

func TestDebugInflictConditions(t *testing.T) {
	t.Parallel()

	t.Run("主要部位へ状態異常を付与する", func(t *testing.T) {
		t.Parallel()
		world := spawnTestPlayer(t)
		player, err := query.GetPlayerEntity(world)
		require.NoError(t, err)

		require.NoError(t, debugInflictConditions(world))
		hs := world.Components.HealthStatus.Get(player)
		assert.True(t, hasCondition(hs, gc.BodyPartArms, gc.ConditionFracture), "腕に骨折を付与する")
		assert.True(t, hasCondition(hs, gc.BodyPartTorso, gc.ConditionLiverIllness), "胴に肝疾患を付与する")
	})

	t.Run("プレイヤー不在はエラーを返す", func(t *testing.T) {
		t.Parallel()
		assert.ErrorContains(t, debugInflictConditions(testutil.InitTestWorld(t)), "player")
	})
}

func TestDebugTreatAllConditions(t *testing.T) {
	t.Parallel()

	t.Run("付与済みの全状態異常へ良質な治療を適用する", func(t *testing.T) {
		t.Parallel()
		world := spawnTestPlayer(t)
		player, err := query.GetPlayerEntity(world)
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
	})

	t.Run("プレイヤー不在はエラーを返す", func(t *testing.T) {
		t.Parallel()
		assert.ErrorContains(t, debugTreatAllConditions(testutil.InitTestWorld(t)), "player")
	})
}

func TestPlayerGridElement(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーの座標を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 7}, "ash")
		require.NoError(t, err)

		grid, err := playerGridElement(world)
		require.NoError(t, err)
		require.NotNil(t, grid)
		assert.Equal(t, consts.Tile(5), grid.X)
		assert.Equal(t, consts.Tile(7), grid.Y)
	})

	t.Run("プレイヤー不在はエラーを返す", func(t *testing.T) {
		t.Parallel()
		_, err := playerGridElement(testutil.InitTestWorld(t))
		assert.ErrorContains(t, err, "player")
	})
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
