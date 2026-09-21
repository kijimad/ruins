package testutil

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitTestWorld(t *testing.T) {
	t.Parallel()

	t.Run("オプション未指定ならオーバーワールドの50x50で初期化する", func(t *testing.T) {
		t.Parallel()
		world := InitTestWorld(t)

		assert.Equal(t, gc.NewOverworldStage(), query.GetDungeon(world).CurrentStage)

		field := query.GetCurrentStageField(world)
		require.NotNil(t, field)
		assert.Equal(t, gc.Level{TileWidth: 50, TileHeight: 50}, field.Level)
	})

	t.Run("WithCurrentStageで指定したステージキーになる", func(t *testing.T) {
		t.Parallel()
		key := gc.NewDungeonStage("test-dungeon", 3)
		world := InitTestWorld(t, WithCurrentStage(key))

		assert.Equal(t, key, query.GetDungeon(world).CurrentStage)
	})

	t.Run("WithStageLevelで指定した寸法になる", func(t *testing.T) {
		t.Parallel()
		level := gc.Level{TileWidth: 12, TileHeight: 7}
		world := InitTestWorld(t, WithStageLevel(level))

		field := query.GetCurrentStageField(world)
		require.NotNil(t, field)
		assert.Equal(t, level, field.Level)
	})

	t.Run("WithUI未指定ならUIResourcesは空のまま", func(t *testing.T) {
		t.Parallel()
		world := InitTestWorld(t)

		assert.Nil(t, world.Resources.UIResources.Fonts)
		assert.Nil(t, world.Resources.UIResources.Text)
	})

	t.Run("WithUIでフォントとテキストリソースを読み込む", func(t *testing.T) {
		t.Parallel()
		world := InitTestWorld(t, WithUI())

		require.NotNil(t, world.Resources.UIResources.Fonts)
		require.NotNil(t, world.Resources.UIResources.Text)
		assert.NotNil(t, world.Resources.UIResources.Text.BodyFace)
	})
}
