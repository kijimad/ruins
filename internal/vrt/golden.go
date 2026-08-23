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

// ebitenuiMu は ebitenui のグローバル状態、遅延イベントキュー・入力ハンドラ・NineSlice キャッシュ等が
// 並行アクセス安全でないため、それに触れる処理を直列化する。widget 生成、AddChild が遅延イベント
// キューへ append する、World 初期化、NineSlice キャッシュ、描画のいずれもこのグローバル状態に触れる。
var ebitenuiMu sync.Mutex

// WithUILock は ebitenuiMu を取得して fn を実行する。ebitenui のグローバル状態に触れる
// 生成・初期化・描画をこのヘルパに通し、ロックを取る箇所を1つに集約する。
func WithUILock(fn func()) {
	ebitenuiMu.Lock()
	defer ebitenuiMu.Unlock()
	fn()
}

// uiJobs は ebiten の画像操作をゲームループの goroutine へ渡すキュー。testHostGame.Update が
// 毎フレーム drain する。ebiten は画像操作をフレーム同期の内側でのみ安全に扱えるため、テスト
// goroutine から直接 NewImage/Draw/ReadPixels/Deallocate を呼ぶと ebiten の描画スレッドと競合する。
var uiJobs = make(chan func())

// RunOnGameThread は fn をゲームループの goroutine で実行し、完了を待つ。ebiten の画像操作を
// フレーム同期の内側へ寄せ、描画スレッドとのデータレースを構造的に無くす。
//
// fn には ebiten の画像操作だけを入れる。require や t.Fatal のような runtime.Goexit する処理を
// 入れてはならない。Goexit はゲームループの goroutine を殺し RunGame ごと落とす。検証は呼び出し側の
// テスト goroutine で行う。fn の panic は回収してテスト goroutine 側で再送出する。
//
// WithUILock 区間の中から呼ぶこと。ebitenui グローバルに触れる描画は、呼び出し側が保持する
// ebitenuiMu で他テストと直列化される。fn の中で RunOnGameThread を再帰呼び出ししてはならない。
func RunOnGameThread(fn func()) {
	done := make(chan any, 1)
	uiJobs <- func() {
		defer func() { done <- recover() }()
		fn()
	}
	if p := <-done; p != nil {
		panic(p)
	}
}

// readScreen はebiten.Imageのピクセルデータを読み取りimage.NRGBAとして返す。解放はしない。
// 呼び出し側が所有する画像を渡されるときはこちらを使う
func readScreen(screen *ebiten.Image) *image.NRGBA {
	bounds := screen.Bounds()
	img := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	screen.ReadPixels(img.Pix)
	return img
}

// captureScreen は readScreen したうえで ebiten.Image を解放する。
// このパッケージ内で作った使い捨ての描画先に使う
func captureScreen(screen *ebiten.Image) *image.NRGBA {
	img := readScreen(screen)
	screen.Deallocate()
	return img
}

// AssertContainerGolden は buildFn が返す widget.Container を描画し、ゴールデン画像と比較する。
// Container を ebitenui.UI に載せてレイアウト・描画までヘルパが行う。宣言的な widget ツリー
// （メニュー・タブ等）のテストに使う。自前で screen に描くものは AssertScreenGolden。
// GOLDIE_UPDATE=1 で実行するとゴールデン画像を更新する
func AssertContainerGolden(t *testing.T, buildFn func() *widget.Container, width, height int) {
	t.Helper()

	var img *image.NRGBA
	WithUILock(func() {
		root := buildFn()
		// 画像操作はゲームループのスレッドで行う。widget 構築は ebitenuiMu 下のこのテスト
		// スレッドで済ませ、描画と読み取りだけを寄せる
		RunOnGameThread(func() {
			ui := &ebitenui.UI{Container: root}
			screen := ebiten.NewImage(width, height)
			// レイアウト確定のため数フレーム回す
			for range 3 {
				ui.Update()
			}
			ui.Draw(screen)
			img = captureScreen(screen)
		})
	})

	pngData := encodePNG(t, img)
	assertPNGGolden(t, t.Name(), pngData)
}

// AssertScreenGolden は setupFn が返す描画関数で自前で ebiten.Image に描き、ゴールデン画像と比較する。
// レイアウト・描画は呼び出し側が制御する。messagelog・HUD など ebitenui.UI を内包する、または明示座標で
// Draw するコンポーネントに使う。widget.Container を渡すだけのものは AssertContainerGolden。
func AssertScreenGolden(t *testing.T, setupFn func() func(screen *ebiten.Image), width, height int) {
	t.Helper()

	var img *image.NRGBA
	WithUILock(func() {
		drawFn := setupFn()
		RunOnGameThread(func() {
			screen := ebiten.NewImage(width, height)
			drawFn(screen)
			img = captureScreen(screen)
		})
	})

	pngData := encodePNG(t, img)
	assertPNGGolden(t, t.Name(), pngData)
}

