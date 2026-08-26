package ui_test

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/vrt"
)

// TestMain はebitenの描画コンテキスト内で全テストを実行する。
// EbitenCanvas の実描画に ebiten の実行状態が要るため必要。
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
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

// TestEbitenCanvas_実フォントで描くと非空になる は EbitenCanvas が実フォントで実際に
// 描くことを確かめるスモークテスト。背景・枠・テキストを描き、不透明画素が出れば描画が起きている。
//
// このテストは実テキストを描く唯一のテストなので t.Parallel でも安全に単独で走る。
// ただし複数のテストが共有フェイスで実テキストを並行描画すると text/v2 のフォントキャッシュ
// GoTextFaceSource の内部キャッシュが競合する。これは UI ツールキットと無関係の text/v2 由来の制約で、
// 本番は描画ゴルーチンが1つなので無害。並行して実描画する pixel-golden を増やすときは、
// フェイスをテストごとに分けるか描画を直列化する。構築・レイアウト・ロジックの検証は
// フェイスに触れない fake canvas で行い、そちらは完全に並列でよい。
func TestEbitenCanvas_実フォントで描くと非空になる(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)

	screen := ebiten.NewImage(220, 100)
	cv := ui.NewEbitenCanvas(screen)
	u := ui.New(buildRealPanel(res))
	u.Layout(image.Rect(0, 0, 220, 100))
	u.Draw(cv)

	require.Positive(t, countOpaque(t, screen), "背景とテキストが描かれれば不透明画素が出る")
}
