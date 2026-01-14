package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/lib/components"
	"github.com/kijimaD/ruins/lib/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestInitDebugData(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 初期状態ではプレイヤーは0人
	memberCount := 0
	world.Manager.Join(
		world.Components.FactionAlly,
		world.Components.Player,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		memberCount++
	}))
	assert.Equal(t, 0, memberCount, "初期状態ではプレイヤーは0人であるべき")

	// デバッグデータ初期化実行
	InitDebugData(world)

	// 初期化後はプレイヤーが1人いるはず
	memberCount = 0
	world.Manager.Join(
		world.Components.FactionAlly,
		world.Components.Player,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		memberCount++
	}))
	assert.Equal(t, 1, memberCount, "デバッグ初期化後はプレイヤーが1人いるべき")

	// 2回目の実行では何も追加されないことを確認
	InitDebugData(world)
	memberCount = 0
	world.Manager.Join(
		world.Components.FactionAlly,
		world.Components.Player,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		memberCount++
	}))
	assert.Equal(t, 1, memberCount, "2回目の実行ではプレイヤー数は変わらないべき")

	// アイテムが生成されていることを確認
	entity, found := FindStackableInInventory(world, "回復薬")
	require.True(t, found, "回復薬のアイテムが生成されているべき")
	item := world.Components.Item.Get(entity).(*gc.Item)
	assert.Greater(t, item.Count, 0, "回復薬の数量が0より大きいべき")
}
