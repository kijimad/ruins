package systems

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
)

// TestTileVisFactor はタイル描画情報から明るさ・可視・光源色を導く純関数を固定する。
// 3Dの Draw を通さず、Darkness の明るさ反映と可視/記憶/未探索の分岐を単体で検証する。
func TestTileVisFactor(t *testing.T) {
	t.Parallel()

	t.Run("可視はDarknessを明るさへ反映する", func(t *testing.T) {
		t.Parallel()
		bright, drawable, visible, light := tileVisFactor(TileRenderVisible{Darkness: 0.3})
		assert.InDelta(t, 0.7, bright, 1e-9, "bright = 1 - Darkness")
		assert.True(t, drawable)
		assert.True(t, visible)
		assert.Equal(t, [3]float64{1, 1, 1}, light, "無色の光源は白のまま")
	})

	t.Run("可視の光源色は最大成分で正規化して色味だけ返す", func(t *testing.T) {
		t.Parallel()
		_, _, _, light := tileVisFactor(TileRenderVisible{LightColor: color.RGBA{R: 255, G: 128, B: 0, A: 255}})
		assert.InDelta(t, 1.0, light[0], 1e-9)
		assert.InDelta(t, 128.0/255, light[1], 1e-9)
		assert.InDelta(t, 0.0, light[2], 1e-9)
	})

	t.Run("記憶は描画可だが可視ではない", func(t *testing.T) {
		t.Parallel()
		bright, drawable, visible, light := tileVisFactor(TileRenderRemembered{Darkness: 0.75})
		assert.InDelta(t, 0.25, bright, 1e-9)
		assert.True(t, drawable)
		assert.False(t, visible)
		assert.Equal(t, [3]float64{1, 1, 1}, light, "記憶は光源色を持たず白")
	})

	t.Run("未探索は描画しない", func(t *testing.T) {
		t.Parallel()
		bright, drawable, visible, _ := tileVisFactor(nil)
		assert.Zero(t, bright)
		assert.False(t, drawable)
		assert.False(t, visible)
	})
}

// assertVecInDelta は r3vec を成分ごとに近似比較する。
func assertVecInDelta(t *testing.T, want, got r3vec, delta float64) {
	t.Helper()
	assert.InDelta(t, want.x, got.x, delta)
	assert.InDelta(t, want.y, got.y, delta)
	assert.InDelta(t, want.z, got.z, delta)
}

// r3identity は 4x4 単位行列。
var r3identity = r3mat{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}

// TestR3VecOps はベクトルの加減・スケール・内積を固定する。
func TestR3VecOps(t *testing.T) {
	t.Parallel()
	assert.Equal(t, r3vec{-2, 0, 2}, r3sub(r3vec{1, 2, 3}, r3vec{3, 2, 1}))
	assert.Equal(t, r3vec{4, 4, 4}, r3add(r3vec{1, 2, 3}, r3vec{3, 2, 1}))
	assert.Equal(t, r3vec{2, 4, 6}, r3scale(r3vec{1, 2, 3}, 2))
	assert.InDelta(t, 10.0, r3dot(r3vec{1, 2, 3}, r3vec{3, 2, 1}), 1e-9) // 3+4+3
}

// TestR3Cross は外積が右手系で軸を巡回することを固定する。
func TestR3Cross(t *testing.T) {
	t.Parallel()
	assert.Equal(t, r3vec{0, 0, 1}, r3cross(r3vec{1, 0, 0}, r3vec{0, 1, 0})) // x×y=z
	assert.Equal(t, r3vec{1, 0, 0}, r3cross(r3vec{0, 1, 0}, r3vec{0, 0, 1})) // y×z=x
}

// TestR3Norm は正規化と、ゼロベクトルでゼロ除算しない保険を固定する。
func TestR3Norm(t *testing.T) {
	t.Parallel()
	t.Run("単位ベクトルへ正規化する", func(t *testing.T) {
		t.Parallel()
		assertVecInDelta(t, r3vec{1, 0, 0}, r3norm(r3vec{5, 0, 0}), 1e-9)
	})
	t.Run("ゼロベクトルはそのまま返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, r3vec{0, 0, 0}, r3norm(r3vec{0, 0, 0}))
	})
}

// TestR3Apply_単位行列は点を保つ は行優先の変換が単位行列で恒等になることを固定する。
func TestR3Apply_単位行列は点を保つ(t *testing.T) {
	t.Parallel()
	x, y, z, wc := r3apply(r3identity, r3vec{2, 3, 4})
	assert.Equal(t, [4]float64{2, 3, 4, 1}, [4]float64{x, y, z, wc})
}

// TestR3Mul_対角行列の積 は行列積が対角成分を掛け合わせることを固定する。
func TestR3Mul_対角行列の積(t *testing.T) {
	t.Parallel()
	a := r3mat{2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1}
	b := r3mat{3, 0, 0, 0, 0, 3, 0, 0, 0, 0, 3, 0, 0, 0, 0, 1}
	assert.Equal(t, r3mat{6, 0, 0, 0, 0, 6, 0, 0, 0, 0, 6, 0, 0, 0, 0, 1}, r3mul(a, b))
}

