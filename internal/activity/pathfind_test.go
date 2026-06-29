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
		si.SquadMembers = make(map[gc.GridElement]ecs.Entity)

		nextX, nextY, ok := FindNextStep(world, 0, 0, 3, 0)
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
		si.SquadMembers = make(map[gc.GridElement]ecs.Entity)

		// 縦に壁を配置する。(2,0)〜(2,3) を壁にして直進を塞ぐ
		for y := 0; y <= 3; y++ {
			si.BlockPass[gc.GridElement{X: consts.Tile(2), Y: consts.Tile(y)}] = true
		}

		// (0,0) → (4,0) は壁で直進できない
		nextX, nextY, ok := FindNextStep(world, 0, 0, 4, 0)
		require.True(t, ok, "壁を迂回する経路が見つかるべき")

		// 最初の1歩は迂回方向に進むはず
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
		si.SquadMembers = make(map[gc.GridElement]ecs.Entity)

		// (5,5) の周囲8マスを壁で囲む
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				si.BlockPass[gc.GridElement{X: consts.Tile(5 + dx), Y: consts.Tile(5 + dy)}] = true
			}
		}

		_, _, ok := FindNextStep(world, 5, 5, 8, 8)
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
		si.SquadMembers = make(map[gc.GridElement]ecs.Entity)

		_, _, ok := FindNextStep(world, 3, 3, 3, 3)
		assert.False(t, ok)
	})

	t.Run("プレイヤーを壁として迂回路を見つける", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.SquadMembers = make(map[gc.GridElement]ecs.Entity)

		// プレイヤーを(1,0)に配置する
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(1), Y: consts.Tile(0)})
		si.BlockPass[gc.GridElement{X: consts.Tile(1), Y: consts.Tile(0)}] = true
		si.PlayerEntity = &player

		// (0,0) → (2,0) はプレイヤーが直線上にいるが、斜め迂回で到達できる
		nextX, nextY, ok := FindNextStep(world, 0, 0, 2, 0)
		require.True(t, ok, "プレイヤーを迂回する経路が見つかるべき")
		assert.False(t, nextX == 1 && nextY == 0, "プレイヤーのタイルは踏まない")
	})

	t.Run("隊員のいるタイルを通過できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.SquadMembers = make(map[gc.GridElement]ecs.Entity)

		// (1,0)に隊員がいてBlockPassも設定されている
		memberKey := gc.GridElement{X: consts.Tile(1), Y: consts.Tile(0)}
		si.BlockPass[memberKey] = true
		memberEntity := world.Manager.NewEntity()
		si.SquadMembers[memberKey] = memberEntity

		// (0,0) → (2,0) を移動する場合、(1,0)の隊員のタイルを通れる
		nextX, nextY, ok := FindNextStep(world, 0, 0, 2, 0)
		require.True(t, ok, "隊員のいるタイルを通過して経路が見つかるべき")
		assert.Equal(t, 1, nextX)
		assert.Equal(t, 0, nextY)
	})

	t.Run("BlockPassなゴールにも経路を見つける", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		si := query.GetSpatialIndex(world)
		si.Built = true
		si.MapWidth = 10
		si.MapHeight = 10
		si.BlockPass = make(map[gc.GridElement]bool)
		si.SquadMembers = make(map[gc.GridElement]ecs.Entity)

		// (5,5)をBlockPassにする
		si.BlockPass[gc.GridElement{X: consts.Tile(5), Y: consts.Tile(5)}] = true

		// ゴールがBlockPassでも経路が見つかる
		nextX, nextY, ok := FindNextStep(world, 0, 0, 5, 5)
		require.True(t, ok, "BlockPassなゴールにも経路が見つかるべき")
		assert.True(t, nextX >= 0 && nextY >= 0, "有効な座標が返る")
	})
}
