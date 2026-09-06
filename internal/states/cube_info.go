package states

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// CubeInfoState は移動拠点キューブの情報を表示する。収納の総重量と燃料残量を読める。
// 表示中は状態が動かないので値は Draw で都度算出する。
type CubeInfoState struct {
	es.BaseState[w.World]

	cube ecs.Entity
}

var _ es.State[w.World] = &CubeInfoState{}

// NewCubeInfoState はキューブの情報画面を作る
func NewCubeInfoState(cube ecs.Entity) (es.State[w.World], error) {
	return &CubeInfoState{cube: cube}, nil
}

// OnPause はステートが一時停止される際に呼ばれる。
func (st *CubeInfoState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる。
func (st *CubeInfoState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる。
func (st *CubeInfoState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる。
func (st *CubeInfoState) OnStart(_ w.World) error { return nil }

// cubeInfoBindings は情報画面の束縛表。Esc で閉じるだけ
var cubeInfoBindings = []keybind.Binding{
	{Key: ebiten.KeyEscape, Action: inputmapper.ActionCloseMenu},
}

// Update はキー入力で閉じるだけ。表示中は時間を進めない。
func (st *CubeInfoState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := keybind.ReadInput(world, cubeInfoBindings); ok && action == inputmapper.ActionCloseMenu {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	return st.ConsumeTransition(), nil
}

// Draw は収納の総重量と燃料残量を行で描く。
func (st *CubeInfoState) Draw(world w.World, screen *ebiten.Image) error {
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

	weight := query.CubeWeight(world, st.cube)
	fuel := query.CubeFuelTotal(world, st.cube)

	line("Cube info", theme.TextPrimary)
	y += 8
	line(fmt.Sprintf("Total weight: %s", weight.KgString()), theme.TextPrimary)
	line(fmt.Sprintf("Fuel: %s", fuel.String()), theme.TextPrimary)
	y += 8
	line("Esc to close", theme.TextAccent)
	return nil
}
