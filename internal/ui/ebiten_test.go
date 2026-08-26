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
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/vrt"
)

// TestMain はebitenの描画コンテキスト内で全テストを実行する。
// EbitenCanvas の実描画に ebiten の実行状態が要るため必要。
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

// isolatedResources は独自のフォントソースを持つ UIResources を新しく作る。
// vrt.SharedUIResources と違い sync.Once で共有しないので、フェイスのグリフキャッシュが
// このインスタンスに独立する。text/v2 は GoTextFaceSource 内に可変キャッシュを持ち、
// 共有フェイスを並行描画すると競合する。フェイスをインスタンスが所有すれば共有が無くなり、
// ロック無しで並列に実描画できる。これは ebitenui のグローバル問題と同じ「共有可変状態」の話で、
// 解も同じインスタンス所有になる。
func isolatedResources(t *testing.T) resources.UIResources {
	t.Helper()
	fonts, err := loader.LoadFonts()
	require.NoError(t, err)
	res, err := loader.LoadUIResources(fonts)
	require.NoError(t, err)
	return res
}

// countOpaque は screen 内の不透明画素を数える。実描画が起きたかの判定に使う。
func countOpaque(t *testing.T, screen *ebiten.Image) int {
	t.Helper()
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
		items = append(items,
			ui.Row(cols, ui.NewText(r.label, face, white), ui.NewText(r.value, face, white)))
	}
	style := ui.BoxStyle{Fill: color.Gray{Y: 30}, Border: color.White, BorderWidth: 1}
	return ui.Panel(style, 20, items...)
}

// drawRealPanel は独自フェイスで1枚描き、不透明画素数を返す。
func drawRealPanel(res resources.UIResources) int {
	screen := ebiten.NewImage(220, 100)
	cv := ui.NewEbitenCanvas(screen)
	u := ui.New(buildRealPanel(res))
	u.Layout(image.Rect(0, 0, 220, 100))
	u.Draw(cv)
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

func TestEbitenCanvas_実フォントで描くと非空になる(t *testing.T) {
	t.Parallel()
	res := isolatedResources(t)

	screen := ebiten.NewImage(220, 100)
	cv := ui.NewEbitenCanvas(screen)
	u := ui.New(buildRealPanel(res))
	u.Layout(image.Rect(0, 0, 220, 100))
	u.Draw(cv)

	require.Positive(t, countOpaque(t, screen), "背景とテキストが描かれれば不透明画素が出る")
}

// TestEbitenCanvas_フェイスをインスタンス所有にすれば並列描画も競合しない は、各ゴルーチンが
// 独自フェイスを持てば実描画も並列でロック無しに競合しないことを確かめる。
// 独立フェイスは require を含むので、ゴルーチン起動前にテストゴルーチンで作っておく。
func TestEbitenCanvas_フェイスをインスタンス所有にすれば並列描画も競合しない(t *testing.T) {
	t.Parallel()
	const workers = 8

	resList := make([]resources.UIResources, workers)
	for i := range resList {
		resList[i] = isolatedResources(t)
	}

	var wg sync.WaitGroup
	counts := make([]int, workers)
	for w := range workers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			counts[id] = drawRealPanel(resList[id])
		}(w)
	}
	wg.Wait()

	for id, c := range counts {
		assert.Positive(t, c, "ゴルーチン %d の描画が非空", id)
	}
}
