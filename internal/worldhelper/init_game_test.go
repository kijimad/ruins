package worldhelper

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestInitNewGameData(t *testing.T) {
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
	InitNewGameData(world)

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
	InitNewGameData(world)
	memberCount = 0
	world.Manager.Join(
		world.Components.FactionAlly,
		world.Components.Player,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		memberCount++
	}))
	assert.Equal(t, 1, memberCount, "2回目の実行ではプレイヤー数は変わらないべき")
}
