package vrt

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"sync"
	"testing"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// renderMu はebitenuiのグローバル入力ハンドラが並行アクセス安全でないため、
// ウィジェット生成からレンダリングまでの呼び出しを直列化する
var renderMu sync.Mutex

// captureScreen はebiten.Imageのピクセルデータを読み取りimage.NRGBAとして返す。
// 読み取り後にebiten.Imageを解放する
func captureScreen(screen *ebiten.Image) *image.NRGBA {
	bounds := screen.Bounds()
	img := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	screen.ReadPixels(img.Pix)
	screen.Deallocate()
	return img
}

// AssertGolden はウィジェットの描画結果をゴールデン画像と比較する。
// buildFn はrenderMu内で実行されるため、ebitenuiグローバル状態への並行アクセスを防ぐ。
// GOLDIE_UPDATE=1 で実行するとゴールデン画像を更新する
func AssertGolden(t *testing.T, buildFn func() *widget.Container, width, height int) {
	t.Helper()

	rendered := renderContainer(buildFn, width, height)
	pngData := encodePNG(t, rendered)
	assertPNGGolden(t, pngData)
}

// AssertScreenGolden はebiten.Image上に描画するコンポーネントのゴールデンテスト用。
// setupFn はrenderMu内で実行されるため、ebitenuiグローバル状態への並行アクセスを防ぐ。
// setupFn はウィジェット生成とUpdate等の準備を行い、drawFnを返す。
// messagelog, HUDなどebitenui.UIを内包するコンポーネントのテストに使用する
func AssertScreenGolden(t *testing.T, setupFn func() func(screen *ebiten.Image), width, height int) {
	t.Helper()

	renderMu.Lock()
	drawFn := setupFn()
	screen := ebiten.NewImage(width, height)
	drawFn(screen)
	img := captureScreen(screen)
	renderMu.Unlock()

	pngData := encodePNG(t, img)
	assertPNGGolden(t, pngData)
}

// renderContainer はビルダー関数でウィジェットを生成し、描画してimage.NRGBAとして返す。
// ウィジェット生成からレンダリングまでをrenderMu内で直列化する
func renderContainer(buildFn func() *widget.Container, width, height int) *image.NRGBA {
	renderMu.Lock()
	defer renderMu.Unlock()

	root := buildFn()
	ui := &ebitenui.UI{Container: root}
	screen := ebiten.NewImage(width, height)

	// レイアウト確定のため数フレーム回す
	for range 3 {
		ui.Update()
	}
	ui.Draw(screen)

	return captureScreen(screen)
}

// encodePNG はimage.Imageをpngバイト列にエンコードする
func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// noiseScale はトレランス算出の係数。
// ebitenuiのノイズはUI要素のエッジで発生し、エッジ量は画像面積の平方根に比例する。
// tolerance = noiseScale / √totalPixels で算出する
const noiseScale = 8.0

// toleranceForSize は画像のピクセル数からトレランス比率を算出する。
// ノイズ量はUIエッジに比例するため √面積 でスケーリングし、
// 小さい画像ほど高い比率、大きい画像ほど低い比率を返す
func toleranceForSize(width, height int) float64 {
	total := width * height
	if total <= 0 {
		return 0
	}
	return noiseScale / math.Sqrt(float64(total))
}

