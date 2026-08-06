package screeneffect

import "github.com/hajimehoshi/ebiten/v2"

// Filter は画面エフェクトのインターフェース
type Filter interface {
	// Apply はソース画像にエフェクトを適用して描画先に出力する
	Apply(dst, src *ebiten.Image)
}

// Pipeline はフィルタとオフスクリーンバッファを管理する。フレームバッファに加えて、
// メニュー等をまとめて描くためのオーバーレイ層も1枚持つ。
type Pipeline struct {
	filter    Filter
	offscreen *ebiten.Image
	lastW     int
	lastH     int
	overlay   alphaLayer
}

// NewPipeline は新しいPipelineを作成する
func NewPipeline(filter Filter) *Pipeline {
	return &Pipeline{
		filter: filter,
	}
}

// Begin はオフスクリーンバッファを準備して返す
// nilレシーバの場合は何もせずnilを返す
func (p *Pipeline) Begin(width, height int) *ebiten.Image {
	if p == nil {
		return nil
	}
	if p.offscreen == nil || p.lastW != width || p.lastH != height {
		p.offscreen = ebiten.NewImage(width, height)
		p.lastW = width
		p.lastH = height
	}
	p.offscreen.Clear()
	return p.offscreen
}

// End はフィルタを適用して最終画面に描画する
// nilレシーバの場合は何もしない
func (p *Pipeline) End(screen *ebiten.Image) {
	if p == nil || p.offscreen == nil {
		return
	}

	if p.filter == nil {
		screen.DrawImage(p.offscreen, nil)
		return
	}

	p.filter.Apply(screen, p.offscreen)
}

// BeginOverlay はフレームバッファとは別のオーバーレイ層を用意して返す。呼び出し側はここへ
// まとめて描き、CompositeOverlay で描画先へ一度だけ合成する。nilレシーバの場合はnilを返す。
func (p *Pipeline) BeginOverlay(width, height int) *ebiten.Image {
	if p == nil {
		return nil
	}
	return p.overlay.Begin(width, height)
}

// CompositeOverlay はオーバーレイ層を dst へ大域アルファで一度だけ重ねる。
// nilレシーバの場合は何もしない。
func (p *Pipeline) CompositeOverlay(dst *ebiten.Image, alpha float64) {
	if p == nil {
		return
	}
	p.overlay.Composite(dst, alpha)
}
