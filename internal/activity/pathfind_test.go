package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
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

		mover := world.Manager.NewEntity()
		mover.AddComponent(world.Components.Player, &gc.Player{})

		nextX, nextY, ok := FindNextStep(world, mover, 0, 0, 3, 0)
		require.True(t, ok)
		assert.Equal(t, 1, nextX)
		assert.Equal(t, 0, nextY)
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

		mover := world.Manager.NewEntity()
		mover.AddComponent(world.Components.Player, &gc.Player{})

		for y := range 4 {
			si.BlockPass[gc.GridElement{X: consts.Tile(2), Y: consts.Tile(y)}] = true
		}

		nextX, nextY, ok := FindNextStep(world, mover, 0, 0, 4, 0)
		require.True(t, ok, "壁を迂回する経路が見つかるべき")
		assert.True(t, nextX >= 0 && nextY >= 0, "有効な座標が返る")
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

		mover := world.Manager.NewEntity()
		mover.AddComponent(world.Components.Player, &gc.Player{})

		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				si.BlockPass[gc.GridElement{X: consts.Tile(5 + dx), Y: consts.Tile(5 + dy)}] = true
			}
		}

		_, _, ok := FindNextStep(world, mover, 5, 5, 8, 8)
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

		mover := world.Manager.NewEntity()
		mover.AddComponent(world.Components.Player, &gc.Player{})

		_, _, ok := FindNextStep(world, mover, 3, 3, 3, 3)
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

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(1), Y: consts.Tile(0)})
		si.Characters[gc.GridElement{X: consts.Tile(1), Y: consts.Tile(0)}] = player
		si.PlayerEntity = &player

		mover := world.Manager.NewEntity()
		mover.AddComponent(world.Components.SquadMember, &gc.SquadMember{})

		nextX, nextY, ok := FindNextStep(world, mover, 0, 0, 2, 0)
		require.True(t, ok, "プレイヤーを迂回する経路が見つかるべき")
		assert.False(t, nextX == 1 && nextY == 0, "プレイヤーのタイルは踏まない")
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

		otherMember := world.Manager.NewEntity()
		otherMember.AddComponent(world.Components.SquadMember, &gc.SquadMember{})
		si.Characters[gc.GridElement{X: consts.Tile(1), Y: consts.Tile(0)}] = otherMember

		mover := world.Manager.NewEntity()
		mover.AddComponent(world.Components.SquadMember, &gc.SquadMember{})

		nextX, nextY, ok := FindNextStep(world, mover, 0, 0, 2, 0)
		require.True(t, ok, "隊員を迂回する経路が見つかるべき")
		assert.False(t, nextX == 1 && nextY == 0, "他の隊員のタイルは踏まない")
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

		mover := world.Manager.NewEntity()
		mover.AddComponent(world.Components.Player, &gc.Player{})

		si.BlockPass[gc.GridElement{X: consts.Tile(5), Y: consts.Tile(5)}] = true

		nextX, nextY, ok := FindNextStep(world, mover, 0, 0, 5, 5)
		require.True(t, ok, "BlockPassなゴールにも経路が見つかるべき")
		assert.True(t, nextX >= 0 && nextY >= 0, "有効な座標が返る")
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

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		si.Characters[gc.GridElement{X: consts.Tile(3), Y: consts.Tile(3)}] = player

		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.SoloAI, &gc.SoloAI{})

		_, _, ok := FindNextStep(world, enemy, 0, 0, 3, 3)
		require.True(t, ok, "敵はプレイヤーの方向を見つけられる")
	})
}
