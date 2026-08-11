package systems

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLightGradient_キャッシュ はグラデーションが一度だけ生成され使い回されることを確認する。
func TestLightGradient_キャッシュ(t *testing.T) {
	t.Parallel()

	g1 := lightGradient()
	g2 := lightGradient()
	require.NotNil(t, g1, "グラデーションが生成される")
	assert.Same(t, g1, g2, "同じインスタンスを使い回す")
}

// TestBuildLightMap_darkness0は素通し は昼相当の暗さ0で光源を触らず素通しになることを確認する。
// darkness=0 は環境光を白で塗って早期リターンするため、World を参照しない。
func TestBuildLightMap_darkness0は素通し(t *testing.T) {
	t.Parallel()

	dst := ebiten.NewImage(64, 64)
	assert.NotPanics(t, func() {
		BuildLightMap(w.World{}, dst, 0)
	}, "darkness=0 の BuildLightMap はパニックしない")
}

// TestBuildLightMap_nil宛先は無処理 は宛先が nil でもパニックしないことを確認する。
func TestBuildLightMap_nil宛先は無処理(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		BuildLightMap(w.World{}, nil, 0.5)
	}, "nil 宛先の BuildLightMap はパニックしない")
}
