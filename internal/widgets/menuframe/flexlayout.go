package menuframe

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/furex/v2"

	ui "github.com/kijimaD/ruins/internal/widgets/internal/ui"
)

// flexItem は縦 flex の1行。w が nil の行は間隔だけを占めるスペーサ。
// grow が真の行は余り高さを吸収し、後続の行を下端へ押し付ける。
type flexItem struct {
	w      ui.Widget
	height int
	grow   bool
}

// frameRecorder は furex の描画コールバックで確定 frame を受け取り、widget の Layout へ流す。
// furex はレイアウト計算にだけ使い、描画は保持ツリーが Canvas 経由で行う。
type frameRecorder struct{ w ui.Widget }

// Draw は furex.Drawer を実装する。画面へは何も描かず、frame の記録だけを行う。
func (r frameRecorder) Draw(_ *ebiten.Image, frame image.Rectangle, _ *furex.View) {
	if r.w != nil {
		r.w.Layout(frame)
	}
}

// layoutFlexColumn は items を flexbox の縦方向で inner に配置し、各 widget の矩形を確定する。
// 幅は既定の AlignItemStretch で inner いっぱいに伸びる。計算は furex が行い、
// 結果の frame を frameRecorder が widget へ渡す。furex の View はインスタンス所有で、
// レイアウト経路にパッケージグローバルを持たないため並行に計算しても競合しない。
func layoutFlexColumn(inner image.Rectangle, items []flexItem) {
	root := &furex.View{
		Left:      inner.Min.X,
		Top:       inner.Min.Y,
		Width:     inner.Dx(),
		Height:    inner.Dy(),
		Direction: furex.Column,
	}
	for _, it := range items {
		child := &furex.View{Handler: frameRecorder{w: it.w}}
		if it.grow {
			child.Grow = 1
		} else {
			child.Height = it.height
		}
		root.AddChild(child)
	}
	root.Layout()
	// furex は確定 frame をハンドラ経由でしか渡さないため、捨て画像へ空描画して記録させる
	root.Draw(ebiten.NewImage(1, 1))
}
