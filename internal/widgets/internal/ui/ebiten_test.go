package ui_test

import (
	"image"
	"image/color"
	"os"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
)

// TestMain はebitenの描画コンテキスト内で全テストを実行する。
// EbitenCanvas の実描画に ebiten の実行状態が要るため必要。
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

// facePool は独立したフォントフェイスを再利用するプール。
//
// text/v2 は GoTextFaceSource 内に可変キャッシュを持ち、共有フェイスを並行描画すると競合する。
// フェイスをインスタンスが所有すれば共有が無くなり、ロック無しで並列に実描画できる。
// ただし独立フェイスを都度作るとフォント再パースのコストが乗る。プールは Get した借り手へ
// フェイスを排他所有で渡し Put で返すので、同時に走る借り手は必ず別フェイスを掴んで独立し、
// 使い終わったフェイスは次の借り手がウォームキャッシュのまま再利用する。プールは実際の並列度ぶん
// だけ遅延生成される。独立性と速度が両立する。
// プールにはポインタを入れる。値を入れると Put でインタフェースへボクシングされ割り当てが増える。
var facePool = sync.Pool{New: func() any { return mustLoadResources() }}

// mustLoadResources は独自フォントソースの UIResources を新しく作る。プールの補充に使う。
// 読み込みの直列化は loader.LoadUIResources が内部で持つので、ここでは守らない。
func mustLoadResources() *resources.UIResources {
	res, err := loader.LoadUIResources()
	if err != nil {
		panic(err)
	}
	return &res
}

// borrowRes はプールから独立フェイスを1つ借りる。ゴルーチンからも呼ぶので require でなく panic で確定する。
func borrowRes() *resources.UIResources {
	res, ok := facePool.Get().(*resources.UIResources)
	if !ok {
		panic("facePool: 想定外の型")
	}
	return res
}

// countOpaque は screen 内の不透明画素を数える。実描画が起きたかの判定に使う。
// ゴルーチンからも呼ぶので *testing.T は受けない。
func countOpaque(screen *ebiten.Image) int {
	b := screen.Bounds()
	pix := make([]byte, b.Dx()*b.Dy()*4)
	screen.ReadPixels(pix)
	n := 0
	for i := 3; i < len(pix); i += 4 {
		if pix[i] > 0 {
			n++
		}
	}
	return n
}

// buildRealPanel は実フォントで entityspec 相当のパネルを宣言的に組む。
func buildRealPanel(res resources.UIResources) *ui.Container {
	face := res.Text.BodyFace
	white := color.White
	cols := []int{90, 90}
	rows := []struct{ label, value string }{
		{"Vitality", "10"},
		{"Strength", "11"},
		{"Defense", "15"},
	}
	items := make([]ui.Widget, 0, len(rows))
	for _, r := range rows {
		// ラベルは左寄せ、値は右寄せ。実 spec パネルの specTableAligns に合わせる
		val := ui.NewText(r.value, face, white)
		val.Align = ui.AlignRight
		items = append(items, ui.Row(cols, ui.NewText(r.label, face, white), val))
	}
	style := ui.BoxStyle{Fill: color.Gray{Y: 30}, Border: color.White, BorderWidth: 1}
	return ui.Panel(style, 20, items...)
}

// drawRealPanel は独自フェイスで1枚描き、不透明画素数を返す。
func drawRealPanel(res resources.UIResources) int {
	screen := ebiten.NewImage(220, 100)
	cv := ui.NewEbitenCanvas(screen)
	panel := buildRealPanel(res)
	panel.Layout(image.Rect(0, 0, 220, 100))
	panel.Draw(cv)
	return countOpaque(screen)
}

func TestEbitenCanvas_実フォントで描くと非空になる(t *testing.T) {
	t.Parallel()
	res := borrowRes()
	defer facePool.Put(res)

	screen := ebiten.NewImage(220, 100)
	cv := ui.NewEbitenCanvas(screen)
	panel := buildRealPanel(*res)
	panel.Layout(image.Rect(0, 0, 220, 100))
	panel.Draw(cv)

	require.Positive(t, countOpaque(screen), "背景とテキストが描かれれば不透明画素が出る")
}

// TestEbitenCanvas_フェイスをインスタンス所有にすれば並列描画も競合しない は、各ゴルーチンが
// プールから独立フェイスを借りて実描画すれば、並列でもロック無しに競合しないことを確かめる。
func TestEbitenCanvas_フェイスをインスタンス所有にすれば並列描画も競合しない(t *testing.T) {
	t.Parallel()
	const workers = 8

	var wg sync.WaitGroup
	counts := make([]int, workers)
	for w := range workers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			res := borrowRes()
			defer facePool.Put(res)
			counts[id] = drawRealPanel(*res)
		}(w)
	}
	wg.Wait()

	for id, c := range counts {
		assert.Positive(t, c, "ゴルーチン %d の描画が非空", id)
	}
}
