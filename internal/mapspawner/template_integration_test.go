package mapspawner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateToMapIntegration(t *testing.T) {
	t.Parallel()

	t.Run("実際のTOMLファイルからマップ生成してスポーンまで実行できる", func(t *testing.T) {
		t.Parallel()

		// 1. パレットを読み込む
		paletteLoader := maptemplate.NewPaletteLoader()
		palette, err := paletteLoader.LoadFromFile("../../assets/levels/palettes/standard.toml")
		require.NoError(t, err)

		// 2. 施設テンプレートを読み込む
		templateLoader := maptemplate.NewTemplateLoader()
		templates, err := templateLoader.LoadFromFile("../../assets/levels/facilities/small_room.toml")
		require.NoError(t, err)
		require.Len(t, templates, 1)

		template := &templates[0]

		// 3. 実際のRawMasterを読み込んだWorldを作成
		world := testutil.InitTestWorld(t)

		// 4. テンプレートからマップを生成
		chain, err := mapplanner.NewTemplatePlannerChain(template, palette, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = world.Resources.RawMaster.(*raw.Master)

		err = chain.Plan()
		require.NoError(t, err)

		// 5. MetaPlanの検証
		metaPlan := &chain.PlanData
		assert.Equal(t, 10, int(metaPlan.Level.TileWidth))
		assert.Equal(t, 10, int(metaPlan.Level.TileHeight))
		assert.Len(t, metaPlan.Tiles, 100) // 10x10=100

		// 外周は壁で通行不可
		assert.False(t, metaPlan.Tiles[0].Walkable, "外周は壁")

		// 中央は床で通行可能
		centerIdx := 5*10 + 5
		assert.True(t, metaPlan.Tiles[centerIdx].Walkable, "中央は床")

		// 家具（Props）が配置されている
		assert.NotEmpty(t, metaPlan.Props, "家具が配置されているはず")

		// 6. mapspawnerでスポーンを実行
		level, err := Spawn(world, metaPlan)
		require.NoError(t, err)

		assert.Equal(t, 10, int(level.TileWidth))
		assert.Equal(t, 10, int(level.TileHeight))
	})
}
