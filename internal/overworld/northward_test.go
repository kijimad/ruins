package overworld

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGolden_Northward は開始の行から北へ20チャンクの地上を、座標を書き込んだ1枚の画像で固定する。地上の生成を変えると、
// 道・区域・集落の並びの変わり方がこの画像の差分で見える。overworldpng の出力と同じ描き方
func TestGolden_Northward(t *testing.T) {
	t.Parallel()

	img, err := RenderNorthward(testutil.InitTestWorld(t), 12345, 20, 1, 4)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	goldie.New(t, goldie.WithNameSuffix(".png")).Assert(t, t.Name(), buf.Bytes())
}

func TestRenderNorthward_地図の左と上に座標の余白を足す(t *testing.T) {
	t.Parallel()

	img, err := RenderNorthward(testutil.InitTestWorld(t), 7, 1, 0, 2)
	require.NoError(t, err)
	// 地上は 9 列 × 24 タイル、行は開始の行と北の1行の2行
	assert.Equal(t, image.Rect(0, 0, gridMarginLeft+9*24*2, gridMarginTop+2*24*2), img.Bounds())
}

func TestGenerateNorthward_帯ローカルの上端と下端が北端と南端の絶対の行になる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	require.NoError(t, generateNorthward(world, 7, 3, 1))
	sb := query.GetSeamlessBand(world)
	require.NotNil(t, sb)
	assert.Equal(t, sb.SpawnChunkY()-3, sb.AbsChunkRow(0))
	assert.Equal(t, sb.SpawnChunkY()+1, sb.AbsChunkRow(5*sb.ChunkH-1))
}

func TestWithChunkGrid_余白を足しチャンクの境目に線を引く(t *testing.T) {
	t.Parallel()

	m := image.NewRGBA(image.Rect(0, 0, 40, 60))
	img := withChunkGrid(m, 20, 30, 2, 2, 1)
	assert.Equal(t, image.Rect(0, 0, gridMarginLeft+40, gridMarginTop+60), img.Bounds())
	// 透明な地図の上に半透明の白い線を重ねる。座標の文字の届かない、チャンクの下の方で確かめる
	y := gridMarginTop + 55
	assert.Equal(t, color.RGBA{R: 160, G: 160, B: 160, A: 160}, img.RGBAAt(gridMarginLeft+20, y), "列の境目")
	assert.Equal(t, color.RGBA{}, img.RGBAAt(gridMarginLeft+10, y), "チャンクの中")
}
