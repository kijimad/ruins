package mapplanner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatePlanner_PlanInitial(t *testing.T) {
	t.Parallel()

	template := &maptemplate.ChunkTemplate{
		Name:     "test",
		Weight:   100,
		Size:     maptemplate.Size{W: 5, H: 3},
		Palettes: []string{"standard"},
		Map: `#####
#...#
#####`,
	}

	palette := &maptemplate.Palette{
		ID: "standard",
		Terrain: map[string]string{
			"#": "wall",
			".": "floor",
		},
		Props: map[string]maptemplate.PaletteEntry{},
	}

	resolvedMap := maptemplate.ResolveMapCells(template.Map, palette)
	planner := NewTemplatePlanner(template, resolvedMap)

	t.Run("マップサイズが正しく設定される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, resolvedMap, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		metaPlan := &chain.PlanData
		assert.Equal(t, 5, int(metaPlan.Level.TileWidth))
		assert.Equal(t, 3, int(metaPlan.Level.TileHeight))
	})

	t.Run("タイル配列が正しく初期化される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, resolvedMap, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		require.NoError(t, err)

		err = planner.PlanInitial(&chain.PlanData)
		require.NoError(t, err)

		metaPlan := &chain.PlanData
		assert.Len(t, metaPlan.Tiles, 15) // 5x3=15
	})

	t.Run("地形が正しく配置される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, resolvedMap, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		require.NoError(t, err)

		err = planner.PlanInitial(&chain.PlanData)
		require.NoError(t, err)

		metaPlan := &chain.PlanData

		// 0,0は'#'なのでWall（通行不可）
		tile00 := metaPlan.Tiles[0]
		assert.True(t, tile00.BlockPass, "Wall should not be walkable")

		// 1,1は'.'なのでFloor（通行可能）
		tile11 := metaPlan.Tiles[1*5+1]
		assert.False(t, tile11.BlockPass, "Floor should be walkable")
	})
}

func TestTemplatePlanner_PlanMeta(t *testing.T) {
	t.Parallel()

	template := &maptemplate.ChunkTemplate{
		Name:     "test",
		Weight:   100,
		Size:     maptemplate.Size{W: 5, H: 3},
		Palettes: []string{"standard"},
		Map: `#####
#T.M#
#####`,
	}

	palette := &maptemplate.Palette{
		ID: "standard",
		Terrain: map[string]string{
			"#": "wall",
			".": "floor",
		},
		Props: map[string]maptemplate.PaletteEntry{
			"T": {ID: "table", Tile: "floor"},
			"M": {ID: "machine", Tile: "floor"},
		},
	}

	resolvedMap := maptemplate.ResolveMapCells(template.Map, palette)
	planner := NewTemplatePlanner(template, resolvedMap)

	t.Run("Propsが正しく配置予定リストに追加される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, resolvedMap, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		require.NoError(t, err)

		err = planner.PlanInitial(&chain.PlanData)
		require.NoError(t, err)

		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		metaPlan := &chain.PlanData

		// テーブルと機械の2つが配置されているはず
		assert.Len(t, metaPlan.Props, 2)

		// テーブルの確認（1, 1の位置）
		foundTable := false
		for _, prop := range metaPlan.Props {
			if prop.X == 1 && prop.Y == 1 && prop.Name == "table" {
				foundTable = true
			}
		}
		assert.True(t, foundTable, "table should be placed at (1, 1)")

		// 機械の確認（3, 1の位置）
		foundMachine := false
		for _, prop := range metaPlan.Props {
			if prop.X == 3 && prop.Y == 1 && prop.Name == "machine" {
				foundMachine = true
			}
		}
		assert.True(t, foundMachine, "machine should be placed at (3, 1)")
	})
}

func TestNewTemplatePlannerChain(t *testing.T) {
	t.Parallel()

	template := &maptemplate.ChunkTemplate{
		Name:     "test",
		Weight:   100,
		Size:     maptemplate.Size{W: 10, H: 10},
		Palettes: []string{"standard"},
		Map: `##########
#........#
#........#
#........#
#........#
#........#
#........#
#........#
#........#
##########`,
	}

	palette := &maptemplate.Palette{
		ID: "standard",
		Terrain: map[string]string{
			"#": "wall",
			".": "floor",
		},
		Props: map[string]maptemplate.PaletteEntry{},
	}

	resolvedMap := maptemplate.ResolveMapCells(template.Map, palette)

	t.Run("PlannerChainが正常に作成される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, resolvedMap, 99999)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		require.NoError(t, err)
		assert.NotNil(t, chain)
		assert.NotNil(t, chain.PlanData)
	})

	t.Run("Planを実行してマップが生成される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, resolvedMap, 99999)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		require.NoError(t, err)

		err = chain.Plan()
		require.NoError(t, err)

		metaPlan := &chain.PlanData
		assert.Len(t, metaPlan.Tiles, 100) // 10x10=100

		// 中央は床で通行可能
		centerIdx := 5*10 + 5
		assert.False(t, metaPlan.Tiles[centerIdx].BlockPass)

		// 外周は壁で通行不可
		assert.True(t, metaPlan.Tiles[0].BlockPass)
	})

	t.Run("セルの地形が未定義の場合はエラー", func(t *testing.T) {
		t.Parallel()
		emptyMap := [][]maptemplate.MapCell{
			{{Terrain: "wall"}, {Terrain: "wall"}, {Terrain: "wall"}},
			{{Terrain: "wall"}, {Terrain: ""}, {Terrain: "wall"}},
			{{Terrain: "wall"}, {Terrain: "wall"}, {Terrain: "wall"}},
		}
		emptyTemplate := &maptemplate.ChunkTemplate{
			Name:   "test",
			Weight: 100,
			Size:   maptemplate.Size{W: 3, H: 3},
			Map:    "###\n#.#\n###",
		}

		chain, err := NewTemplatePlannerChain(emptyTemplate, emptyMap, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		planner := NewTemplatePlanner(emptyTemplate, emptyMap)
		err = planner.PlanInitial(&chain.PlanData)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "セルの地形が未定義です")
	})
}
