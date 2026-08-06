package maingame

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/screeneffect"
	w "github.com/kijimaD/ruins/internal/world"
)

// menuAlpha はメニュー層を世界へ合成する大域アルファ。透明度はこの一度の合成にだけ持たせる。
// メニューのパネルは層の中で不透明に描き、この値だけで世界に対して均一に透過させる。重なっても
// 世界の減衰は一定で、下メニューは上メニューの下で透けない。値は VRT を見ながら詰める。
const menuAlpha = 0.82

// renderer は1フレームの合成を受け持つ。世界を土台に、メニューをオーバーレイ層で重ね、ポスト処理
// チェーンをかけて画面へ出す。合成の詳細を game のループから切り離し、責務を分ける。
type renderer struct {
	frame    screeneffect.AlphaLayer // 世界とメニューを合成する土台
	overlay  screeneffect.AlphaLayer // メニューをまとめて不透明に描く層
	pipeline *screeneffect.Pipeline  // フレームにかけるポスト処理チェーン
}

// newRenderer はポスト処理チェーンを与えて renderer を作る。
func newRenderer(pipeline *screeneffect.Pipeline) renderer {
	return renderer{pipeline: pipeline}
}

// Draw は state 群を1フレームへ合成して screen へ出す。スタック[0]は世界で土台に不透明で描く。
// [1..]はメニューで、オーバーレイ層へ不透明に平坦化してから世界へ一度だけ透過合成する。これで
// 重なっても世界の減衰は一定で、下メニューは上メニューの下で透けない。
func (r *renderer) Draw(screen *ebiten.Image, states []es.State[w.World], world w.World) {
	b := screen.Bounds()
	frame := r.frame.Begin(b.Dx(), b.Dy())

	if len(states) > 0 {
		if err := states[0].Draw(world, frame); err != nil {
			log.Fatal(err)
		}
	}
	if len(states) > 1 {
		layer := r.overlay.Begin(b.Dx(), b.Dy())
		for _, state := range states[1:] {
			if err := state.Draw(world, layer); err != nil {
				log.Fatal(err)
			}
		}
		r.overlay.Composite(frame, menuAlpha)
	}

	r.pipeline.Apply(screen, frame)
}
