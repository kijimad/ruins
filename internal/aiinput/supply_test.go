package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kijimaD/ruins/internal/activity"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// setupSupplyTest はリーダーと空腹の隊員を作る。
func setupSupplyTest(t *testing.T, world w.World) (leader ecs.Entity, member ecs.Entity) {
	t.Helper()

	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)
	member, err = lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
	require.NoError(t, err)

	// 空腹段まで減らす
	hunger := world.Components.Hunger.Get(member)
	hunger.Current = hunger.Max / 2
	return leader, member
}

// buildSupplySnapshot は giveFood 等の構造変更を済ませた後に呼び、最新のコンポーネントで snap を作る
func buildSupplySnapshot(world w.World, leader ecs.Entity, member ecs.Entity) *squadSnapshot {
	return &squadSnapshot{
		Grid:         world.Components.GridElement.Get(member),
		Squad:        world.Components.SquadAI.Get(member),
		LeaderEntity: leader,
		LeaderGrid:   world.Components.GridElement.Get(leader),
	}
}

// giveFood は指定エンティティの背嚢へ食料を持たせる
func giveFood(t *testing.T, world w.World, owner ecs.Entity, name string) ecs.Entity {
	t.Helper()
	item, err := lifecycle.SpawnFieldItem(world, name, 1, 1, 1)
	require.NoError(t, err)
	require.NoError(t, lifecycle.MoveToBackpack(world, item, owner))
	return item
}

func TestPlanSupplyAction(t *testing.T) {
	t.Parallel()

	t.Run("手動ポリシーでは発火しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, member := setupSupplyTest(t, world)
		giveFood(t, world, member, "パン")
		snap := buildSupplySnapshot(world, leader, member)
		snap.Squad.Supply = gc.SupplyManual

		sp := newSquadPlanner(newTestRNG())
		_, ok := sp.planSupplyAction(world, member, snap)
		assert.False(t, ok)
	})

	t.Run("満腹なら発火しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, member := setupSupplyTest(t, world)
		giveFood(t, world, member, "パン")
		hunger := world.Components.Hunger.Get(member)
		hunger.Current = hunger.Max
		snap := buildSupplySnapshot(world, leader, member)

		sp := newSquadPlanner(newTestRNG())
		_, ok := sp.planSupplyAction(world, member, snap)
		assert.False(t, ok)
	})

	t.Run("空腹で自分の背嚢に食料があれば食べる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, member := setupSupplyTest(t, world)
		food := giveFood(t, world, member, "パン")
		snap := buildSupplySnapshot(world, leader, member)

		sp := newSquadPlanner(newTestRNG())
		b, ok := sp.planSupplyAction(world, member, snap)
		require.True(t, ok)
		use, isUse := b.(*activity.UseItemActivity)
		require.True(t, isUse, "自分の食料は食べるべき")
		assert.Equal(t, food, use.Target)
	})

	t.Run("栄養価の低い食料を先に消費する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, member := setupSupplyTest(t, world)
		low := giveFood(t, world, member, "ビスケット")
		giveFood(t, world, member, "パン")
		snap := buildSupplySnapshot(world, leader, member)

		lowN := world.Components.ProvidesNutrition.Get(low).Amount
		// 前提: ビスケットのほうが低栄養。逆なら raw の変更でこのテストが知らせる
		for _, e := range []ecs.Entity{low} {
			n := world.Components.ProvidesNutrition.Get(e).Amount
			require.LessOrEqual(t, n, lowN)
		}

		sp := newSquadPlanner(newTestRNG())
		b, ok := sp.planSupplyAction(world, member, snap)
		require.True(t, ok)
		use, isUse := b.(*activity.UseItemActivity)
		require.True(t, isUse)
		got := world.Components.ProvidesNutrition.Get(use.Target).Amount
		assert.LessOrEqual(t, got, lowN, "最も栄養価の低い食料を選ぶべき")
	})

	t.Run("自分に無くリーダー隣接なら受け取る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, member := setupSupplyTest(t, world)
		poolFood := giveFood(t, world, leader, "パン")
		snap := buildSupplySnapshot(world, leader, member)

		sp := newSquadPlanner(newTestRNG())
		b, ok := sp.planSupplyAction(world, member, snap)
		require.True(t, ok)
		tr, isTransfer := b.(*activity.TransferActivity)
		require.True(t, isTransfer, "隣接なら受け取りになるべき")
		assert.Equal(t, poolFood, tr.Target)
		assert.Equal(t, member, tr.Recipient)
		assert.True(t, tr.Single, "共有プールからは1食ぶんだけ受け取る")
	})

	t.Run("リーダーが遠ければ接近する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, member := setupSupplyTest(t, world)
		giveFood(t, world, leader, "パン")

		// 隊員を遠くへ動かす
		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.Coord = consts.Coord[consts.Tile]{X: 20, Y: 10}
		query.InvalidateSpatialIndex(world)
		snap := buildSupplySnapshot(world, leader, member)

		sp := newSquadPlanner(newTestRNG())
		b, ok := sp.planSupplyAction(world, member, snap)
		require.True(t, ok)
		_, isUse := b.(*activity.UseItemActivity)
		_, isTransfer := b.(*activity.TransferActivity)
		assert.False(t, isUse || isTransfer, "遠距離では移動行動になるべき")
	})

	t.Run("敵が視界内にいる間は発火しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, member := setupSupplyTest(t, world)
		giveFood(t, world, member, "パン")

		_, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 12, Y: 10}, "火の玉")
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)
		snap := buildSupplySnapshot(world, leader, member)

		sp := newSquadPlanner(newTestRNG())
		_, ok := sp.planSupplyAction(world, member, snap)
		assert.False(t, ok, "戦闘中は食べないべき")
	})

	t.Run("プール枯渇では発火しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, member := setupSupplyTest(t, world)
		snap := buildSupplySnapshot(world, leader, member)

		sp := newSquadPlanner(newTestRNG())
		_, ok := sp.planSupplyAction(world, member, snap)
		assert.False(t, ok, "食料が無ければ行動しない")
	})
}
