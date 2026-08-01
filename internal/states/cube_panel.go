package states

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// CubePanelState はキューブ内部のコントロールパネル。現ステージ、すなわち今いる内部の
// 全体情報を表示する。まずは総重量を読めるようにし、将来は拡張の操作UIを
// ここへ足していく器にする。
type CubePanelState struct {
	es.BaseState[w.World]

	totalWeight consts.Milligram // 内部に置いた物の総重量
}

var _ es.State[w.World] = &CubePanelState{}
var _ Configurable = &CubePanelState{}

// StateConfig はこのステートの設定を返す。背景をぼかして手前のパネルを際立たせる。
func (st *CubePanelState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: true}
}

// OnPause はステートが一時停止される際に呼ばれる。
func (st *CubePanelState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる。
func (st *CubePanelState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる。
func (st *CubePanelState) OnStop(_ w.World) error { return nil }

// OnStart は表示する全体情報を算出して保持する。表示中は状態が動かないため一度だけ計算する。
func (st *CubePanelState) OnStart(world w.World) error {
	interior := query.GetDungeon(world).CurrentStage
	st.totalWeight = query.CubeWeight(world, interior)
	return nil
}

// Update はキー入力で閉じるだけ。パネル表示中は時間を進めない。
func (st *CubePanelState) Update(_ w.World) (es.Transition[w.World], error) {
	keyboardInput := input.GetSharedKeyboardInput()
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	return st.ConsumeTransition(), nil
}

// Draw は全体情報を行で描く。将来はこの下へ拡張の操作項目を並べる。
func (st *CubePanelState) Draw(world w.World, screen *ebiten.Image) error {
	face := world.Resources.UIResources.Text.BodyFace

	drawText := func(str string, x, y consts.ScreenPixel, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.ColorScale.ScaleWithColor(c)
		text.Draw(screen, str, face, op)
	}

	const x consts.ScreenPixel = 40
	y := consts.ScreenPixel(40)
	line := func(s string, c color.Color) {
		drawText(s, x, y, c)
		y += 28
	}

	line("コントロールパネル", theme.TextPrimary)
	y += 8
	line(fmt.Sprintf("総重量: %s", st.totalWeight.KgString()), theme.TextPrimary)
	y += 8
	line("Esc で閉じる", theme.TextAccent)
	return nil
}
