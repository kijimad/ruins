package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findBackpackItem は owner の背嚢から name のアイテムを1つ返す。見つからなければ nil
func findBackpackItem(world w.World, owner ecs.Entity, name string) *ecs.Entity {
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		item := q.Entity()
		if world.Components.LocationInBackpack.Get(item).Owner != owner {
			continue
		}
		if nameComp := world.Components.Name.Get(item); nameComp != nil && nameComp.Name == name {
			q.Close()
			return &item
		}
	}
	return nil
}

func TestTransferActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("アイテムと受取人が指定されていれば検証を通過する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "木刀", 5, 5, 1)
		require.NoError(t, err)
		err = lifecycle.MoveToBackpack(world, item, member)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTransfer,
			Target:       &item,
			Recipient:    &leader,
		}

		ta := &TransferActivity{}
		err = ta.Validate(comp, member, world)
		assert.NoError(t, err)
	})

	t.Run("Targetが指定されていない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTransfer,
			Recipient:    &leader,
		}

		ta := &TransferActivity{}
		err = ta.Validate(comp, leader, world)
		assert.Error(t, err)
	})

	t.Run("Recipientが指定されていない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)
		err = lifecycle.MoveToBackpack(world, item, player)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTransfer,
			Target:       &item,
		}

		ta := &TransferActivity{}
		err = ta.Validate(comp, player, world)
		assert.Error(t, err)
	})
}

func TestTransferActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("アイテムが受取人のバックパックに移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "木刀", 5, 5, 1)
		require.NoError(t, err)
		err = lifecycle.MoveToBackpack(world, item, member)
		require.NoError(t, err)

		loc := world.Components.LocationInBackpack.Get(item)
		assert.Equal(t, member, loc.Owner)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTransfer,
			State:        gc.ActivityStateRunning,
			Target:       &item,
			Recipient:    &leader,
			TurnsTotal:   1,
			TurnsLeft:    1,
		}

		ta := &TransferActivity{Target: item, Recipient: leader, Count: 1}
		err = ta.DoTurn(comp, member, world)
		require.NoError(t, err)

		loc = world.Components.LocationInBackpack.Get(item)
		assert.Equal(t, leader, loc.Owner)
	})

	t.Run("Count1は1個だけ渡し主体を所有者にする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		// リーダーの共有プールにパンを3個持たせる
		pool, err := lifecycle.SpawnFieldItem(world, "パン", 5, 5, 3)
		require.NoError(t, err)
		require.NoError(t, lifecycle.MoveToBackpack(world, pool, leader))
		require.Equal(t, 3, world.Components.Stackable.Get(pool).Count)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTransfer,
			State:        gc.ActivityStateRunning,
			Target:       &pool,
			Recipient:    &member,
			TurnsTotal:   1,
			TurnsLeft:    1,
		}
		// アクターは受け取る隊員。丸ごとでなく1個だけ引く
		ta := &TransferActivity{Target: pool, Recipient: member, Count: 1}
		require.NoError(t, ta.DoTurn(comp, member, world))

		// 元スタックは1減り、隊員は1個だけ受け取る
		assert.Equal(t, 2, world.Components.Stackable.Get(pool).Count, "プールは1個ずつ減る")
		memberBread := findBackpackItem(world, member, "パン")
		require.NotNil(t, memberBread, "隊員がパンを受け取る")
		assert.Equal(t, 1, world.Components.Stackable.Get(*memberBread).Count, "受け取りは1個")

		// 主体はアクターの隊員でなく食料の所有者リーダー。自己転送の誤ログにならないこと
		recent := query.GetGameLog(world).GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "Ash", "渡す主体はリーダー")
		assert.Contains(t, recent[0], "隊員A に渡した", "受取人は隊員")
	})

	t.Run("Countは指定個数だけ分割して渡す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		// リーダーの共有プールにパンを5個持たせ、うち2個だけ渡す
		pool, err := lifecycle.SpawnFieldItem(world, "パン", 5, 5, 5)
		require.NoError(t, err)
		require.NoError(t, lifecycle.MoveToBackpack(world, pool, leader))

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTransfer,
			State:        gc.ActivityStateRunning,
			Target:       &pool,
			Recipient:    &member,
			TurnsTotal:   1,
			TurnsLeft:    1,
		}
		ta := &TransferActivity{Target: pool, Recipient: member, Count: 2}
		require.NoError(t, ta.DoTurn(comp, member, world))

		assert.Equal(t, 3, world.Components.Stackable.Get(pool).Count, "プールは指定個数ぶん減る")
		memberBread := findBackpackItem(world, member, "パン")
		require.NotNil(t, memberBread, "隊員がパンを受け取る")
		assert.Equal(t, 2, world.Components.Stackable.Get(*memberBread).Count, "受け取りは2個")

		// ログは在庫全体でなく渡した個数で表示する
		recent := query.GetGameLog(world).GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "パン(2個)", "ログは渡した個数を表示する")
	})
}

func TestTransferActivity_Info(t *testing.T) {
	t.Parallel()
	ta := &TransferActivity{}
	info := ta.Info()
	assert.Equal(t, "転送", info.Name)
	assert.Equal(t, gc.BehaviorTransfer, ta.Name())
}

func testAbilities() gc.Abilities {
	return gc.Abilities{
		Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
		Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
		Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
	}
}
