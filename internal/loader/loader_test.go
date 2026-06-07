package loader

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFonts(t *testing.T) {
	t.Parallel()
	t.Run("正常にフォントを読み込める", func(t *testing.T) {
		t.Parallel()
		fonts, err := LoadFonts()

		assert.NoError(t, err)
		assert.NotNil(t, fonts)
		assert.Greater(t, len(fonts), 0)
	})
}

func TestLoadSpriteSheets(t *testing.T) {
	t.Parallel()
	t.Run("正常にスプライトシートを読み込める", func(t *testing.T) {
		t.Parallel()
		rw, err := LoadRaws()
		require.NoError(t, err)

		sprites, err := LoadSpriteSheets(rw)

		assert.NoError(t, err)
		assert.NotNil(t, sprites)
		assert.Greater(t, len(sprites), 0)

		// 各スプライトシートに名前が設定されていることを確認
		for name, sprite := range sprites {
			assert.Equal(t, name, sprite.Name)
		}
	})

	t.Run("tileスプライトシートに全てのタイルが含まれる", func(t *testing.T) {
		t.Parallel()
		rw, err := LoadRaws()
		require.NoError(t, err)

		sprites, err := LoadSpriteSheets(rw)
		require.NoError(t, err)

		tileSheet, ok := sprites["tile"]
		require.True(t, ok, "tileスプライトシートが存在すること")

		// dirt_0 から dirt_15 まで存在することを確認
		for i := 0; i < 16; i++ {
			key := fmt.Sprintf("dirt_%d", i)
			_, exists := tileSheet.Sprites[key]
			assert.True(t, exists, "%s が存在すること", key)
		}

		// wall_0 から wall_15 まで存在することを確認
		for i := 0; i < 16; i++ {
			key := fmt.Sprintf("wall_%d", i)
			_, exists := tileSheet.Sprites[key]
			assert.True(t, exists, "%s が存在すること", key)
		}

		// floor_0 から floor_15 まで存在することを確認
		for i := 0; i < 16; i++ {
			key := fmt.Sprintf("floor_%d", i)
			_, exists := tileSheet.Sprites[key]
			assert.True(t, exists, "%s が存在すること", key)
		}

		// voidが存在することを確認
		_, exists := tileSheet.Sprites["void"]
		assert.True(t, exists, "void が存在すること")

		// 合計65個のスプライトがあることを確認（dirt: 16, floor: 16, wall: 16, dwall: 16, void: 1）
		assert.Equal(t, 65, len(tileSheet.Sprites), "65個のタイルスプライトが存在すること")
	})
}

func TestLoadRaws(t *testing.T) {
	t.Parallel()
	t.Run("正常にRawデータを読み込める", func(t *testing.T) {
		t.Parallel()
		rawMaster, err := LoadRaws()

		assert.NoError(t, err)
		assert.NotEmpty(t, rawMaster.Items)
	})
}

func TestLoadSpriteSheetFromAseprite(t *testing.T) {
	t.Parallel()

	t.Run("bgスプライトシートを正常に読み込める", func(t *testing.T) {
		t.Parallel()
		sheet, err := LoadSpriteSheetFromAseprite("file/textures/dist/bg.json")

		require.NoError(t, err)
		assert.NotNil(t, sheet)
		assert.NotNil(t, sheet.Texture)
		assert.Greater(t, len(sheet.Sprites), 0)

		// 各スプライトのサイズが正しいことを確認
		for key, sprite := range sheet.Sprites {
			assert.Greater(t, sprite.Width, 0, "スプライト %s の幅が正の値であること", key)
			assert.Greater(t, sprite.Height, 0, "スプライト %s の高さが正の値であること", key)
		}
	})

	t.Run("tilesスプライトシートを正常に読み込める", func(t *testing.T) {
		t.Parallel()
		sheet, err := LoadSpriteSheetFromAseprite("file/textures/dist/tiles.json")

		require.NoError(t, err)
		assert.NotNil(t, sheet)

		// 65個のタイルスプライトが存在することを確認（dirt: 16, floor: 16, wall: 16, dwall: 16, void: 1）
		assert.Equal(t, 65, len(sheet.Sprites), "65個のタイルスプライトが存在すること")
	})

	t.Run("singleスプライトシートを正常に読み込める", func(t *testing.T) {
		t.Parallel()
		sheet, err := LoadSpriteSheetFromAseprite("file/textures/dist/single.json")

		require.NoError(t, err)
		assert.NotNil(t, sheet)
		assert.Greater(t, len(sheet.Sprites), 0)
	})

	t.Run("存在しないファイルを読み込むとエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadSpriteSheetFromAseprite("file/textures/dist/nonexistent.json")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JSONファイルの読み込みに失敗")
	})

	t.Run("不正なパスを指定するとエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadSpriteSheetFromAseprite("")

		assert.Error(t, err)
	})

	t.Run("全てのスプライトファイル名が'_'で終わっていることを確認", func(t *testing.T) {
		t.Parallel()
		testFiles := []string{
			"file/textures/dist/bg.json",
			"file/textures/dist/tiles.json",
			"file/textures/dist/single.json",
		}

		for _, file := range testFiles {
			sheet, err := LoadSpriteSheetFromAseprite(file)
			require.NoError(t, err, "ファイル %s の読み込みに失敗", file)

			assert.Greater(t, len(sheet.Sprites), 0, "ファイル %s にスプライトが存在すること", file)
		}
	})
}
