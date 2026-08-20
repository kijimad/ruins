// Package replay は実在メニュー state を本番の MainGame ループで Action 列から駆動する。
// 本番との違いは入力の出どころだけで、キーボードから変換する代わりに Action 列をそのまま
// 供給する。更新も描画も本番と同じ MainGame.Update・MainGame.Draw を通す。
//
// vrt 本体へ menuloop 依存を持ち込むと menuloop のテストとで import 循環になるため、
// ここを vrt のサブパッケージに分けて循環を避ける。vrt の公開シンボルだけ使う。
package replay

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/maingame"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/stretchr/testify/require"
)

// PlayScenario は buildStates で組んだ state を本番の MainGame ループで actions から駆動し、
// 駆動し終えた MainGame を返す。遷移結果は game.StateMachine から検査できる。
//
// フレーム数は len(actions)+1 になる。StateMachine は state が返した遷移を次フレーム冒頭で
// 適用するので、最後の Action の Push/Pop を確定させる1フレームを足す。capture は各フレームの
// 描画後に0起点のフレーム番号で呼ぶ。nil なら駆動のみで描画しない。screen は capture から
// 戻ったところで解放するので、抱え込まずその場で読み切る。
//
// state 側に再生用の口は要らない。入力供給源は world が持ち、押し込んだ先の state にも同じ源が
// 効く。ebitenui グローバルに触れるため一連の駆動を vrt.WithUILock で直列化する。
//
// overlay が Active な間は Screen の入力ゲートが overlay へ入力を渡すが、overlay も
// keybind.ReadInput で同じ供給源から読むので、詳細モーダルを開いた先も Action 列で駆動できる。
func PlayScenario(
	t *testing.T,
	buildStates func(w.World) []es.State[w.World],
	actions []inputmapper.ActionID,
	capture func(frame int, world w.World, screen *ebiten.Image),
) *maingame.MainGame {
	t.Helper()
	world := vrt.InitVRTWorld(t)

	var game *maingame.MainGame
	vrt.WithUILock(func() {
		// レイアウト確定フレームは供給源を差す前に回す。Action を消費させない
		sm := vrt.SetupStateMachine(t, world, buildStates)

		var err error
		game, err = maingame.NewMainGame(world, sm)
		require.NoError(t, err)
		world.Resources.InputSource = actionSource(actions)

		for frame := range len(actions) + 1 {
			if err := game.Update(); err != nil {
				// 全ての state が Pop されたときの正常終了。以降は駆動するものが無い
				require.ErrorIs(t, err, ebiten.Termination, "frame %d update failed", frame)
				return
			}
			if capture == nil {
				continue
			}
			screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
			game.Draw(screen)
			capture(frame, world, screen)
			screen.Deallocate()
		}
	})
	return game
}

// NoInput は入力の無いフレームを表す。本番はキーを押していないフレームが大半なので、
// 待ちが要る場面ではこれを並べる。state を push した直後のフレームは新しい Screen の
// タブ登録がまだ済んでいないため、続けて操作するなら1つ挟む
const NoInput inputmapper.ActionID = ""

// actionSource は Action 列を1フレーム1件で吐く供給源を作る。NoInput と列が尽きたあとは
// 偽を返し、Screen はキーボード経路へ戻る
func actionSource(actions []inputmapper.ActionID) inputmapper.Source {
	rest := actions
	return func() (inputmapper.ActionID, bool) {
		if len(rest) == 0 {
			return NoInput, false
		}
		action := rest[0]
		rest = rest[1:]
		return action, action != NoInput
	}
}
