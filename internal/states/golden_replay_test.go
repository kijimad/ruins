package states_test

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/vrt/replay"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

// TestGolden の静止画は state を組んだ直後の1枚を撮る。ここは操作した結果の画を撮る点が違う。
// 実在メニューを本番の MainGame ループで Action 列から駆動し、節目のフレームを golden で固定する。
// カーソルの移動先やメニューの開閉が壊れると、遷移のテストが通っても見た目で落ちる。
//
// shots はフレーム番号ごとの golden 名で、空文字のフレームは撮らない。Action 数+1 フレーム回るので
// 長さもそれに合わせる。全フレーム残すと差分が読みにくく資産も増えるため、節目だけ名前を付ける。
func TestGoldenReplay(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		build   func(world w.World) []es.State[w.World]
		actions []inputmapper.ActionID
		shots   []string
	}{
		// 設定メニューをカーソル移動して閉じるまでを撮る。開いた直後は TestGolden_SettingsMenu が
		// 押さえているので、カーソルが Back へ移った画と、閉じてメインメニューへ戻った画を残す
		{
			name: "SettingsMenuClose",
			build: func(w.World) []es.State[w.World] {
				return []es.State[w.World]{&gs.MainMenuState{}, &gs.SettingsMenuState{}}
			},
			actions: []inputmapper.ActionID{
				inputmapper.ActionMenuDown,   // カーソルを Language から Back へ移す
				inputmapper.ActionMenuSelect, // Back を決定して設定メニューを閉じる
			},
			shots: []string{
				"",
				"TestGolden_ReplaySettingsBack",
				"TestGolden_ReplaySettingsClosed",
			},
		},
		// メインメニューから設定メニューを開くまでを撮る。push を跨いでも同じ列で駆動できることを
		// 見た目でも固定する。カーソルが Settings に載った画と、開いた直後の画を残す
		{
			name: "MainMenuOpenSettings",
			build: func(w.World) []es.State[w.World] {
				return []es.State[w.World]{&gs.MainMenuState{}}
			},
			actions: []inputmapper.ActionID{
				inputmapper.ActionMenuDown,   // Start から Demo へ
				inputmapper.ActionMenuDown,   // Demo から Load へ
				inputmapper.ActionMenuDown,   // Load から Settings へ
				inputmapper.ActionMenuSelect, // Settings を開いて push する
			},
			shots: []string{
				"",
				"",
				"",
				"TestGolden_ReplayMainMenuSettingsFocused",
				"TestGolden_ReplayMainMenuSettingsOpened",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Len(t, tc.shots, len(tc.actions)+1, "shots は Action 数+1 フレームぶん要る")

			replay.PlayScenario(t, tc.build, tc.actions,
				func(frame int, _ w.World, screen *ebiten.Image) {
					name := tc.shots[frame]
					if name == "" {
						// 撮らないフレームの画は使わない。AssertFrameGolden と同じく解放しておく
						screen.Deallocate()
						return
					}
					vrt.AssertFrameGolden(t, name, screen)
				},
			)
		})
	}
}
