package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestPlayer(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.FactionAlly, &gc.FactionAlly)
	player.AddComponent(world.Components.Name, &gc.Name{Name: "プレイヤー"})

	// 敵を作成（除外されることを確認）
	enemy := world.Manager.NewEntity()
	enemy.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
	enemy.AddComponent(world.Components.Name, &gc.Name{Name: "敵"})

	// クエリを実行
	var foundEntities []ecs.Entity
	Player(world, func(entity ecs.Entity) {
		foundEntities = append(foundEntities, entity)
	})

	// 結果を検証
	assert.Len(t, foundEntities, 1, "プレイヤーが1つだけ見つかるべき")
	assert.Equal(t, player, foundEntities[0], "正しいプレイヤーが見つかるべき")

	// クリーンアップ
	world.Manager.DeleteEntity(player)
	world.Manager.DeleteEntity(enemy)
}

func TestGetPlayerEntity(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーが1個存在する場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		entity, err := GetPlayerEntity(world)
		require.NoError(t, err)
		assert.Equal(t, player, entity)

		world.Manager.DeleteEntity(player)
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

		player1 := world.Manager.NewEntity()
		player1.AddComponent(world.Components.Player, &gc.Player{})

		player2 := world.Manager.NewEntity()
		player2.AddComponent(world.Components.Player, &gc.Player{})

		_, err := GetPlayerEntity(world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "プレイヤーエンティティが複数存在します")

		world.Manager.DeleteEntity(player1)
		world.Manager.DeleteEntity(player2)
	})
}

func TestIsPickable(t *testing.T) {
	t.Parallel()

	t.Run("LocationOnFieldを持つエンティティは拾える", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.LocationOnField, &gc.LocationOnField{})

		assert.True(t, IsPickable(entity, world))
	})

	t.Run("LocationOnFieldがないエンティティは拾えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()

		assert.False(t, IsPickable(entity, world))
	})

	t.Run("Propは拾えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.LocationOnField, &gc.LocationOnField{})
		entity.AddComponent(world.Components.Prop, nil)

		assert.False(t, IsPickable(entity, world), "Propは設置物なので拾えない")
	})
}

func TestGetEntitiesAt(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	e1 := world.Manager.NewEntity()
	e1.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
	e2 := world.Manager.NewEntity()
	e2.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
	e3 := world.Manager.NewEntity()
	e3.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	entities := GetEntitiesAt(world, consts.Tile(5), consts.Tile(5))
	assert.Len(t, entities, 2)

	empty := GetEntitiesAt(world, consts.Tile(99), consts.Tile(99))
	assert.Empty(t, empty)
}
