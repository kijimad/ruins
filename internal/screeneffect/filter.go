package screeneffect

import "github.com/hajimehoshi/ebiten/v2"

// Filter は画面エフェクトのインターフェース
type Filter interface {
	// Apply はソース画像にエフェクトを適用して描画先に出力する
	Apply(dst, src *ebiten.Image)
}

// Pipeline はフィルタとオフスクリーンバッファを管理する
type Pipeline struct {
	filter    Filter
	offscreen *ebiten.Image
	lastW     int
	lastH     int
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
