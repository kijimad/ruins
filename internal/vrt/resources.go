package vrt

import (
	"sync"
	"testing"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/require"
)

// loadMu はリソース読み込みを直列化する。loader.LoadUIResources は内部で font source を
// 構築する共有状態を持ち並行安全でない。読み込みは各テストで一度きりの setup なので、
// ここを直列化しても実描画の並列は損なわない。読み込んだフェイス自体はインスタンスごとに
// 独立するので、描画は共有無しにロック無しで並列に走れる。
var loadMu sync.Mutex

// LoadTestUIResources はテスト用に独立した UIResources を読み込んで返す。
//
// 呼び出しごとに自前のフォントフェイスを構築するので、フェイスの共有が無くなる。text/v2 は
// GoTextFaceSource 内に可変キャッシュを持ち共有フェイスを並行描画すると競合するが、独立所有なら
// ロック無しで並列に実描画できる。読み込み自体は並行安全でないので loadMu で直列化する。
func LoadTestUIResources(t *testing.T) resources.UIResources {
	t.Helper()
	loadMu.Lock()
	defer loadMu.Unlock()

	fonts, err := loader.LoadFonts()
	require.NoError(t, err)
	uir, err := loader.LoadUIResources(fonts)
	require.NoError(t, err)
	return uir
}

// InitUIWorld は widget や画面の描画テスト用の world を返す。UI を描くテストはこれ1つを使う。
//
// ECS シングルトン、GameLog など、は testutil.InitTestWorld が用意し、フォントフェイスは
// LoadTestUIResources がテストごとに独立所有で足す。フルゲームを構築する重い InitVRTWorld は、
// 実プレイどおりフルフレームを駆動する states の golden_replay だけに使う。
func InitUIWorld(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)
	world.Resources.UIResources = LoadTestUIResources(t)
	return world
}
