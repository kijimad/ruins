package replay_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt/replay"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// settingsOnMainMenu はメインメニューの上に設定メニューを積んだ state 構成を返す
func settingsOnMainMenu(_ w.World) []es.State[w.World] {
	return []es.State[w.World]{&states.MainMenuState{}, &states.SettingsMenuState{}}
}

// TestPlayScenario_実在メニューをAction列で駆動し遷移する は、使い捨て state を作らず
// 実在の設定メニューを本番の MainGame ループで駆動できることを固定する。
// メインメニューの上に設定メニューを積み、下へ1つ動かして決定する列を流すと、
// 戻る項目の決定で設定メニューが Pop され、メインメニューだけが残る。
func TestPlayScenario_実在メニューをAction列で駆動し遷移する(t *testing.T) {
	t.Parallel()
	game := replay.PlayScenario(t, settingsOnMainMenu,
		[]inputmapper.ActionID{
			inputmapper.ActionMenuDown,   // カーソルを Language から Back へ移す
			inputmapper.ActionMenuSelect, // Back を決定して設定メニューを閉じる
		},
		nil,
	)

	remaining := game.StateMachine.GetStates()
	require.Len(t, remaining, 1, "Back の決定で設定メニューが Pop され1段になる")
	_, ok := remaining[0].(*states.MainMenuState)
	assert.True(t, ok, "残るのはメインメニュー")
}

// TestPlayScenario_captureが各フレームで呼ばれる は capture が nil でないとき、
// Action 数+1のフレームで0起点連番で呼ばれることを固定する。+1 は最後の Action が返した
// 遷移を確定させるフレームで、そこで撮った絵が遷移後の画になる。
func TestPlayScenario_captureが各フレームで呼ばれる(t *testing.T) {
	t.Parallel()
	frames := 0
	replay.PlayScenario(t, settingsOnMainMenu,
		[]inputmapper.ActionID{
			inputmapper.ActionMenuDown,
			inputmapper.ActionMenuUp,
		},
		func(frame int, _ w.World, screen *ebiten.Image) {
			assert.Equal(t, frames, frame, "frame は0起点で連番")
			assert.NotNil(t, screen, "描画先が渡る")
			frames++
		},
	)
	assert.Equal(t, 3, frames, "Action 数+1のフレームで capture が呼ばれる")
}

// TestPlayScenario_押し込んだ先のstateも同じ列で駆動する は、Action 列が state の push を
// 跨いでも駆動が続くことを固定する。入力供給源は world が持つので、途中で積まれた state にも
// そのまま効く。メインメニューだけから始め、設定メニューを開いて閉じ、元へ戻るまでを1列で回す。
func TestPlayScenario_押し込んだ先のstateも同じ列で駆動する(t *testing.T) {
	t.Parallel()
	game := replay.PlayScenario(t,
		func(_ w.World) []es.State[w.World] {
			return []es.State[w.World]{&states.MainMenuState{}}
		},
		[]inputmapper.ActionID{
			inputmapper.ActionMenuDown,   // Start から Demo へ
			inputmapper.ActionMenuDown,   // Demo から Load へ
			inputmapper.ActionMenuDown,   // Load から Settings へ
			inputmapper.ActionMenuSelect, // Settings を開いて push する
			replay.NoInput,               // 積まれた設定メニューのタブ登録を待つ
			inputmapper.ActionMenuDown,   // 押し込んだ設定メニューで Language から Back へ
			inputmapper.ActionMenuSelect, // Back を決定して pop する
		},
		nil,
	)

	remaining := game.StateMachine.GetStates()
	require.Len(t, remaining, 1, "開いて閉じたのでメインメニューだけが残る")
	_, ok := remaining[0].(*states.MainMenuState)
	assert.True(t, ok, "残るのはメインメニュー")
}
