package screeneffect

import "github.com/hajimehoshi/ebiten/v2"

// Filter は画面エフェクトのインターフェース
type Filter interface {
	// Apply はソース画像にエフェクトを適用して描画先に出力する
	Apply(dst, src *ebiten.Image)
}

// Pipeline は Filter を適用順に並べたポスト処理チェーン。src へ各 Filter を順にかけて dst へ出す。
// 中間結果は内部の ping-pong バッファに置く。エフェクトを増やすときは NewPipeline に足すだけでよい。
type Pipeline struct {
	filters []Filter
	scratch [2]*ebiten.Image // 多段適用の中間バッファ。dst と同じサイズで使い回す
	lastW   int
	lastH   int
}

// NewPipeline は Filter を適用順に受け取ってチェーンを作る。Filter が無ければ素通しになる。
func NewPipeline(filters ...Filter) *Pipeline {
	return &Pipeline{filters: filters}
}

// Apply は src へ Filter を順にかけて dst へ出力する。有効な Filter が無ければ src をそのまま dst へ
// 写す。nilレシーバや nil の src では何もしない。
func (p *Pipeline) Apply(dst, src *ebiten.Image) {
	if p == nil || dst == nil || src == nil {
		return
	}

	active := p.activeFilters()
	if len(active) == 0 {
		dst.DrawImage(src, nil)
		return
	}
	if len(active) == 1 {
		// 1枚だけなら中間バッファは要らない。dst へ直接かける。
		active[0].Apply(dst, src)
		return
	}

	b := dst.Bounds()
	p.ensureScratch(b.Dx(), b.Dy())
	cur := src
	for i, f := range active {
		if i == len(active)-1 {
			f.Apply(dst, cur)
			break
		}
		out := p.scratch[i%2]
		out.Clear()
		f.Apply(out, cur)
		cur = out
	}
}

// activeFilters は nil を除いた有効な Filter だけを返す。
func (p *Pipeline) activeFilters() []Filter {
	fs := make([]Filter, 0, len(p.filters))
	for _, f := range p.filters {
		if f != nil {
			fs = append(fs, f)
		}
	}

	return fs
}

// ensureScratch は中間バッファを dst と同じサイズで用意する。サイズが変わったら作り直す。
func (p *Pipeline) ensureScratch(width, height int) {
	if p.scratch[0] != nil && p.lastW == width && p.lastH == height {
		return
	}
	p.scratch[0] = ebiten.NewImage(width, height)
	p.scratch[1] = ebiten.NewImage(width, height)
	p.lastW = width
	p.lastH = height
}
