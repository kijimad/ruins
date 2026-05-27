package systems

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
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
		assert.Equal(t, initialLen+1, len(sys.spriteImageCache), "キャッシュサイズが正しくない")

		// キャッシュをクリア（テスト後の処理）
		delete(sys.spriteImageCache, testKey)

		// 元の状態に戻ったことを確認
		assert.Equal(t, initialLen, len(sys.spriteImageCache), "キャッシュクリア後のサイズが正しくない")
	})
}
