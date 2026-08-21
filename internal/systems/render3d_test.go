package systems

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/render3d"
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
		return r3quad{p: [4]render3d.Vec{render3d.At(0, 0, z), render3d.At(0, 0, z), render3d.At(0, 0, z), render3d.At(0, 0, z)}}
	}
	quads := []r3quad{mk(5), mk(-3), mk(1)}
	sortQuadsByDepth(quads, func(v render3d.Vec) float64 { return v.Z }) // 奥行きは重心zそのもの
	assert.Equal(t, -3.0, quads[0].p[0].Z)
	assert.Equal(t, 1.0, quads[1].p[0].Z)
	assert.Equal(t, 5.0, quads[2].p[0].Z)
}

// 同一タイルの立て板は4隅が一致して奥行きが同値になる。走査順ではなく depth で前後を確定し、
// 大きい depth を後に、つまり手前に描くことを固定する。プレイヤーが足元のアイテムより上に出る根拠
func TestSortQuadsByDepth_奥行き同値はdepthの大きい方を手前にする(t *testing.T) {
	t.Parallel()
	mk := func(depth int) r3quad {
		// 全 quad を同一座標に置き key をビット同値にする。差は depth だけ
		return r3quad{p: [4]render3d.Vec{render3d.At(1, 0, 1), render3d.At(1, 0, 1), render3d.At(1, 0, 1), render3d.At(1, 0, 1)}, depth: depth}
	}
	// アイテム(1)を先、プレイヤー(3)を後に積んでも、逆に積んでも同じ順に並ぶことを確かめる
	quads := []r3quad{mk(1), mk(3)}
	sortQuadsByDepth(quads, func(v render3d.Vec) float64 { return v.Z })
	assert.Equal(t, 1, quads[0].depth, "小さい depth が先で奥")
	assert.Equal(t, 3, quads[1].depth, "大きい depth が後で手前")

	reversed := []r3quad{mk(3), mk(1)}
	sortQuadsByDepth(reversed, func(v render3d.Vec) float64 { return v.Z })
	assert.Equal(t, 1, reversed[0].depth, "積む順に依らず depth 昇順で並ぶ")
	assert.Equal(t, 3, reversed[1].depth)
}
