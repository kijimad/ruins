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

func TestMoveActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効な移動先の場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			Destination:  &gc.GridElement{X: 11, Y: 10},
		}

		ma := &MoveActivity{}
		err = ma.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("移動先がnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			Destination:  nil,
		}

		ma := &MoveActivity{}
		err = ma.Validate(comp, player, world)
		require.Error(t, err)
		assert.Equal(t, ErrMoveTargetNotSet, err)
	})

	t.Run("位置情報がない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 位置情報なしのプレイヤーを手動で作成
		player := world.World.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			Destination:  &gc.GridElement{X: 11, Y: 10},
		}

		ma := &MoveActivity{}
		err := ma.Validate(comp, player, world)
		require.Error(t, err)
		assert.Equal(t, ErrMoveNoGridElement, err)
	})
}

func TestMoveActivity_Info(t *testing.T) {
	t.Parallel()

	ma := &MoveActivity{}
	info := ma.Info()

	assert.Equal(t, "移動", info.Name)
	assert.False(t, info.Interruptible)
	assert.False(t, info.Resumable)
}

func TestMoveActivity_Name(t *testing.T) {
	t.Parallel()

	ma := &MoveActivity{}
	assert.Equal(t, gc.BehaviorMove, ma.Name())
}

func TestMoveActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("正常に移動して完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			State:        gc.ActivityStateRunning,
			Destination:  &gc.GridElement{X: 11, Y: 10},
		}

		ma := &MoveActivity{}
		err = ma.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)

		// 移動していることを確認
		gridElement := world.Components.GridElement.Get(player)
		assert.Equal(t, 11, int(gridElement.X))
		assert.Equal(t, 10, int(gridElement.Y))
	})

	t.Run("移動先がnilの場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			State:        gc.ActivityStateRunning,
			Destination:  nil,
		}

		ma := &MoveActivity{}
		err = ma.DoTurn(comp, player, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})

	t.Run("位置情報がない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 位置情報なしのプレイヤーを手動で作成
		player := world.World.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			State:        gc.ActivityStateRunning,
			Destination:  &gc.GridElement{X: 11, Y: 10},
		}

		ma := &MoveActivity{}
		err := ma.DoTurn(comp, player, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})

	t.Run("プレイヤーが隊員のいるタイルに移動すると位置が入れ替わる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		playerX, playerY := 10, 10
		player, err := lifecycle.SpawnPlayer(world, playerX, playerY, "Ash")
		require.NoError(t, err)

		abilities := gc.Abilities{
			Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
			Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
			Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
		}
		member, err := lifecycle.SpawnSquadMember(world, player, "隊員", abilities, "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		origMemberX, origMemberY := int(memberGrid.X), int(memberGrid.Y)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			State:        gc.ActivityStateRunning,
			Destination:  &gc.GridElement{X: memberGrid.X, Y: memberGrid.Y},
		}

		ma := &MoveActivity{}
		err = ma.DoTurn(comp, player, world)
		require.NoError(t, err)

		// プレイヤーが隊員の元位置に移動している
		playerGrid := world.Components.GridElement.Get(player)
		assert.Equal(t, origMemberX, int(playerGrid.X))
		assert.Equal(t, origMemberY, int(playerGrid.Y))

		// 隊員がプレイヤーの元位置に移動している
		memberGrid = world.Components.GridElement.Get(member)
		assert.Equal(t, playerX, int(memberGrid.X))
		assert.Equal(t, playerY, int(memberGrid.Y))
	})
}