// TestR3LookAt は視点が原点へ、注視点がカメラ前方(-Z)へ写ることを固定する。
func TestR3LookAt(t *testing.T) {
	t.Parallel()
	view := r3lookAt(r3vec{0, 0, 5}, r3vec{0, 0, 0}, r3vec{0, 1, 0})
	t.Run("視点はカメラ原点に写る", func(t *testing.T) {
		t.Parallel()
		x, y, z, wc := r3apply(view, r3vec{0, 0, 5})
		assert.InDelta(t, 0, x, 1e-9)
		assert.InDelta(t, 0, y, 1e-9)
		assert.InDelta(t, 0, z, 1e-9)
		assert.InDelta(t, 1, wc, 1e-9)
	})
	t.Run("注視点はカメラ前方5に写る", func(t *testing.T) {
		t.Parallel()
		_, _, z, _ := r3apply(view, r3vec{0, 0, 0})
		assert.InDelta(t, -5, z, 1e-9)
	})
}

// TestR3Perspective は fov90/aspect1 で近面の端がNDCの端へ写ることを固定する。
func TestR3Perspective(t *testing.T) {
	t.Parallel()
	m := r3perspective(90, 1, 1, 100)
	cx, _, _, cw := r3apply(m, r3vec{1, 0, -1}) // カメラ空間の右端
	assert.InDelta(t, 1, cw, 1e-9)              // w = -z = 1
	assert.InDelta(t, 1, cx/cw, 1e-9)           // NDC x = 1(右端)
}

// TestProjectToScreen は投影の要、画面中央への写像とカメラ後方の除外を固定する。
func TestProjectToScreen(t *testing.T) {
	t.Parallel()
	vp := r3mul(r3perspective(90, 1, 1, 100), r3lookAt(r3vec{0, 0, 5}, r3vec{0, 0, 0}, r3vec{0, 1, 0}))
	t.Run("注視点は画面中央へ写る", func(t *testing.T) {
		t.Parallel()
		x, y, ok := projectToScreen(vp, r3vec{0, 0, 0}, 960, 720)
		assert.True(t, ok)
		assert.InDelta(t, 480, x, 1e-6)
		assert.InDelta(t, 360, y, 1e-6)
	})
	t.Run("カメラ後方の点は描かない", func(t *testing.T) {
		t.Parallel()
		_, _, ok := projectToScreen(vp, r3vec{0, 0, 10}, 960, 720) // 視点より後ろ
		assert.False(t, ok)
	})
}

// TestNormalizeLight は光源色の正規化と、無効時に白へフォールバックすることを固定する。
func TestNormalizeLight(t *testing.T) {
	t.Parallel()
	t.Run("最大成分で正規化し色味だけ残す", func(t *testing.T) {
		t.Parallel()
		got := normalizeLight(color.RGBA{R: 255, G: 128, B: 0, A: 255})
		assert.InDelta(t, 1.0, got[0], 1e-9)
		assert.InDelta(t, 128.0/255, got[1], 1e-9)
		assert.InDelta(t, 0.0, got[2], 1e-9)
	})
	t.Run("アルファ0は白を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, [3]float64{1, 1, 1}, normalizeLight(color.RGBA{R: 255, A: 0}))
	})
	t.Run("黒は白を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, [3]float64{1, 1, 1}, normalizeLight(color.RGBA{A: 255}))
	})
}

// TestRender3DSystem_String は w.Renderer の識別名を固定する。
func TestRender3DSystem_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Render3DSystem", NewRender3DSystem().String())
}

// TestVisFactorFunc_FOV無効は全タイルを等倍で描く は視界無効時の分岐を固定する。
func TestVisFactorFunc_FOV無効は全タイルを等倍で描く(t *testing.T) {
	t.Parallel()
	sys := &Render3DSystem{UseFOV: false}
	bright, drawable, visible, light := sys.visFactorFunc(w.World{})(&gc.GridElement{})
	assert.InDelta(t, 1.0, bright, 1e-9)
	assert.True(t, drawable)
	assert.True(t, visible)
	assert.Equal(t, [3]float64{1, 1, 1}, light)
}

// TestSortQuadsByDepth_奥から手前へ並べる は画家アルゴリズムの前段ソートを固定する。
func TestSortQuadsByDepth_奥から手前へ並べる(t *testing.T) {
	t.Parallel()
	mk := func(z float64) r3quad {
		return r3quad{p: [4]r3vec{{0, 0, z}, {0, 0, z}, {0, 0, z}, {0, 0, z}}}
	}
	quads := []r3quad{mk(5), mk(-3), mk(1)}
	sortQuadsByDepth(quads, r3identity) // view=単位行列なので key=重心z
	assert.Equal(t, -3.0, quads[0].p[0].z)
	assert.Equal(t, 1.0, quads[1].p[0].z)
	assert.Equal(t, 5.0, quads[2].p[0].z)
}
