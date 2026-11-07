package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 既定カメラでは上矢印が北、下矢印が南のタイルを指す
var (
	northOfPlayer = consts.Coord[consts.Tile]{X: 10, Y: 9}
	southOfPlayer = consts.Coord[consts.Tile]{X: 10, Y: 11}
)

func TestStartDirectionAction(t *testing.T) {
	t.Parallel()

	t.Run("候補が無ければ行動しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		trans, err := startDirectionAction(world, attackAction(world))
		require.NoError(t, err)

		assert.Equal(t, es.TransNone, trans.Type)
		assert.Nil(t, activity.GetLastResult(player, world))
	})

	t.Run("候補が1つなら選ばせずに実行する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", northOfPlayer.X, northOfPlayer.Y)
		require.NoError(t, err)

		trans, err := startDirectionAction(world, attackAction(world))
		require.NoError(t, err)

		assert.Equal(t, es.TransNone, trans.Type)
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMelee, result.BehaviorName)
	})

	t.Run("候補が複数なら方向選択を開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", northOfPlayer.X, northOfPlayer.Y)
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", southOfPlayer.X, southOfPlayer.Y)
		require.NoError(t, err)

		trans, err := startDirectionAction(world, attackAction(world))
		require.NoError(t, err)

		assert.Equal(t, es.TransPush, trans.Type)
		assert.Nil(t, activity.GetLastResult(player, world))
	})
}

func TestDirectionSelectState_doAction(t *testing.T) {
	t.Parallel()

	t.Run("候補の向きを選ぶと実行して閉じる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", northOfPlayer.X, northOfPlayer.Y)
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", southOfPlayer.X, southOfPlayer.Y)
		require.NoError(t, err)
		options, err := directionOptions(world, attackAction(world))
		require.NoError(t, err)
		st := &DirectionSelectState{action: attackAction(world), options: options}

		trans, err := st.doAction(world, inputmapper.ActionMoveNorth)
		require.NoError(t, err)

		assert.Equal(t, es.TransPop, trans.Type)
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMelee, result.BehaviorName)
	})

	t.Run("候補でない向きは無視して開いたままにする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", northOfPlayer.X, northOfPlayer.Y)
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", southOfPlayer.X, southOfPlayer.Y)
		require.NoError(t, err)
		options, err := directionOptions(world, attackAction(world))
		require.NoError(t, err)
		st := &DirectionSelectState{action: attackAction(world), options: options}

		trans, err := st.doAction(world, inputmapper.ActionMoveEast)
		require.NoError(t, err)

		assert.Equal(t, es.TransNone, trans.Type)
		assert.Nil(t, activity.GetLastResult(player, world))
	})

	t.Run("Escでは行動せずに閉じる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", northOfPlayer.X, northOfPlayer.Y)
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", southOfPlayer.X, southOfPlayer.Y)
		require.NoError(t, err)
		options, err := directionOptions(world, attackAction(world))
		require.NoError(t, err)
		st := &DirectionSelectState{action: attackAction(world), options: options}

		trans, err := st.doAction(world, inputmapper.ActionCloseMenu)
		require.NoError(t, err)

		assert.Equal(t, es.TransPop, trans.Type)
		assert.Nil(t, activity.GetLastResult(player, world))
	})
}

func TestInteractableAt(t *testing.T) {
	t.Parallel()

	t.Run("指定の相互作用を受け付ける生存中の実体を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		enemy, err := lifecycle.SpawnEnemy(world, northOfPlayer, "fireball")
		require.NoError(t, err)

		got, ok := interactableAt(world, northOfPlayer, gc.InteractionMelee)

		require.True(t, ok)
		assert.Equal(t, enemy, got)
	})

	t.Run("死んだ実体は返さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		enemy, err := lifecycle.SpawnEnemy(world, northOfPlayer, "fireball")
		require.NoError(t, err)
		world.Components.Dead.Add(enemy, &gc.Dead{})

		_, ok := interactableAt(world, northOfPlayer, gc.InteractionMelee)

		assert.False(t, ok)
	})

	t.Run("相互作用の種類が違えば返さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnEnemy(world, northOfPlayer, "fireball")
		require.NoError(t, err)

		_, ok := interactableAt(world, northOfPlayer, gc.InteractionStorage)

		assert.False(t, ok)
	})
}