func TestCanSwapPosition(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーは隊員と位置交換できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.World.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		member := world.World.NewEntity()
		world.Components.SquadMember.Add(member, &gc.SquadMember{})

		assert.True(t, CanSwapPosition(world, player, member))
	})

	t.Run("プレイヤーは敵と位置交換できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.World.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		enemy := world.World.NewEntity()
		world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemyData{})

		assert.False(t, CanSwapPosition(world, player, enemy))
	})

	t.Run("隊員は他の隊員と位置交換できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member1 := world.World.NewEntity()
		world.Components.SquadMember.Add(member1, &gc.SquadMember{})

		member2 := world.World.NewEntity()
		world.Components.SquadMember.Add(member2, &gc.SquadMember{})

		assert.False(t, CanSwapPosition(world, member1, member2))
	})

	t.Run("隊員はプレイヤーと位置交換できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := world.World.NewEntity()
		world.Components.SquadMember.Add(member, &gc.SquadMember{})

		player := world.World.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		assert.False(t, CanSwapPosition(world, member, player))
	})

	t.Run("敵は隊員と位置交換できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		enemy := world.World.NewEntity()
		world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemyData{})

		member := world.World.NewEntity()
		world.Components.SquadMember.Add(member, &gc.SquadMember{})

		assert.False(t, CanSwapPosition(world, enemy, member))
	})
}

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

		from := consts.Coord[int]{X: 10, Y: 10}

		// 左側(9, 10)への移動は可能なはず
		canMove := CanMoveTo(world, consts.Coord[int]{X: 9, Y: 10}, from, player)
		assert.True(t, canMove, "左側への移動は可能なはず")

		// 下側(10, 11)への移動は可能なはず
		canMove = CanMoveTo(world, consts.Coord[int]{X: 10, Y: 11}, from, player)
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

		from := consts.Coord[int]{X: 10, Y: 10}

		// 右側(11, 10)への移動は壁によってブロックされるはず
		canMove := CanMoveTo(world, consts.Coord[int]{X: 11, Y: 10}, from, player)
		assert.False(t, canMove, "右側の壁に移動は不可なはず")

		// 上側(10, 9)への移動は壁によってブロックされるはず
		canMove = CanMoveTo(world, consts.Coord[int]{X: 10, Y: 9}, from, player)
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

		from := consts.Coord[int]{X: 10, Y: 10}

		// 全方向への移動が不可能になるはず
		directions := []struct {
			name string
			to   consts.Coord[int]
		}{
			{"右", consts.Coord[int]{X: 11, Y: 10}},
			{"左", consts.Coord[int]{X: 9, Y: 10}},
			{"上", consts.Coord[int]{X: 10, Y: 9}},
			{"下", consts.Coord[int]{X: 10, Y: 11}},
		}

		for _, dir := range directions {
			canMove := CanMoveTo(world, dir.to, from, player)
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
		canMove := CanMoveTo(world, consts.Coord[int]{X: 11, Y: 9}, consts.Coord[int]{X: 10, Y: 10}, player)
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

		memberGrid := world.Components.GridElement.Get(member)
		memberX, memberY := int(memberGrid.X), int(memberGrid.Y)

		canMove := CanMoveTo(world, consts.Coord[int]{X: memberX, Y: memberY}, consts.Coord[int]{X: 10, Y: 10}, player)
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

		memberGrid := world.Components.GridElement.Get(member)
		memberX, memberY := int(memberGrid.X), int(memberGrid.Y)

		// AIエンティティを手動で作成する
		aiEntity := world.World.NewEntity()
		world.Components.SoloAI.Add(aiEntity, &gc.SoloAI{})
		world.Components.GridElement.Add(aiEntity, &gc.GridElement{
			X: consts.Tile(memberX + 1), Y: consts.Tile(memberY),
		})

		// エンティティ追加後にSpatialIndexを再構築させる
		query.InvalidateSpatialIndex(world)

		canMove := CanMoveTo(world, consts.Coord[int]{X: memberX, Y: memberY}, consts.Coord[int]{X: memberX + 1, Y: memberY}, aiEntity)
		assert.False(t, canMove, "AI側からは隊員のタイルに移動できない")
	})

	t.Run("隊員は他の隊員がいるタイルに移動できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		abilities := gc.Abilities{
			Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
			Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
			Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
		}
		member1, err := lifecycle.SpawnSquadMember(world, player, "隊員A", abilities, "player")
		require.NoError(t, err)
		member2, err := lifecycle.SpawnSquadMember(world, player, "隊員B", abilities, "player")
		require.NoError(t, err)

		member1Grid := world.Components.GridElement.Get(member1)
		member1Grid.X = consts.Tile(11)
		member1Grid.Y = consts.Tile(10)
		member2Grid := world.Components.GridElement.Get(member2)
		member2Grid.X = consts.Tile(12)
		member2Grid.Y = consts.Tile(10)

		query.InvalidateSpatialIndex(world)

		canMove := CanMoveTo(world, consts.Coord[int]{X: 11, Y: 10}, consts.Coord[int]{X: 12, Y: 10}, member2)
		assert.False(t, canMove, "隊員は他の隊員のタイルに移動できない")
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
		canMove := CanMoveTo(world, consts.Coord[int]{X: 11, Y: 9}, consts.Coord[int]{X: 10, Y: 10}, player)
		assert.True(t, canMove, "隣接1方向のみ壁なら斜め移動は可能")
	})
}
