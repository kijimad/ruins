package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayer(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.FactionAlly.Add(player, &gc.FactionAlly{})
	world.Components.Name.Add(player, &gc.Name{Name: "プレイヤー"})

	// 敵を作成（除外されることを確認）
	enemy := world.ECS.NewEntity()
	world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemy{})
	world.Components.Name.Add(enemy, &gc.Name{Name: "敵"})

	// クエリを実行
	var foundEntities []ecs.Entity
	Player(world, func(entity ecs.Entity) {
		foundEntities = append(foundEntities, entity)
	})

	// 結果を検証
	assert.Len(t, foundEntities, 1, "プレイヤーが1つだけ見つかるべき")
	assert.Equal(t, player, foundEntities[0], "正しいプレイヤーが見つかるべき")

	// クリーンアップ
	world.ECS.RemoveEntity(player)
	world.ECS.RemoveEntity(enemy)
}

func TestGetPlayerEntity(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーが1個存在する場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		entity, err := GetPlayerEntity(world)
		require.NoError(t, err)
		assert.Equal(t, player, entity)

		world.ECS.RemoveEntity(player)
	})

	t.Run("プレイヤーが0個の場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := GetPlayerEntity(world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "プレイヤーエンティティが存在しません")
	})

	t.Run("プレイヤーが2個以上の場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player1 := world.ECS.NewEntity()
		world.Components.Player.Add(player1, &gc.Player{})

		player2 := world.ECS.NewEntity()
		world.Components.Player.Add(player2, &gc.Player{})

		_, err := GetPlayerEntity(world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "プレイヤーエンティティが複数存在します")

		world.ECS.RemoveEntity(player1)
		world.ECS.RemoveEntity(player2)
	})
}

func TestIsPickable(t *testing.T) {
	t.Parallel()

	t.Run("LocationOnFieldを持つエンティティは拾える", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.LocationOnField.Add(entity, &gc.LocationOnField{})

		assert.True(t, IsPickable(entity, world))
	})

	t.Run("LocationOnFieldがないエンティティは拾えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()

		assert.False(t, IsPickable(entity, world))
	})

	t.Run("Propは拾えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.LocationOnField.Add(entity, &gc.LocationOnField{})
		world.Components.Prop.Add(entity, &gc.Prop{})

		assert.False(t, IsPickable(entity, world), "Propは設置物なので拾えない")
	})
}

func TestGetEntitiesAt(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	e1 := world.ECS.NewEntity()
	world.Components.GridElement.Add(e1, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	e2 := world.ECS.NewEntity()
	world.Components.GridElement.Add(e2, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	e3 := world.ECS.NewEntity()
	world.Components.GridElement.Add(e3, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

	entities := GetEntitiesAt(world, consts.Tile(5), consts.Tile(5))
	assert.Len(t, entities, 2)

	empty := GetEntitiesAt(world, consts.Tile(99), consts.Tile(99))
	assert.Empty(t, empty)
}
