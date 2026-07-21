package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGameProgress(t *testing.T) {
	t.Parallel()

	t.Run("InitWorldで生成されたGameProgressを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		gp := GetGameProgress(world)
		require.NotNil(t, gp)
	})

	t.Run("複数回取得しても同じポインタを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		gp1 := GetGameProgress(world)
		gp2 := GetGameProgress(world)
		assert.Same(t, gp1, gp2)
	})
}

func TestGetDungeon(t *testing.T) {
	t.Parallel()

	t.Run("InitWorldで生成されたDungeonを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		d := GetDungeon(world)
		require.NotNil(t, d)
	})

	t.Run("SetDungeonで設定した値を取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		newDungeon := &gc.Dungeon{Depth: 3}
		SetDungeon(world, newDungeon)

		d := GetDungeon(world)
		require.NotNil(t, d)
		assert.Equal(t, 3, d.Depth)
	})

	t.Run("SetDungeonでnilを設定するとGetDungeonはnilを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		SetDungeon(world, nil)

		d := GetDungeon(world)
		assert.Nil(t, d)
	})
}

// TestIsOnOverworld は現在地判定を検証する。共存方式では遺跡滞在中も SeamlessBand は Active の
// まま残るため、Active を場所判定の代理にしてはいけない。現ステージで判定することを固定する。
func TestIsOnOverworld(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := GetDungeon(world)

	d.CurrentStage = gc.NewOverworldStage()
	assert.True(t, IsOnOverworld(world), "現ステージがオーバーワールドなら真")

	// 遺跡滞在中。帯・前線が Active のまま残っても、オーバーワールド判定にしてはいけない。
	d.CurrentStage = gc.NewNamedDungeonStage("テスト遺跡", 1)
	d.SeamlessBand.Active = true
	d.SeamlessBand.Front.Active = true
	assert.False(t, IsOnOverworld(world), "遺跡滞在中は SeamlessBand/Front が Active でも偽")
}
