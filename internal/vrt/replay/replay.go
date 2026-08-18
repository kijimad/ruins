// Package replay は実在メニュー state を本番の Screen.Update フローでコマンド列駆動する。
// VRT の world 構築と直列化を再利用しつつ、menuloop のコマンド供給源へ依存する。
// vrt 本体へ menuloop 依存を持ち込むと menuloop のテストとで import 循環になるため、
// ここを vrt のサブパッケージに分けて循環を避ける。vrt の公開シンボルだけ使う。
package replay

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/stretchr/testify/require"
)

// PlayScenario は buildStates で組んだ state スタックの最上段へ Scenario のコマンドを
// 1フレーム1件で流し込み、各ステップ後に capture を呼ぶ。capture が nil なら駆動のみで描画しない。
// 使い捨て state を作らず、DoAction から先のメニュー挙動を実物のまま検証する。
//
// 最上段 state は menuloop.CommandDriven を実装している必要がある。返す StateMachine で
// 遷移結果を検査できる。ebitenui グローバルに触れるため一連の駆動を vrt.WithUILock で直列化する。
func PlayScenario(
	t *testing.T,
	buildStates func(w.World) []es.State[w.World],
	scenario menuloop.Scenario,
	capture func(step int, world w.World, screen *ebiten.Image),
) es.StateMachine[w.World] {
	t.Helper()
	world := vrt.InitVRTWorld(t)
	src := menuloop.NewScenarioReplay(scenario)

	var sm es.StateMachine[w.World]
	vrt.WithUILock(func() {
		sm = vrt.SetupStateMachine(t, world, buildStates)

		// 最上段の state へ供給源を差す。レイアウト確定フレームは供給源なしで回るのでコマンドを消費しない
		states := sm.GetStates()
		driven, ok := states[len(states)-1].(menuloop.CommandDriven)
		require.True(t, ok, "top state must implement menuloop.CommandDriven")
		driven.SetCommandSource(src)

		for step := range scenario.Commands {
			require.NoError(t, sm.Update(world), "scenario step %d update failed", step)
			if capture == nil {
				continue
			}
			// 各ステップの見た目を撮る。ステート列を下から重ねて描く
			screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
			for _, s := range sm.GetStates() {
				require.NoError(t, s.Draw(world, screen), "scenario step %d draw failed", step)
			}
			capture(step, world, screen)
		}

		// StateMachine は state が返した遷移を次フレーム冒頭で適用する。最後のコマンドが返した
		// Push/Pop を反映させるため、供給源が尽きた状態でもう1フレーム回して遷移を確定させる
		require.NoError(t, sm.Update(world), "final transition flush failed")
	})
	return sm
}
