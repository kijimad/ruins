package systems

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSpriteImageCache(t *testing.T) {
	t.Parallel()
	t.Run("sprite image cache initialization", func(t *testing.T) {
		t.Parallel()
		sys := NewRenderSpriteSystem()
		assert.NotNil(t, sys.spriteImageCache, "spriteImageCacheがnilになっている")
		assert.Empty(t, sys.spriteImageCache, "新規作成時はキャッシュが空のはず")
	})

	t.Run("sprite image cache is map", func(t *testing.T) {
		t.Parallel()
		// キャッシュがmap型であることを確認
		sys := NewRenderSpriteSystem()
		cache := sys.spriteImageCache
		expectedType := make(map[spriteImageCacheKey]*ebiten.Image)
		assert.IsType(t, expectedType, cache, "spriteImageCacheの型が正しくない")
	})
}

// spriteImageCacheの操作テスト（実際の画像なしでテスト）
func TestSpriteImageCacheOperations(t *testing.T) {
	t.Parallel()
	t.Run("cache operations", func(t *testing.T) {
		t.Parallel()
		// 各テストで独立したシステムインスタンスを作成
		sys := NewRenderSpriteSystem()

		// 初期状態の確認
		initialLen := len(sys.spriteImageCache)
		assert.Equal(t, 0, initialLen, "新規作成時はキャッシュが空のはず")

		testKey := spriteImageCacheKey{
			SpriteSheetName: "test_sheet",
			SpriteKey:       "test_sprite",
		}

		// キーが存在しないことを確認
		_, exists := sys.spriteImageCache[testKey]
		assert.False(t, exists, "存在しないキーがtrueを返している")

		// キャッシュに値を設定（nilでテスト）
		sys.spriteImageCache[testKey] = nil

		// キーが存在することを確認
		_, exists = sys.spriteImageCache[testKey]
		assert.True(t, exists, "設定したキーが存在しない")

		// サイズが増えたことを確認
		assert.Len(t, sys.spriteImageCache, initialLen+1, "キャッシュサイズが正しくない")

		// キャッシュをクリア（テスト後の処理）
		delete(sys.spriteImageCache, testKey)

		// 元の状態に戻ったことを確認
		assert.Len(t, sys.spriteImageCache, initialLen, "キャッシュクリア後のサイズが正しくない")
	})
}

func TestViewportTileBounds(t *testing.T) {
	t.Parallel()

	t.Run("カメラがnilのときは原点基準で計算する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		minX, maxX, minY, maxY := viewportTileBounds(world, 2, nil)

		assert.Equal(t, -17, minX)
		assert.Equal(t, 17, maxX)
		assert.Equal(t, -13, minY)
		assert.Equal(t, 13, maxY)
	})

	t.Run("カメラの位置とスケールを考慮する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		camera := &gc.Camera{Scale: 2.0, Pos: consts.Coord[consts.WorldPixel]{X: 320, Y: 320}}

		minX, maxX, minY, maxY := viewportTileBounds(world, 0, camera)

		assert.Equal(t, 2, minX)
		assert.Equal(t, 17, maxX)
		assert.Equal(t, 4, minY)
		assert.Equal(t, 15, maxY)
	})

	t.Run("スケールが0以下なら1.0として扱う", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		camera := &gc.Camera{Scale: 0}

		minX, maxX, minY, maxY := viewportTileBounds(world, 2, camera)

		assert.Equal(t, -17, minX)
		assert.Equal(t, 17, maxX)
		assert.Equal(t, -13, minY)
		assert.Equal(t, 13, maxY)
	})
}

func TestInViewport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		grid *gc.GridElement
		want bool
	}{
		{
			name: "範囲内",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}},
			want: true,
		},
		{
			name: "X下限ちょうどは含む",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 5}},
			want: true,
		},
		{
			name: "X上限ちょうどは含む",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 5}},
			want: true,
		},
		{
			name: "X下限未満は範囲外",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: -1, Y: 5}},
			want: false,
		},
		{
			name: "X上限超過は範囲外",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 5}},
			want: false,
		},
		{
			name: "Y下限未満は範囲外",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: -1}},
			want: false,
		},
		{
			name: "Y上限超過は範囲外",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 11}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := inViewport(tt.grid, 0, 10, 0, 10)
			assert.Equal(t, tt.want, got)
		})
	}
}
