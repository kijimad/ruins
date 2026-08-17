package styled

import (
	"image"
	"math"
	"sync/atomic"

	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// selectionBlinkFloor は明滅の谷のアルファ下限。選択が消えたと錯覚しないよう完全には落とさない
	selectionBlinkFloor = 0.55
	// selectionBlinkSpeed は1フレームあたりの位相の進み。ラジアンで表す。60fps で約1.0秒周期になる
	selectionBlinkSpeed = 0.1
)

// animationEnabled は選択カーソルの点滅を有効にするか。プロセス全体で1つ持ち、起動時に設定から
// 一度書き、描画時に読む。テスト描画は並行するので atomic で扱い競合を避ける
var animationEnabled atomic.Bool

func init() { animationEnabled.Store(true) }

// SetAnimationEnabled は選択カーソルの点滅可否を設定する。設定の DisableAnimation から起動時に呼ぶ
func SetAnimationEnabled(enabled bool) { animationEnabled.Store(enabled) }

// selectionBlinkAlpha はフレームカウンタから選択バーのアルファ倍率を返す。
// 位相0をピークにするため余弦を使う。よって単フレーム描画では常にフルアルファになり静止表示と一致する。
// animate が偽なら常にピークで静止する
func selectionBlinkAlpha(tick int, animate bool) float32 {
	if !animate {
		return 1.0
	}
	dip := (1 - math.Cos(float64(tick)*selectionBlinkSpeed)) / 2
	return float32(1.0 - (1.0-selectionBlinkFloor)*dip)
}

// selectionBar は選択中の行の背面に敷く点滅バー。選択時だけ NineSlice をアルファを揺らして描く。
// 時間源は自前のフレームカウンタで、ツリーが組み直されると新規生成され位相0から始まる。
// これによりカーソルが動いた瞬間はピークから明滅し直し、動いた行が強調される。
//
// content は重ねる内容行への参照。AnchorLayout は先頭の子だけで大きさを決めるため、
// バーを背面すなわち先頭の子に置くと行の高さがバーの大きさに縛られる。行の高さは内容が
// 決めるべきなので、大きさの問い合わせを内容へ委譲する
type selectionBar struct {
	widget   *widget.Widget
	src      *euiimage.NineSlice
	content  widget.PreferredSizeLocateableWidget
	selected bool
	tick     int
}

// newSelectionBar は選択バーを作る。src は選択バーの NineSlice 画像、content は重ねる内容行
func newSelectionBar(src *euiimage.NineSlice, content widget.PreferredSizeLocateableWidget, selected bool) *selectionBar {
	w := widget.NewWidget(
		widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			StretchHorizontal: true,
			StretchVertical:   true,
		}),
	)
	return &selectionBar{widget: w, src: src, content: content, selected: selected}
}

// setSelected は選択状態を切り替える
func (b *selectionBar) setSelected(selected bool) { b.selected = selected }

// GetWidget は widget.PreferredSizeLocateableWidget を満たす
func (b *selectionBar) GetWidget() *widget.Widget { return b.widget }

// SetLocation は widget.PreferredSizeLocateableWidget を満たす
func (b *selectionBar) SetLocation(rect image.Rectangle) { b.widget.Rect = rect }

// PreferredSize は重ねる内容行の大きさを返す。行の高さを内容に合わせるため委譲する
func (b *selectionBar) PreferredSize() (int, int) {
	if b.content == nil {
		return 0, 0
	}
	return b.content.PreferredSize()
}

// Update は毎フレーム時間を進める
func (b *selectionBar) Update(updateObj *widget.UpdateObject) {
	b.widget.Update(updateObj)
	b.tick++
}

// Render は選択時に選択バーを描く。tick からアルファを揺らして明滅させる
func (b *selectionBar) Render(screen *ebiten.Image) {
	b.widget.Render(screen)
	if !b.selected || b.src == nil {
		return
	}
	rect := b.widget.Rect
	width, height := rect.Dx(), rect.Dy()
	if width <= 0 || height <= 0 {
		return
	}
	alpha := selectionBlinkAlpha(b.tick, animationEnabled.Load())
	b.src.Draw(screen, width, height, func(opts *ebiten.DrawImageOptions) {
		opts.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
		opts.ColorScale.ScaleAlpha(alpha)
	})
}

// Validate は widget.PreferredSizeLocateableWidget を満たす
func (b *selectionBar) Validate() {}

// IsValidated は ebitenui の再検証判定を満たす
func (b *selectionBar) IsValidated() bool { return true }
