package maingame

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/screeneffect"
	w "github.com/kijimaD/ruins/internal/world"
)

// renderer は1フレームの合成を受け持つ。state をスタックの下から順にフレームへ描き、ポスト処理
// チェーンをかけて画面へ出す。合成を game のループから切り離し、game.go を薄く保つ。
type renderer struct {
	frame    *ebiten.Image // state を描くフレームバッファ。画面サイズで使い回す
	frameW   int
	frameH   int
	pipeline *screeneffect.Pipeline // フレームにかけるポスト処理チェーン
}

// newRenderer はポスト処理チェーンを与えて renderer を作る。
func newRenderer(pipeline *screeneffect.Pipeline) renderer {
	return renderer{pipeline: pipeline}
}

// Draw は state 群をスタックの下から順にフレームへ描き、ポスト処理をかけて screen へ出す。
func (r *renderer) Draw(screen *ebiten.Image, states []es.State[w.World], world w.World) {
	b := screen.Bounds()
	r.ensureFrame(b.Dx(), b.Dy())
	r.frame.Clear()

	for _, state := range states {
		if err := state.Draw(world, r.frame); err != nil {
			log.Fatal(err)
		}
	}

	r.pipeline.Apply(screen, r.frame)
}

// ensureFrame はフレームバッファを画面サイズで用意する。サイズが変わったら作り直す。
func (r *renderer) ensureFrame(width, height int) {
	if r.frame != nil && r.frameW == width && r.frameH == height {
		return
	}
	r.frame = ebiten.NewImage(width, height)
	r.frameW = width
	r.frameH = height
}
