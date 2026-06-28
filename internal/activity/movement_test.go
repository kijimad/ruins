package activity

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

func TestCanMoveTo(t *testing.T) {
	t.Parallel()

	t.Run("壁がない方向への移動は可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// プレイヤーの右側(11, 10)に壁を配置
		_, err = lifecycle.SpawnTile(world, "wall", 11, 10, nil)
		require.NoError(t, err)

		// プレイヤーの上側(10, 9)に壁を配置
		_, err = lifecycle.SpawnTile(world, "wall", 10, 9, nil)
		require.NoError(t, err)

		// 左側(9, 10)への移動は可能なはず
		canMove := CanMoveTo(world, 9, 10, 10, 10, player)
		assert.True(t, canMove, "左側への移動は可能なはず")

		// 下側(10, 11)への移動は可能なはず
		canMove = CanMoveTo(world, 10, 11, 10, 10, player)
		assert.True(t, canMove, "下側への移動は可能なはず")
	})

	t.Run("壁がある方向への移動は不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// プレイヤーの右側(11, 10)に壁を配置
		_, err = lifecycle.SpawnTile(world, "wall", 11, 10, nil)
		require.NoError(t, err)

		// プレイヤーの上側(10, 9)に壁を配置
		_, err = lifecycle.SpawnTile(world, "wall", 10, 9, nil)
		require.NoError(t, err)

		// 右側(11, 10)への移動は壁によってブロックされるはず
		canMove := CanMoveTo(world, 11, 10, 10, 10, player)
		assert.False(t, canMove, "右側の壁に移動は不可なはず")

		// 上側(10, 9)への移動は壁によってブロックされるはず
		canMove = CanMoveTo(world, 10, 9, 10, 10, player)
		assert.False(t, canMove, "上側の壁に移動は不可なはず")
	})

	t.Run("プレイヤーが壁に完全に囲まれた場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 全方向に壁を配置
		_, err = lifecycle.SpawnTile(world, "wall", 11, 10, nil) // 右
		require.NoError(t, err)
		_, err = lifecycle.SpawnTile(world, "wall", 10, 9, nil) // 上
		require.NoError(t, err)
		_, err = lifecycle.SpawnTile(world, "wall", 9, 10, nil) // 左
		require.NoError(t, err)
		_, err = lifecycle.SpawnTile(world, "wall", 10, 11, nil) // 下
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
			canMove := CanMoveTo(world, dir.x, dir.y, 10, 10, player)
			assert.False(t, canMove, "Direction %s への移動は壁によってブロックされるはず", dir.name)
		}
	})

	t.Run("斜め移動で隣接2方向が両方壁なら移動不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 右(11,10)と上(10,9)に壁を配置
		_, err = lifecycle.SpawnTile(world, "wall", 11, 10, nil)
		require.NoError(t, err)
		_, err = lifecycle.SpawnTile(world, "wall", 10, 9, nil)
		require.NoError(t, err)

		// 右上(11,9)への斜め移動は不可（右と上の両方が壁）
		canMove := CanMoveTo(world, 11, 9, 10, 10, player)
		assert.False(t, canMove, "隣接2方向が両方壁なら斜め移動は不可")
	})

	t.Run("プレイヤーは自分の隊員がいるタイルに移動できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		abilities := gc.Abilities{
			Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
			Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
			Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
		}
		member, err := lifecycle.SpawnSquadMember(world, player, "隊員", abilities, "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member).(*gc.GridElement)
		memberX, memberY := int(memberGrid.X), int(memberGrid.Y)

		canMove := CanMoveTo(world, memberX, memberY, 10, 10, player)
		assert.True(t, canMove, "プレイヤーは自分の隊員のタイルに移動できる")
	})

	t.Run("AIは隊員がいるタイルに移動できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		abilities := gc.Abilities{
			Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
			Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
			Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
		}
		member, err := lifecycle.SpawnSquadMember(world, player, "隊員", abilities, "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member).(*gc.GridElement)
		memberX, memberY := int(memberGrid.X), int(memberGrid.Y)

		// AIエンティティを手動で作成する
		aiEntity := world.Manager.NewEntity()
		aiEntity.AddComponent(world.Components.AIMoveFSM, &gc.AIMoveFSM{})
		aiEntity.AddComponent(world.Components.GridElement, &gc.GridElement{
			X: consts.Tile(memberX + 1), Y: consts.Tile(memberY),
		})

		// エンティティ追加後にSpatialIndexを再構築させる
		query.InvalidateSpatialIndex(world)

		canMove := CanMoveTo(world, memberX, memberY, memberX+1, memberY, aiEntity)
		assert.False(t, canMove, "AI側からは隊員のタイルに移動できない")
	})

	t.Run("斜め移動で隣接1方向のみ壁なら移動可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 右(11,10)にだけ壁を配置
		_, err = lifecycle.SpawnTile(world, "wall", 11, 10, nil)
		require.NoError(t, err)

		// 右上(11,9)への斜め移動は可能（上方向は空いている）
		canMove := CanMoveTo(world, 11, 9, 10, 10, player)
		assert.True(t, canMove, "隣接1方向のみ壁なら斜め移動は可能")
	})
}