// AssertFrameGolden は読み取り済みの画像を name のゴールデン画像 testdata/name.png と比較する。
// 再生ドライバが各フレームでゲームスレッド上で読み取った画をそのまま渡す用途。ebiten 画像の
// 読み取りは PlayScenario がゲームスレッドで済ませ、このヘルパはエンコードと比較だけを行う。
// GOLDIE_UPDATE=1 で更新する。
func AssertFrameGolden(t *testing.T, name string, img *image.NRGBA) {
	t.Helper()
	assertPNGGolden(t, name, encodePNG(t, img))
}

// CaptureFrame は width×height の画像を作り draw で描いて画素を読み取り image.NRGBA で返す。
// NewImage・描画・ReadPixels・Deallocate をゲームループのスレッドで行い、ebiten の描画スレッドとの
// データレースを避ける。返す画像は plain な画素データなので、比較・検証は呼び出し側のテスト
// goroutine で安全に行える。WithUILock 区間の中から呼ぶこと。
func CaptureFrame(width, height int, draw func(screen *ebiten.Image)) *image.NRGBA {
	var img *image.NRGBA
	RunOnGameThread(func() {
		screen := ebiten.NewImage(width, height)
		draw(screen)
		img = captureScreen(screen)
	})
	return img
}

// encodePNG はimage.Imageをpngバイト列にエンコードする
func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// noiseScale はトレランス算出の係数。
// ノイズはUI要素のエッジで発生し、エッジ量は画像面積の平方根に比例する。
// tolerance = noiseScale / √totalPixels で算出する。
//
// 960×720 の全画面ステートで約0.036%、約250画素、になる値にしている。VRT は
// Config.DisableScreenFilter でレトロフィルタを切って撮る。切ったあと、同一マシンでの
// ハードウェア GL とソフトウェア GL の差は振幅8bitで2以内に収まり、channelTolerance16 が
// 吸収する。取りこぼすのは mesa のバージョン差でグリフが1px ずれる位置ノイズで、これは
// 振幅255のエッジ反転として数えられる。アイコンを多く重ねた全画面ほどエッジが増え、
// CI のソフトウェア GL で標準の83画素を超えて落ちていた。250画素はそこへ余裕を持たせた値。
//
// 検出できる粒度: メニューのラベル2行を書き換えた実測が0.0825%、570画素、で検出する。
// 250画素はその半分より下なので実変化は拾える。逆に単語1つぶんの微変化は250画素を
// 下回ると見逃す。文字単位の変化まで見たいならトレランスではなく専用の小さい golden を撮る。
//
// この係数を緩めすぎると実変化が静かに素通りする。上限の目安は2行ラベル変更 570画素の
// 半分、係数0.34。TestToleranceForSize がこの上限を守らせる。以前の45.0は全画面で5.4%あり、
// テキスト行が丸ごと別言語になっていた golden を見逃していた。
const noiseScale = 0.3

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

// assertPNGGolden はPNGバイト列を name のゴールデン画像と比較する。golden は testdata/name.png。
// name はサブテスト名でなく明示的に渡す。t.Run のスラッシュがパスに混ざらず、保存先が平置きになる。
// 画像サイズからトレランスを自動算出し、小さい画像は寛容に、大きい画像は厳密に判定する。
// GOLDIE_UPDATE=1 のときはトレランスを見ず無条件に上書きする。トレランス内スキップは
// 実変化を隠すので、更新は手動で走らせたときに必ず反映させる
func assertPNGGolden(t *testing.T, name string, pngData []byte) {
	t.Helper()

	cfg, err := png.DecodeConfig(bytes.NewReader(pngData))
	require.NoError(t, err, "failed to decode PNG header")
	toleranceRatio := toleranceForSize(cfg.Width, cfg.Height)

	if isGoldieUpdate() {
		g := newGoldie(t)
		require.NoError(t, g.Update(t, name, pngData))
		t.Logf("updated golden image: %s", g.GoldenFileName(t, name))
		return
	}

	g := newGoldie(t,
		goldie.WithEqualFn(pngPixelEqualFn(toleranceRatio)),
		goldie.WithDiffFn(func(_, _ string) string {
			return fmt.Sprintf(
				"pixel diff exceeds tolerance: image %dx%d, tolerance %.2f%%",
				cfg.Width, cfg.Height, toleranceRatio*100,
			)
		}),
	)
	g.Assert(t, name, pngData)
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
// 0x101 は color の RGBA() が8bit階調を16bit空間へ拡張する係数(v*0x101)。8bitで16階調ぶん許容する。
// フォントのアンチエイリアスはグリフ境界で数階調ゆれるが、これを差分として数えない。文字や色の
// 実変化はアルファが 0↔全開など全階調規模で動くので取りこぼさない。差分の「数」だけでなく
// 「大きさ」も見る二段トレランスになり、AAノイズ由来のフレークを防ぐ。
//
// 限界: 広域に及ぶ中振幅の変化、たとえば背景色の微変更は、各画素が許容振幅内に収まると見逃す。
// メニュー系の現状の用途では実害ないが、複雑な画面へ転用するときはこの盲点に注意する。
const channelTolerance16 = 16 * 0x101

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
