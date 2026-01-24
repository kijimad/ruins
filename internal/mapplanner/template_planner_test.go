package mapplanner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatePlanner_PlanInitial(t *testing.T) {
	t.Parallel()

	// テスト用のテンプレート
	template := &maptemplate.ChunkTemplate{
		Name:     "test",
		Weight:   100,
		Size:     [2]int{5, 3},
		Palettes: []string{"standard"},
		Map: `#####
#...#
#####`,
	}

	// テスト用のパレット
	palette := &maptemplate.Palette{
		ID: "standard",
		Terrain: map[string]string{
			"#": "Wall",
			".": "Floor",
		},
		Props: map[string]string{},
	}

	planner := NewTemplatePlanner(template, palette)

	t.Run("マップサイズが正しく設定される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, palette, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		metaPlan := &chain.PlanData
		assert.Equal(t, 5, int(metaPlan.Level.TileWidth))
		assert.Equal(t, 3, int(metaPlan.Level.TileHeight))
	})

	t.Run("タイル配列が正しく初期化される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, palette, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		require.NoError(t, err)

		err = planner.PlanInitial(&chain.PlanData)
		require.NoError(t, err)

		metaPlan := &chain.PlanData
		assert.Len(t, metaPlan.Tiles, 15) // 5x3=15
	})

	t.Run("地形が正しく配置される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, palette, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		require.NoError(t, err)

		err = planner.PlanInitial(&chain.PlanData)
		require.NoError(t, err)

		metaPlan := &chain.PlanData

		// 0,0は'#'なのでWall（通行不可）
		tile00 := metaPlan.Tiles[0]
		assert.False(t, tile00.Walkable, "Wall should not be walkable")

		// 1,1は'.'なのでFloor（通行可能）
		tile11 := metaPlan.Tiles[1*5+1]
		assert.True(t, tile11.Walkable, "Floor should be walkable")
	})
}

func TestTemplatePlanner_PlanMeta(t *testing.T) {
	t.Parallel()

	// テスト用のテンプレート
	template := &maptemplate.ChunkTemplate{
		Name:     "test",
		Weight:   100,
		Size:     [2]int{5, 3},
		Palettes: []string{"standard"},
		Map: `#####
#T.M#
#####`,
	}

	// テスト用のパレット
	palette := &maptemplate.Palette{
		ID: "standard",
		Terrain: map[string]string{
			"#": "Wall",
			".": "Floor",
			"T": "Floor",
			"M": "Floor",
		},
		Props: map[string]string{
			"T": "table",
			"M": "machine",
		},
	}

	planner := NewTemplatePlanner(template, palette)

	t.Run("Propsが正しく配置予定リストに追加される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, palette, 12345)
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
			if prop.X == 1 && prop.Y == 1 && prop.PropKey == "table" {
				foundTable = true
			}
		}
		assert.True(t, foundTable, "table should be placed at (1, 1)")

		// 機械の確認（3, 1の位置）
		foundMachine := false
		for _, prop := range metaPlan.Props {
			if prop.X == 3 && prop.Y == 1 && prop.PropKey == "machine" {
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
		Size:     [2]int{10, 10},
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
			"#": "Wall",
			".": "Floor",
		},
		Props: map[string]string{},
	}

	t.Run("PlannerChainが正常に作成される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, palette, 99999)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		require.NoError(t, err)
		assert.NotNil(t, chain)
		assert.NotNil(t, chain.PlanData)
	})

	t.Run("Planを実行してマップが生成される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewTemplatePlannerChain(template, palette, 99999)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		require.NoError(t, err)

		err = chain.Plan()
		require.NoError(t, err)

		metaPlan := &chain.PlanData
		assert.Len(t, metaPlan.Tiles, 100) // 10x10=100

		// 中央は床で通行可能
		centerIdx := 5*10 + 5
		assert.True(t, metaPlan.Tiles[centerIdx].Walkable)

		// 外周は壁で通行不可
		assert.False(t, metaPlan.Tiles[0].Walkable)
	})

	t.Run("パレットに定義がない文字はエラー", func(t *testing.T) {
		t.Parallel()
		invalidTemplate := &maptemplate.ChunkTemplate{
			Name:     "test",
			Weight:   100,
			Size:     [2]int{3, 3},
			Palettes: []string{"standard"},
			Map: `###
#X#
###`,
		}

		invalidPalette := &maptemplate.Palette{
			ID: "standard",
			Terrain: map[string]string{
				"#": "Wall",
				// "X"の定義がない
			},
			Props: map[string]string{},
		}

		chain, err := NewTemplatePlannerChain(invalidTemplate, invalidPalette, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		planner := NewTemplatePlanner(invalidTemplate, invalidPalette)
		err = planner.PlanInitial(&chain.PlanData)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "パレットに文字 'X' の地形定義がありません")
	})
}
