package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindNextStep(t *testing.T) {
	t.Parallel()

	t.Run("障害物がなければ目標に直進する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.Characters = make(map[gc.GridElement]ecs.Entity)

		mover := world.ECS.NewEntity()
		world.Components.Player.Add(mover, &gc.Player{})

		next, ok := FindNextStep(world, mover, consts.Coord[consts.Tile]{X: 0, Y: 0}, consts.Coord[consts.Tile]{X: 3, Y: 0})
		require.True(t, ok)
		assert.Equal(t, 1, int(next.X))
		assert.Equal(t, 0, int(next.Y))
	})

	t.Run("壁を迂回して経路を見つける", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.Characters = make(map[gc.GridElement]ecs.Entity)

		mover := world.ECS.NewEntity()
		world.Components.Player.Add(mover, &gc.Player{})

		for y := range 4 {
			si.BlockPass[gc.GridElement{X: consts.Tile(2), Y: consts.Tile(y)}] = true
		}

		next, ok := FindNextStep(world, mover, consts.Coord[consts.Tile]{X: 0, Y: 0}, consts.Coord[consts.Tile]{X: 4, Y: 0})
		require.True(t, ok, "壁を迂回する経路が見つかるべき")
		assert.True(t, next.X >= 0 && next.Y >= 0, "有効な座標が返る")
	})

	t.Run("完全に囲まれていれば経路がない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.Characters = make(map[gc.GridElement]ecs.Entity)

		mover := world.ECS.NewEntity()
		world.Components.Player.Add(mover, &gc.Player{})

		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				si.BlockPass[gc.GridElement{X: consts.Tile(5 + dx), Y: consts.Tile(5 + dy)}] = true
			}
		}

		_, ok := FindNextStep(world, mover, consts.Coord[consts.Tile]{X: 5, Y: 5}, consts.Coord[consts.Tile]{X: 8, Y: 8})
		assert.False(t, ok, "囲まれている場合は経路がない")
	})

	t.Run("同じ位置の場合はfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.Characters = make(map[gc.GridElement]ecs.Entity)

		mover := world.ECS.NewEntity()
		world.Components.Player.Add(mover, &gc.Player{})

		_, ok := FindNextStep(world, mover, consts.Coord[consts.Tile]{X: 3, Y: 3}, consts.Coord[consts.Tile]{X: 3, Y: 3})
		assert.False(t, ok)
	})

	t.Run("隊員はプレイヤーを迂回する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.Characters = make(map[gc.GridElement]ecs.Entity)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{X: consts.Tile(1), Y: consts.Tile(0)})
		si.Characters[gc.GridElement{X: consts.Tile(1), Y: consts.Tile(0)}] = player
		si.PlayerEntity = &player

		mover := world.ECS.NewEntity()
		world.Components.SquadMember.Add(mover, &gc.SquadMember{})

		next, ok := FindNextStep(world, mover, consts.Coord[consts.Tile]{X: 0, Y: 0}, consts.Coord[consts.Tile]{X: 2, Y: 0})
		require.True(t, ok, "プレイヤーを迂回する経路が見つかるべき")
		assert.False(t, next.X == 1 && next.Y == 0, "プレイヤーのタイルは踏まない")
	})

	t.Run("隊員同士は迂回する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.Characters = make(map[gc.GridElement]ecs.Entity)

		otherMember := world.ECS.NewEntity()
		world.Components.SquadMember.Add(otherMember, &gc.SquadMember{})
		si.Characters[gc.GridElement{X: consts.Tile(1), Y: consts.Tile(0)}] = otherMember

		mover := world.ECS.NewEntity()
		world.Components.SquadMember.Add(mover, &gc.SquadMember{})

		next, ok := FindNextStep(world, mover, consts.Coord[consts.Tile]{X: 0, Y: 0}, consts.Coord[consts.Tile]{X: 2, Y: 0})
		require.True(t, ok, "隊員を迂回する経路が見つかるべき")
		assert.False(t, next.X == 1 && next.Y == 0, "他の隊員のタイルは踏まない")
	})

	t.Run("BlockPassなゴールにも経路を見つける", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.Characters = make(map[gc.GridElement]ecs.Entity)

		mover := world.ECS.NewEntity()
		world.Components.Player.Add(mover, &gc.Player{})

		si.BlockPass[gc.GridElement{X: consts.Tile(5), Y: consts.Tile(5)}] = true

		next, ok := FindNextStep(world, mover, consts.Coord[consts.Tile]{X: 0, Y: 0}, consts.Coord[consts.Tile]{X: 5, Y: 5})
		require.True(t, ok, "BlockPassなゴールにも経路が見つかるべき")
		assert.True(t, next.X >= 0 && next.Y >= 0, "有効な座標が返る")
	})

	t.Run("敵はプレイヤーへの方向を見つける", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.Characters = make(map[gc.GridElement]ecs.Entity)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		si.Characters[gc.GridElement{X: consts.Tile(3), Y: consts.Tile(3)}] = player

		enemy := world.ECS.NewEntity()
		world.Components.SoloAI.Add(enemy, &gc.SoloAI{})

		_, ok := FindNextStep(world, enemy, consts.Coord[consts.Tile]{X: 0, Y: 0}, consts.Coord[consts.Tile]{X: 3, Y: 3})
		require.True(t, ok, "敵はプレイヤーの方向を見つけられる")
	})
}
