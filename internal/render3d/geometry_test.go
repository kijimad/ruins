package render3d

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// assertVecInDelta は Vec を成分ごとに近似比較する。
func assertVecInDelta(t *testing.T, want, got Vec, delta float64) {
	t.Helper()
	assert.InDelta(t, want.X, got.X, delta)
	assert.InDelta(t, want.Y, got.Y, delta)
	assert.InDelta(t, want.Z, got.Z, delta)
}

// identity は 4x4 単位行列。
var identity = mat{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}

// TestVecOps はベクトルの加減・スケール・内積を固定する。
func TestVecOps(t *testing.T) {
	t.Parallel()
	assert.Equal(t, Vec{-2, 0, 2}, sub(Vec{1, 2, 3}, Vec{3, 2, 1}))
	assert.Equal(t, Vec{4, 4, 4}, Add(Vec{1, 2, 3}, Vec{3, 2, 1}))
	assert.Equal(t, Vec{2, 4, 6}, Scale(Vec{1, 2, 3}, 2))
	assert.InDelta(t, 10.0, dot(Vec{1, 2, 3}, Vec{3, 2, 1}), 1e-9) // 3+4+3
}

// TestCross は外積が右手系で軸を巡回することを固定する。
func TestCross(t *testing.T) {
	t.Parallel()
	assert.Equal(t, Vec{0, 0, 1}, cross(Vec{1, 0, 0}, Vec{0, 1, 0})) // x×y=z
	assert.Equal(t, Vec{1, 0, 0}, cross(Vec{0, 1, 0}, Vec{0, 0, 1})) // y×z=x
}

// TestNorm は正規化と、ゼロベクトルでゼロ除算しない保険を固定する。
func TestNorm(t *testing.T) {
	t.Parallel()
	t.Run("単位ベクトルへ正規化する", func(t *testing.T) {
		t.Parallel()
		assertVecInDelta(t, Vec{1, 0, 0}, norm(Vec{5, 0, 0}), 1e-9)
	})
	t.Run("ゼロベクトルはそのまま返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, Vec{0, 0, 0}, norm(Vec{0, 0, 0}))
	})
}

// TestApply_単位行列は点を保つ は行優先の変換が単位行列で恒等になることを固定する。
func TestApply_単位行列は点を保つ(t *testing.T) {
	t.Parallel()
	x, y, z, wc := apply(identity, Vec{2, 3, 4})
	assert.Equal(t, [4]float64{2, 3, 4, 1}, [4]float64{x, y, z, wc})
}

// TestMul_対角行列の積 は行列積が対角成分を掛け合わせることを固定する。
func TestMul_対角行列の積(t *testing.T) {
	t.Parallel()
	a := mat{2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1}
	b := mat{3, 0, 0, 0, 0, 3, 0, 0, 0, 0, 3, 0, 0, 0, 0, 1}
	assert.Equal(t, mat{6, 0, 0, 0, 0, 6, 0, 0, 0, 0, 6, 0, 0, 0, 0, 1}, mul(a, b))
}

// TestLookAt は視点が原点へ、注視点がカメラ前方(-Z)へ写ることを固定する。
func TestLookAt(t *testing.T) {
	t.Parallel()
	view := lookAt(Vec{0, 0, 5}, Vec{0, 0, 0}, Vec{0, 1, 0})
	t.Run("視点はカメラ原点に写る", func(t *testing.T) {
		t.Parallel()
		x, y, z, wc := apply(view, Vec{0, 0, 5})
		assert.InDelta(t, 0, x, 1e-9)
		assert.InDelta(t, 0, y, 1e-9)
		assert.InDelta(t, 0, z, 1e-9)
		assert.InDelta(t, 1, wc, 1e-9)
	})
	t.Run("注視点はカメラ前方5に写る", func(t *testing.T) {
		t.Parallel()
		_, _, z, _ := apply(view, Vec{0, 0, 0})
		assert.InDelta(t, -5, z, 1e-9)
	})
}

// TestPerspective は fov90/aspect1 で近面の端がNDCの端へ写ることを固定する。
func TestPerspective(t *testing.T) {
	t.Parallel()
	m := perspective(90, 1, 1, 100)
	cx, _, _, cw := apply(m, Vec{1, 0, -1}) // カメラ空間の右端
	assert.InDelta(t, 1, cw, 1e-9)          // w = -z = 1
	assert.InDelta(t, 1, cx/cw, 1e-9)       // NDC x = 1(右端)
}

// TestProjector_Point は投影の要、画面中央への写像とカメラ後方の除外を固定する。
func TestProjector_Point(t *testing.T) {
	t.Parallel()
	// 原点を真上から見下ろさず、+Z 側から見る素朴な視点を直接組む
	p := Projector{
		vp: mul(perspective(90, 1, 1, 100), lookAt(Vec{0, 0, 5}, Vec{0, 0, 0}, Vec{0, 1, 0})),
		sw: 960,
		sh: 720,
	}
	t.Run("注視点は画面中央へ写る", func(t *testing.T) {
		t.Parallel()
		got, ok := p.Point(Vec{0, 0, 0})
		assert.True(t, ok)
		assert.InDelta(t, 480, float64(got.X), 1e-6)
		assert.InDelta(t, 360, float64(got.Y), 1e-6)
	})
	t.Run("カメラ後方の点は描かない", func(t *testing.T) {
		t.Parallel()
		_, ok := p.Point(Vec{0, 0, 10}) // 視点より後ろ
		assert.False(t, ok)
	})
}
