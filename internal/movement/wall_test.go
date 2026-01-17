package movement

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerMovementWithWalls(t *testing.T) {
	t.Parallel()

	t.Run("壁がない方向への移動は可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを(10, 10)にスポーン
		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// プレイヤーの右側(11, 10)に壁を配置
		_, err = worldhelper.SpawnTile(world, "Wall", 11, 10, nil)
		require.NoError(t, err)

		// プレイヤーの上側(10, 9)に壁を配置
		_, err = worldhelper.SpawnTile(world, "Wall", 10, 9, nil)
		require.NoError(t, err)

		// 左側(9, 10)への移動は可能なはず
		canMove := CanMoveTo(world, 9, 10, player)
		assert.True(t, canMove, "左側への移動は可能なはず")

		// 下側(10, 11)への移動は可能なはず
		canMove = CanMoveTo(world, 10, 11, player)
		assert.True(t, canMove, "下側への移動は可能なはず")
	})

	t.Run("壁がある方向への移動は不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを(10, 10)にスポーン
		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// プレイヤーの右側(11, 10)に壁を配置
		_, err = worldhelper.SpawnTile(world, "Wall", 11, 10, nil)
		require.NoError(t, err)

		// プレイヤーの上側(10, 9)に壁を配置
		_, err = worldhelper.SpawnTile(world, "Wall", 10, 9, nil)
		require.NoError(t, err)

		// 右側(11, 10)への移動は壁によってブロックされるはず
		canMove := CanMoveTo(world, 11, 10, player)
		assert.False(t, canMove, "右側の壁に移動は不可なはず")

		// 上側(10, 9)への移動は壁によってブロックされるはず
		canMove = CanMoveTo(world, 10, 9, player)
		assert.False(t, canMove, "上側の壁に移動は不可なはず")
	})

	t.Run("プレイヤーが壁に完全に囲まれた場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを(10, 10)にスポーン
		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// 全方向に壁を配置
		_, err = worldhelper.SpawnTile(world, "Wall", 11, 10, nil) // 右
		require.NoError(t, err)
		_, err = worldhelper.SpawnTile(world, "Wall", 10, 9, nil) // 上
		require.NoError(t, err)
		_, err = worldhelper.SpawnTile(world, "Wall", 9, 10, nil) // 左
		require.NoError(t, err)
		_, err = worldhelper.SpawnTile(world, "Wall", 10, 11, nil) // 下
		require.NoError(t, err)

		// 全方向への移動が不可能になるはず
		directions := []struct {
			name string
			x, y int
		}{
			{"右", 11, 10},
			{"左", 9, 10},
			{"上", 10, 9},
			{"下", 10, 11},
		}

		for _, dir := range directions {
			canMove := CanMoveTo(world, dir.x, dir.y, player)
			assert.False(t, canMove, "Direction %s への移動は壁によってブロックされるはず", dir.name)
		}
	})
}
