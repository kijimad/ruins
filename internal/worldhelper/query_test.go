package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestQueryOwnedStackable(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用スタック可能アイテムエンティティを作成
	stackableEntity := world.Manager.NewEntity()
	stackableEntity.AddComponent(world.Components.Item, &gc.Item{Count: 5})
	stackableEntity.AddComponent(world.Components.Stackable, &gc.Stackable{})
	stackableEntity.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.ItemLocationInPlayerBackpack)
	stackableEntity.AddComponent(world.Components.Name, &gc.Name{Name: "テストスタック可能アイテム"})

	// スタック不可アイテムを作成（除外されることを確認）
	nonStackableEntity := world.Manager.NewEntity()
	nonStackableEntity.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.ItemLocationInPlayerBackpack)
	nonStackableEntity.AddComponent(world.Components.Name, &gc.Name{Name: "テストアイテム"})

	// クエリを実行
	var foundEntities []ecs.Entity
	QueryOwnedStackable(world, func(entity ecs.Entity) {
		foundEntities = append(foundEntities, entity)
	})

	// 結果を検証
	assert.Len(t, foundEntities, 1, "スタック可能アイテムが1つだけ見つかるべき")
	assert.Equal(t, stackableEntity, foundEntities[0], "正しいスタック可能アイテムが見つかるべき")

	// クリーンアップ
	world.Manager.DeleteEntity(stackableEntity)
	world.Manager.DeleteEntity(nonStackableEntity)
}

func TestQueryPlayer(t *testing.T) {
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
	QueryPlayer(world, func(entity ecs.Entity) {
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
