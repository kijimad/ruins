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

		require.NoError(t, err)
		assert.NotNil(t, fonts)
		assert.NotEmpty(t, fonts)
	})
}

func TestLoadSpriteSheets(t *testing.T) {
	t.Parallel()
	t.Run("正常にスプライトシートを読み込める", func(t *testing.T) {
		t.Parallel()
		rw, err := LoadRaws()
		require.NoError(t, err)

		sprites, err := LoadSpriteSheets(rw)

		require.NoError(t, err)
		assert.NotNil(t, sprites)
		assert.NotEmpty(t, sprites)

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
		for i := range 16 {
			key := fmt.Sprintf("dirt_%d", i)
			_, exists := tileSheet.Sprites[key]
			assert.True(t, exists, "%s が存在すること", key)
		}

		// wall_0 から wall_15 まで存在することを確認
		for i := range 16 {
			key := fmt.Sprintf("wall_%d", i)
			_, exists := tileSheet.Sprites[key]
			assert.True(t, exists, "%s が存在すること", key)
		}

		// floor_0 から floor_15 まで存在することを確認
		for i := range 16 {
			key := fmt.Sprintf("floor_%d", i)
			_, exists := tileSheet.Sprites[key]
			assert.True(t, exists, "%s が存在すること", key)
		}

		// voidが存在することを確認
		_, exists := tileSheet.Sprites["void"]
		assert.True(t, exists, "void が存在すること")

		// 合計65個のスプライトがあることを確認（dirt: 16, floor: 16, wall: 16, dwall: 16, void: 1）
		assert.Len(t, tileSheet.Sprites, 65, "65個のタイルスプライトが存在すること")
	})
}

func TestLoadRaws(t *testing.T) {
	t.Parallel()
	t.Run("正常にRawデータを読み込める", func(t *testing.T) {
		t.Parallel()
		rawMaster, err := LoadRaws()

		require.NoError(t, err)
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
		assert.NotEmpty(t, sheet.Sprites)

		// 各スプライトのサイズが正しいことを確認
		for key, sprite := range sheet.Sprites {
			assert.Positive(t, sprite.Width, "スプライト %s の幅が正の値であること", key)
			assert.Positive(t, sprite.Height, "スプライト %s の高さが正の値であること", key)
		}
	})

	t.Run("tilesスプライトシートを正常に読み込める", func(t *testing.T) {
		t.Parallel()
		sheet, err := LoadSpriteSheetFromAseprite("file/textures/dist/tiles.json")

		require.NoError(t, err)
		assert.NotNil(t, sheet)

		// 65個のタイルスプライトが存在することを確認（dirt: 16, floor: 16, wall: 16, dwall: 16, void: 1）
		assert.Len(t, sheet.Sprites, 65, "65個のタイルスプライトが存在すること")
	})

	t.Run("singleスプライトシートを正常に読み込める", func(t *testing.T) {
		t.Parallel()
		sheet, err := LoadSpriteSheetFromAseprite("file/textures/dist/single.json")

		require.NoError(t, err)
		assert.NotNil(t, sheet)
		assert.NotEmpty(t, sheet.Sprites)
	})

	t.Run("存在しないファイルを読み込むとエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadSpriteSheetFromAseprite("file/textures/dist/nonexistent.json")

		require.Error(t, err)
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

			assert.NotEmpty(t, sheet.Sprites, "ファイル %s にスプライトが存在すること", file)
		}
	})
}
