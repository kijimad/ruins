package replay_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menuloop"
	states "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt/replay"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlayScenario_実在メニューをコマンド列で駆動し遷移する は、使い捨て state を作らず
// 実在の設定メニューを本番の Screen.Update フローで駆動できることを固定する。
// メインメニューの上に設定メニューを積み、下へ1つ動かして決定する列を流すと、
// 戻る項目の決定で設定メニューが Pop され、メインメニューだけが残る。
func TestPlayScenario_実在メニューをコマンド列で駆動し遷移する(t *testing.T) {
	t.Parallel()
	sm := replay.PlayScenario(t,
		func(_ w.World) []es.State[w.World] {
			return []es.State[w.World]{&states.MainMenuState{}, &states.SettingsMenuState{}}
		},
		menuloop.Scenario{Commands: []menuloop.Command{
			inputmapper.ActionMenuDown,   // カーソルを Language から Back へ移す
			inputmapper.ActionMenuSelect, // Back を決定して設定メニューを閉じる
		}},
		nil,
	)

	remaining := sm.GetStates()
	require.Len(t, remaining, 1, "Back の決定で設定メニューが Pop され1段になる")
	_, ok := remaining[0].(*states.MainMenuState)
	assert.True(t, ok, "残るのはメインメニュー")
}

// TestPlayScenario_captureが各ステップで呼ばれる は capture が nil でないとき、
// コマンド数ぶんのステップで0起点連番で呼ばれることを固定する。
func TestPlayScenario_captureが各ステップで呼ばれる(t *testing.T) {
	t.Parallel()
	steps := 0
	replay.PlayScenario(t,
		func(_ w.World) []es.State[w.World] {
			return []es.State[w.World]{&states.MainMenuState{}, &states.SettingsMenuState{}}
		},
		menuloop.Scenario{Commands: []menuloop.Command{
			inputmapper.ActionMenuDown,
			inputmapper.ActionMenuUp,
		}},
		func(step int, _ w.World, screen *ebiten.Image) {
			assert.Equal(t, steps, step, "step は0起点で連番")
			assert.NotNil(t, screen, "描画先が渡る")
			steps++
		},
	)
	assert.Equal(t, 2, steps, "コマンド数ぶん capture が呼ばれる")
}