// assertPNGGolden はPNGバイト列をゴールデン画像と比較する。
// 画像サイズからトレランスを自動算出し、小さい画像は寛容に、大きい画像は厳密に判定する。
// GOLDIE_UPDATE=1 のときはトレランス内なら更新をスキップする
func assertPNGGolden(t *testing.T, pngData []byte) {
	t.Helper()

	cfg, err := png.DecodeConfig(bytes.NewReader(pngData))
	require.NoError(t, err, "PNGヘッダのデコードに失敗")
	toleranceRatio := toleranceForSize(cfg.Width, cfg.Height)

	if isGoldieUpdate() {
		g := newGoldie(t)
		goldenPath := g.GoldenFileName(t, t.Name())
		if existingData, err := os.ReadFile(goldenPath); err == nil {
			equalFn := pngPixelEqualFn(toleranceRatio)
			if equalFn(pngData, existingData) {
				t.Logf("トレランス内のため更新をスキップ: %s", goldenPath)
				return
			}
		}
		require.NoError(t, g.Update(t, t.Name(), pngData))
		t.Logf("ゴールデン画像を更新: %s", goldenPath)
		return
	}

	g := newGoldie(t,
		goldie.WithEqualFn(pngPixelEqualFn(toleranceRatio)),
		goldie.WithDiffFn(func(_, _ string) string {
			return fmt.Sprintf(
				"ピクセル差分が許容範囲を超えている（画像: %dx%d, トレランス: %.2f%%）",
				cfg.Width, cfg.Height, toleranceRatio*100,
			)
		}),
	)
	g.Assert(t, t.Name(), pngData)
}

// isGoldieUpdate は GOLDIE_UPDATE が有効かどうかを返す
func isGoldieUpdate() bool {
	switch os.Getenv("GOLDIE_UPDATE") {
	case "1", "true", "t":
		return true
	default:
		return false
	}
}

// newGoldie はgoldieインスタンスを生成する。サフィックスを.pngにして画像ファイルとして扱う
func newGoldie(t *testing.T, opts ...goldie.Option) *goldie.Goldie {
	t.Helper()
	all := make([]goldie.Option, 0, 1+len(opts))
	all = append(all, goldie.WithNameSuffix(".png"))
	all = append(all, opts...)
	return goldie.New(t, all...)
}

// channelTolerance16 は1チャンネルあたり許容する差分。RGBA() が返す16bit値(0..65535)で表す。
// フォントのアンチエイリアスはグリフ境界で数階調ゆれる。8bitで16階調ぶんを許容し、この揺れを
// 差分として数えない。文字や色の実変化はアルファが 0↔全開など全階調規模で動くので取りこぼさない。
// 差分の「数」だけでなく「大きさ」も見る二段トレランスになり、AAノイズ由来のフレークを防ぐ。
const channelTolerance16 = 16 * 257

// absDiffU32 は2値の差の絶対値を返す
func absDiffU32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// pngPixelEqualFn は2つのPNGバイト列をピクセル単位で比較する。
// toleranceRatio で許容する差分ピクセル比率を、channelTolerance16 で1画素内の許容振幅を指定する。
// どのチャンネルも許容振幅以内なら同一画素とみなし、振幅を超えた画素数が比率を超えたら不一致とする
func pngPixelEqualFn(toleranceRatio float64) goldie.EqualFn {
	return func(actual, expected []byte) bool {
		actualImg, err := png.Decode(bytes.NewReader(actual))
		if err != nil {
			return false
		}
		expectedImg, err := png.Decode(bytes.NewReader(expected))
		if err != nil {
			return false
		}

		eb := expectedImg.Bounds()
		ab := actualImg.Bounds()
		if eb.Dx() != ab.Dx() || eb.Dy() != ab.Dy() {
			return false
		}

		totalPixels := eb.Dx() * eb.Dy()
		maxAllowed := int(float64(totalPixels) * toleranceRatio)
		diffCount := 0
		for y := eb.Min.Y; y < eb.Max.Y; y++ {
			for x := eb.Min.X; x < eb.Max.X; x++ {
				er, eg, ebl, ea := expectedImg.At(x, y).RGBA()
				ar, ag, abl, aa := actualImg.At(x, y).RGBA()
				if absDiffU32(er, ar) > channelTolerance16 ||
					absDiffU32(eg, ag) > channelTolerance16 ||
					absDiffU32(ebl, abl) > channelTolerance16 ||
					absDiffU32(ea, aa) > channelTolerance16 {
					diffCount++
					if diffCount > maxAllowed {
						return false
					}
				}
			}
		}
		return true
	}
}
