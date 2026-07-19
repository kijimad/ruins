package aiinput

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChebyshevDistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ax   int
		ay   int
		bx   int
		by   int
		want int
	}{
		{"同じ位置", 5, 5, 5, 5, 0},
		{"水平距離", 0, 0, 3, 0, 3},
		{"垂直距離", 0, 0, 0, 4, 4},
		{"斜め距離", 0, 0, 3, 3, 3},
		{"斜め距離で水平が大きい", 0, 0, 5, 3, 5},
		{"負の座標", 0, 0, -3, -4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(tt.ax), Y: consts.Tile(tt.ay)}}
			b := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(tt.bx), Y: consts.Tile(tt.by)}}
			assert.Equal(t, tt.want, gridDistance(a, b))
		})
	}
}

func TestShouldRetreatLowHP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current int
		max     int
		want    bool
	}{
		{"HP満タン", 100, 100, false},
		{"HP50%", 50, 100, false},
		{"HP26%", 26, 100, false},
		{"HP25%", 25, 100, true},
		{"HP10%", 10, 100, true},
		{"HP0%", 0, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hp := &gc.HP{Current: tt.current, Max: tt.max}
			result := hp.Max > 0 && hp.Current*100/hp.Max <= hpRetreatThreshold
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestNewSquadPlanner(t *testing.T) {
	t.Parallel()
	sp := newSquadPlanner(newTestRNG())
	assert.NotNil(t, sp)
	assert.NotNil(t, sp.logger)
	assert.NotNil(t, sp.visionSystem)
}

func TestTryMoveCloser(t *testing.T) {
	t.Parallel()

	t.Run("距離が縮まる方向にのみ移動する", func(t *testing.T) {
		t.Parallel()
		from := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}
		target := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 8, Y: 5}}
		currentDist := gridDistance(from, target) // 3

		dx := int(target.X) - int(from.X)
		dy := int(target.Y) - int(from.Y)
		candidates := calculateMoveCandidates(consts.Coord[consts.Tile]{X: consts.Tile(dx), Y: consts.Tile(dy)})

		assert.NotEmpty(t, candidates)
		bestCandidate := candidates[0]
		newGrid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: from.X + bestCandidate.X, Y: from.Y + bestCandidate.Y}}

		newDist := gridDistance(newGrid, target)
		assert.Less(t, newDist, currentDist, "最優先候補は距離を縮める")
	})

	t.Run("横移動では距離が縮まらないことを検出できる", func(t *testing.T) {
		t.Parallel()
		from := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}
		target := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 8, Y: 5}}
		currentDist := gridDistance(from, target) // 3

		sideGrid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 4}}
		sideDist := gridDistance(sideGrid, target)
		assert.GreaterOrEqual(t, sideDist, currentDist, "横移動は距離を縮めない")
	})
}

func testAbilities() gc.Abilities {
	return gc.Abilities{
		Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
		Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
		Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
	}
}

func TestPlanItemPickupAction(t *testing.T) {
	t.Parallel()

	t.Run("PolicyPickupで足元にアイテムがあれば拾得アクションを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)

		_, err = lifecycle.SpawnFieldItem(world, "木刀", memberGrid.X, memberGrid.Y, 1)
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{ItemPickup: gc.PolicyPickup, ViewDistance: 5},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		b, ok := sp.planItemPickupAction(world, member, ctx)
		assert.True(t, ok, "拾得アクションが返る")
		assert.NotNil(t, b)
	})

	t.Run("PolicyIgnoreではアイテムがあっても拾わない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)

		_, err = lifecycle.SpawnFieldItem(world, "木刀", memberGrid.X, memberGrid.Y, 1)
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{ItemPickup: gc.PolicyIgnore, ViewDistance: 5},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		_, ok := sp.planItemPickupAction(world, member, ctx)
		assert.False(t, ok, "PolicyIgnoreでは拾得しない")
	})

	t.Run("視界内にアイテムがなければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{ItemPickup: gc.PolicyPickup, ViewDistance: 5},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		_, ok := sp.planItemPickupAction(world, member, ctx)
		assert.False(t, ok, "アイテムがなければ何もしない")
	})

	t.Run("視界外のアイテムには反応しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)

		_, err = lifecycle.SpawnFieldItem(world, "木刀", memberGrid.X+10, memberGrid.Y+10, 1)
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{ItemPickup: gc.PolicyPickup, ViewDistance: 5},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		_, ok := sp.planItemPickupAction(world, member, ctx)
		assert.False(t, ok, "視界外のアイテムには反応しない")
	})
}

func TestPlanItemHandlingAction(t *testing.T) {
	t.Parallel()

	t.Run("PolicyDistributeでリーダーと隣接しているとき転送アクションを返す", func(t *testing.T) {
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

		memberGrid := world.Components.GridElement.Get(member)
		leaderGrid := world.Components.GridElement.Get(leader)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{ItemHandling: gc.PolicyDistribute, ViewDistance: 5},
			LeaderEntity: leader,
			LeaderGrid:   leaderGrid,
		}

		b, ok := sp.planItemHandlingAction(world, member, ctx)
		assert.True(t, ok, "転送アクションが返る")
		assert.NotNil(t, b)
		transfer, ok := b.(*activity.TransferActivity)
		require.True(t, ok, "型が *activity.TransferActivity であるべき")
		assert.NotZero(t, transfer.Target)
	})

	t.Run("PolicyKeepではアイテムがあっても転送しない", func(t *testing.T) {
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

		memberGrid := world.Components.GridElement.Get(member)
		leaderGrid := world.Components.GridElement.Get(leader)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{ItemHandling: gc.PolicyKeep, ViewDistance: 5},
			LeaderEntity: leader,
			LeaderGrid:   leaderGrid,
		}

		_, ok := sp.planItemHandlingAction(world, member, ctx)
		assert.False(t, ok, "PolicyKeepでは転送しない")
	})

	t.Run("PolicyDistributeでもリーダーと離れていれば転送しない", func(t *testing.T) {
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

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 20
		memberGrid.Y = 20

		leaderGrid := world.Components.GridElement.Get(leader)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{ItemHandling: gc.PolicyDistribute, ViewDistance: 5},
			LeaderEntity: leader,
			LeaderGrid:   leaderGrid,
		}

		_, ok := sp.planItemHandlingAction(world, member, ctx)
		assert.False(t, ok, "離れているときは転送しない")
	})

	t.Run("PolicyDistributeでバックパックが空なら転送しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		leaderGrid := world.Components.GridElement.Get(leader)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{ItemHandling: gc.PolicyDistribute, ViewDistance: 5},
			LeaderEntity: leader,
			LeaderGrid:   leaderGrid,
		}

		_, ok := sp.planItemHandlingAction(world, member, ctx)
		assert.False(t, ok, "バックパックが空なら転送しない")
	})
}
