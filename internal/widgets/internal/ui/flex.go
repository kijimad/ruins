package ui

import (
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yohamta/furex/v2"
)

// layoutProbe は furex から確定 frame を受け取るための捨て描画先。furex は frame を
// 描画コールバック経由でしか渡さないため、1×1 の画像へ空描画して記録させる。
// ハンドラは画像へ何も描かないので読み書きが無く、全コンテナで共有できる。
// 生成に ebiten の描画コンテキストが要るので遅延して一度だけ作る。
var (
	probeOnce sync.Once
	probe     *ebiten.Image
)

func layoutProbe() *ebiten.Image {
	probeOnce.Do(func() { probe = ebiten.NewImage(1, 1) })
	return probe
}

// frameTo は furex.Drawer を実装し、確定 frame を widget の Layout へ流す。
// furex はレイアウト計算にだけ使い、描画は保持ツリーが Canvas 経由で行う。
type frameTo struct{ w Widget }

// Draw は furex.Drawer を実装する。画面へは何も描かず、frame の記録だけを行う。
func (f frameTo) Draw(_ *ebiten.Image, frame image.Rectangle, _ *furex.View) {
	if f.w != nil {
		f.w.Layout(frame)
	}
}

// FlexItem は縦 flex の1行。W が nil の行は間隔だけを占めるスペーサ。
// Grow が真の行は余り高さを吸収し、後続の行を下端へ押し付ける。
type FlexItem struct {
	W      Widget
	Height int
	Grow   bool
}

// FlexColumn は items を flexbox の縦方向で inner に配置し、各 widget の矩形を確定する。
// 幅は既定の AlignItemStretch で inner いっぱいに伸びる。Container の固定高の積み上げと
// 違い、Grow の行で余り高さを吸収できる。画面の枠組みがフッタを下端へ固定するのに使う。
func FlexColumn(inner image.Rectangle, items []FlexItem) {
	root := &furex.View{
		Left:      inner.Min.X,
		Top:       inner.Min.Y,
		Width:     inner.Dx(),
		Height:    inner.Dy(),
		Direction: furex.Column,
	}
	for _, it := range items {
		child := &furex.View{Handler: frameTo{w: it.W}}
		if it.Grow {
			child.Grow = 1
		} else {
			child.Height = it.Height
		}
		root.AddChild(child)
	}
	root.Layout()
	root.Draw(layoutProbe())
}
