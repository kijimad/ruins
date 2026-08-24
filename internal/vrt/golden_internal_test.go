package vrt

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToleranceForSize はトレランスが noiseScale / √面積 であることと、面積が増えるほど
// 比率が下がることを固定する。noiseScale の実数値には依存させない。係数の調整はノイズ実測に
// 応じて起きるので、値を焼き込むとその都度テストを書き換えることになる
func TestToleranceForSize(t *testing.T) {
	t.Parallel()

	t.Run("noiseScaleを面積の平方根で割った値を返す", func(t *testing.T) {
		t.Parallel()
		got := toleranceForSize(960, 720)
		assert.InDelta(t, noiseScale/math.Sqrt(960*720), got, 1e-12)
	})

	t.Run("面積が増えるほど比率が下がる", func(t *testing.T) {
		t.Parallel()
		sizes := [][2]int{{300, 30}, {400, 120}, {640, 480}, {960, 720}}

		prev := math.Inf(1)
		for _, s := range sizes {
			got := toleranceForSize(s[0], s[1])
			assert.Less(t, got, prev, "%dx%d は1つ小さい画像より低い比率になる", s[0], s[1])
			prev = got
		}
	})

	t.Run("ゼロサイズは0を返す", func(t *testing.T) {
		t.Parallel()
		assert.Zero(t, toleranceForSize(0, 0))
	})

	t.Run("フルスクリーンでもメニュー1行規模の変化を検出できる比率になっている", func(t *testing.T) {
		t.Parallel()
		// ラベル2行の書き換えが 0.0825% だった実測に対し、余裕を持って下回っていること。
		// 係数を緩めすぎると実変化が静かに素通りするので、上限を仕込んで気づけるようにする
		got := toleranceForSize(960, 720)
		require.Less(t, got, 0.0825/100/2, "メニュー行の変化を見逃さない比率であること")
	})
}

// TestAbsDiffU32 はどちらが大きくても差の絶対値を返すことを固定する
func TestAbsDiffU32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    uint32
		b    uint32
		want uint32
	}{
		{"aがbより大きい", 10, 3, 7},
		{"bがaより大きい", 3, 10, 7},
		{"aとbが等しい", 5, 5, 0},
		{"どちらもゼロ", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, absDiffU32(tt.a, tt.b))
		})
	}
}

// TestIsGoldieUpdate はGOLDIE_UPDATEの値ごとの判定を固定する。t.Setenvを使うためt.Parallelは呼ばない
func TestIsGoldieUpdate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"1なら有効", "1", true},
		{"trueなら有効", "true", true},
		{"tなら有効", "t", true},
		{"0なら無効", "0", false},
		{"falseなら無効", "false", false},
		{"未知の値は無効", "yes", false},
		{"未設定は無効", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GOLDIE_UPDATE", tt.value)
			assert.Equal(t, tt.want, isGoldieUpdate())
		})
	}
}

// solidPNG は幅4の単色画像をPNGへエンコードして返す
func solidPNG(t *testing.T, width int, c color.NRGBA) []byte {
	t.Helper()
	const height = 4
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// TestPngPixelEqualFn はPNGバイト列のピクセル単位比較の各分岐を固定する
func TestPngPixelEqualFn(t *testing.T) {
	t.Parallel()

	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.NRGBA{R: 0, G: 0, B: 0, A: 255}

	t.Run("actualが不正なPNGなら不一致", func(t *testing.T) {
		t.Parallel()
		expected := solidPNG(t, 4, white)
		eq := pngPixelEqualFn(1.0)
		assert.False(t, eq([]byte("not a png"), expected))
	})

	t.Run("expectedが不正なPNGなら不一致", func(t *testing.T) {
		t.Parallel()
		actual := solidPNG(t, 4, white)
		eq := pngPixelEqualFn(1.0)
		assert.False(t, eq(actual, []byte("not a png")))
	})

	t.Run("サイズが異なれば不一致", func(t *testing.T) {
		t.Parallel()
		actual := solidPNG(t, 4, white)
		expected := solidPNG(t, 5, white)
		eq := pngPixelEqualFn(1.0)
		assert.False(t, eq(actual, expected))
	})

	t.Run("完全に同一画像なら一致", func(t *testing.T) {
		t.Parallel()
		actual := solidPNG(t, 4, white)
		expected := solidPNG(t, 4, white)
		eq := pngPixelEqualFn(0)
		assert.True(t, eq(actual, expected))
	})

	t.Run("許容振幅内の差分は同一とみなす", func(t *testing.T) {
		t.Parallel()
		// 8bit差5は16bit換算で5*257=1285、channelTolerance16の4112を下回るのでノイズ扱いになる
		actual := solidPNG(t, 4, color.NRGBA{R: 250, G: 250, B: 250, A: 255})
		expected := solidPNG(t, 4, white)
		eq := pngPixelEqualFn(0)
		assert.True(t, eq(actual, expected))
	})

	t.Run("許容振幅を超える差分ピクセル数が比率以下なら一致", func(t *testing.T) {
		t.Parallel()
		// 4x4=16画素中9画素を黒にする。比率0.6の上限は int(16*0.6)=9 なので9画素までは許容される
		expectedImg := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		actualImg := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for y := range 4 {
			for x := range 4 {
				expectedImg.SetNRGBA(x, y, white)
				if y*4+x < 9 {
					actualImg.SetNRGBA(x, y, black)
				} else {
					actualImg.SetNRGBA(x, y, white)
				}
			}
		}
		var actualBuf, expectedBuf bytes.Buffer
		require.NoError(t, png.Encode(&actualBuf, actualImg))
		require.NoError(t, png.Encode(&expectedBuf, expectedImg))

		eq := pngPixelEqualFn(0.6)
		assert.True(t, eq(actualBuf.Bytes(), expectedBuf.Bytes()))
	})

	t.Run("許容振幅を超える差分ピクセル数が比率を超えれば不一致", func(t *testing.T) {
		t.Parallel()
		// 4x4=16画素中10画素を黒にする。比率0.5の上限は int(16*0.5)=8 を超えるので不一致になる
		expectedImg := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		actualImg := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for y := range 4 {
			for x := range 4 {
				expectedImg.SetNRGBA(x, y, white)
				if y*4+x < 10 {
					actualImg.SetNRGBA(x, y, black)
				} else {
					actualImg.SetNRGBA(x, y, white)
				}
			}
		}
		var actualBuf, expectedBuf bytes.Buffer
		require.NoError(t, png.Encode(&actualBuf, actualImg))
		require.NoError(t, png.Encode(&expectedBuf, expectedImg))

		eq := pngPixelEqualFn(0.5)
		assert.False(t, eq(actualBuf.Bytes(), expectedBuf.Bytes()))
	})
}
