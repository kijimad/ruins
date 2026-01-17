package systems

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestDeadCleanupSystem(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// テスト用エンティティを作成

	// 1. 通常の敵（AI）エンティティ - 削除されるべき
	enemy := world.Manager.NewEntity()
	enemy.AddComponent(world.Components.Name, &gc.Name{Name: "テスト敵"})
	enemy.AddComponent(world.Components.AIMoveFSM, &gc.AIMoveFSM{})
	enemy.AddComponent(world.Components.Dead, &gc.Dead{})

	// 2. プレイヤーエンティティ - 削除されないべき
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Name, &gc.Name{Name: "プレイヤー"})
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.Dead, &gc.Dead{})

	// 3. その他のDeadエンティティ - 削除されるべき
	otherDead := world.Manager.NewEntity()
	otherDead.AddComponent(world.Components.Name, &gc.Name{Name: "その他"})
	otherDead.AddComponent(world.Components.Dead, &gc.Dead{})

	// 4. 生きているエンティティ - 削除されないべき
	alive := world.Manager.NewEntity()
	alive.AddComponent(world.Components.Name, &gc.Name{Name: "生きている敵"})
	alive.AddComponent(world.Components.AIMoveFSM, &gc.AIMoveFSM{})

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 結果を検証

	// 敵エンティティは削除されているべき（Nameコンポーネントも削除される）
	assert.False(t, enemy.HasComponent(world.Components.Name), "敵エンティティは削除されるべき")

	// プレイヤーエンティティは削除されていないべき
	assert.True(t, player.HasComponent(world.Components.Name), "プレイヤーエンティティは削除されないべき")
	assert.True(t, player.HasComponent(world.Components.Dead), "プレイヤーのDeadコンポーネントは残るべき")

	// その他のDeadエンティティは削除されているべき
	assert.False(t, otherDead.HasComponent(world.Components.Name), "その他のDeadエンティティは削除されるべき")

	// 生きているエンティティは削除されていないべき
	assert.True(t, alive.HasComponent(world.Components.Name), "生きているエンティティは削除されないべき")
	assert.False(t, alive.HasComponent(world.Components.Dead), "生きているエンティティにDeadコンポーネントはないべき")

	// クリーンアップ
	world.Manager.DeleteEntity(player)
	world.Manager.DeleteEntity(alive)
}

func TestDeadCleanupSystem_NoDeadEntities(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// Deadエンティティが存在しない状態でテスト
	alive1 := world.Manager.NewEntity()
	alive1.AddComponent(world.Components.Name, &gc.Name{Name: "生きている1"})

	alive2 := world.Manager.NewEntity()
	alive2.AddComponent(world.Components.Name, &gc.Name{Name: "生きている2"})
	alive2.AddComponent(world.Components.AIMoveFSM, &gc.AIMoveFSM{})

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// すべてのエンティティが残っているべき
	assert.True(t, alive1.HasComponent(world.Components.Name), "生きているエンティティ1は残るべき")
	assert.True(t, alive2.HasComponent(world.Components.Name), "生きているエンティティ2は残るべき")

	// クリーンアップ
	world.Manager.DeleteEntity(alive1)
	world.Manager.DeleteEntity(alive2)
}

func TestDeadCleanupSystem_EmptyWorld(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// エンティティが存在しない状態でテスト
	// パニックやエラーが発生しないことを確認
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// エンティティ数が0であることを確認
	count := 0
	world.Manager.Join().Visit(ecs.Visit(func(_ ecs.Entity) {
		count++
	}))
	assert.Equal(t, 0, count, "空のworldではエンティティ数は0であるべき")
}

func TestDeadCleanupSystem_WithDropTable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// ドロップテーブルを持つ敵エンティティを作成（灰の偶像は100%ドロップ）
	enemy := world.Manager.NewEntity()
	enemy.AddComponent(world.Components.Name, &gc.Name{Name: "灰の偶像"})
	enemy.AddComponent(world.Components.Dead, &gc.Dead{})
	enemy.AddComponent(world.Components.DropTable, &gc.DropTable{Name: "灰の偶像"})
	enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

	// DeadCleanupSystem実行前のアイテムエンティティ数をカウント
	itemCountBefore := 0
	world.Manager.Join(world.Components.Item).Visit(ecs.Visit(func(_ ecs.Entity) {
		itemCountBefore++
	}))

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 敵エンティティは削除されているべき
	assert.False(t, enemy.HasComponent(world.Components.Name), "敵エンティティは削除されるべき")

	// ドロップアイテムが生成されているべき（"鉄くず"がドロップテーブルに定義されている）
	itemCountAfter := 0
	world.Manager.Join(world.Components.Item).Visit(ecs.Visit(func(_ ecs.Entity) {
		itemCountAfter++
	}))

	assert.Greater(t, itemCountAfter, itemCountBefore, "ドロップアイテムが生成されているべき")
	assert.Equal(t, itemCountBefore+1, itemCountAfter, "1つのアイテムがドロップされるべき")
}

func TestDeadCleanupSystem_WithDropTableDrops(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// シード2でドロップするケース
	world.Resources.RNG = rand.New(rand.NewPCG(2, 0))

	// 敵エンティティを作成
	enemy := world.Manager.NewEntity()
	enemy.AddComponent(world.Components.Name, &gc.Name{Name: "火の玉"})
	enemy.AddComponent(world.Components.Dead, &gc.Dead{})
	enemy.AddComponent(world.Components.DropTable, &gc.DropTable{Name: "火の玉"})
	enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

	// 実行前のアイテム数
	itemCountBefore := 0
	world.Manager.Join(world.Components.Item).Visit(ecs.Visit(func(_ ecs.Entity) {
		itemCountBefore++
	}))

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 実行後のアイテム数
	itemCountAfter := 0
	world.Manager.Join(world.Components.Item).Visit(ecs.Visit(func(_ ecs.Entity) {
		itemCountAfter++
	}))

	// シード2ではドロップする
	assert.Equal(t, itemCountBefore+1, itemCountAfter, "シード2ではドロップするはず")
}

func TestDeadCleanupSystem_WithDropTableNoDrops(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// シード1でドロップしないケース
	world.Resources.RNG = rand.New(rand.NewPCG(1, 0))

	// 敵エンティティを作成
	enemy := world.Manager.NewEntity()
	enemy.AddComponent(world.Components.Name, &gc.Name{Name: "火の玉"})
	enemy.AddComponent(world.Components.Dead, &gc.Dead{})
	enemy.AddComponent(world.Components.DropTable, &gc.DropTable{Name: "火の玉"})
	enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

	// 実行前のアイテム数
	itemCountBefore := 0
	world.Manager.Join(world.Components.Item).Visit(ecs.Visit(func(_ ecs.Entity) {
		itemCountBefore++
	}))

	// DeadCleanupSystemを実行
	sys := &DeadCleanupSystem{}
	require.NoError(t, sys.Update(world))

	// 実行後のアイテム数
	itemCountAfter := 0
	world.Manager.Join(world.Components.Item).Visit(ecs.Visit(func(_ ecs.Entity) {
		itemCountAfter++
	}))

	// シード1ではドロップしない
	assert.Equal(t, itemCountBefore, itemCountAfter, "シード1ではドロップしないはず")
}
