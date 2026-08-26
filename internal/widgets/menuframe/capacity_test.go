package menuframe

import (
	"os"
	"testing"

	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/stretchr/testify/assert"
)

// TestMain はebitenグラフィックスコンテキスト内で全テストを実行する。
// UIResources のロードや widget の生成が ebiten の実行状態に依存するため必要
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

func TestListCapacity_同じ構成では同じ結果を返す(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)

	first := ListCapacity(world, true, true)
	second := ListCapacity(world, true, true)

	assert.Equal(t, first, second, "同じ構成なら何度呼んでも同じ結果になるはず")
	assert.Positive(t, first, "1行も収まらないのは異常")
}

func TestListCapacity_見出しとタブ帯が増えると収まる行数が減る(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)

	bare := ListCapacity(world, false, false)
	headerOnly := ListCapacity(world, true, false)
	headerAndTabs := ListCapacity(world, true, true)

	assert.Greater(t, bare, headerOnly, "見出しぶんだけ本体の高さが減り、収まる行数も減るはず")
	assert.Greater(t, headerOnly, headerAndTabs, "タブ帯ぶんだけさらに収まる行数が減るはず")
}
