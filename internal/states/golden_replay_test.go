package states_test

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/vrt/replay"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

// TestGolden の静止画は state を組んだ直後の1枚を撮る。ここは操作した結果の画を撮る点が違う。
// 実在メニューを本番の MainGame ループで Action 列から駆動し、節目のフレームを golden で固定する。
// カーソルの移動先やメニューの開閉が壊れると、遷移のテストが通っても見た目で落ちる。
//
// shots はフレーム番号ごとの golden 名で、空文字のフレームは撮らない。Action 数+1 フレーム回るので
// 長さもそれに合わせる。撮るのは静止画の golden では作れない画だけにする。既存 golden と同一の画を
// 別名で残しても、資産が増えて README のギャラリーが重複するだけで検出力は上がらない。
func TestGoldenReplay(t *testing.T) {
	t.Parallel()

	// build は fixture の error をそのまま返し、サブテストで require する。
	// *testing.T を取らないことで thelper の誤検知を避ける。golden_test.go の build と同じ規約
	cases := []struct {
		name    string
		build   func(world w.World) ([]es.State[w.World], error)
		actions []inputmapper.ActionID
		shots   []string
	}{
		// 設定メニューのカーソルが Back へ移った画を撮る。開いた直後と閉じたあとの画は
		// TestGolden_SettingsMenu・TestGolden_MainMenu と完全に同一なので撮らない。
		// 閉じたあと1段になることは replay の遷移テストが押さえている
		{
			name: "SettingsMenuClose",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.MainMenuState{}, &gs.SettingsMenuState{}}, nil
			},
			actions: []inputmapper.ActionID{
				inputmapper.ActionMenuDown,   // カーソルを Language から Back へ移す
				inputmapper.ActionMenuSelect, // Back を決定して設定メニューを閉じる
			},
			shots: []string{
				"",
				"TestGolden_ReplaySettingsBack",
				"",
			},
		},
		// メインメニューでカーソルが Settings に載った画を撮る。静止画の golden は先頭行に
		// カーソルがある状態しか撮れないので、移動後の見た目はここでしか固定できない。
		// push した先の画は TestGolden_SettingsMenu と同一なので撮らない
		{
			name: "MainMenuOpenSettings",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.MainMenuState{}}, nil
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
				"",
			},
		},
		// ? で開くキー一覧ヘルプの描画を固定する。動詞タブ画面の束縛表と共通表から
		// 一覧が導出されることを覆う。Screen の入力ゲートが OpenKeyHelp を吸って push する経路で撮る
		{
			name: "KeyHelp",
			build: func(world w.World) ([]es.State[w.World], error) {
				if _, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3); err != nil {
					return nil, err
				}
				return []es.State[w.World]{&gs.ItemActionState{}}, nil
			},
			actions: []inputmapper.ActionID{
				inputmapper.ActionOpenKeyHelp, // キー一覧ヘルプを push する
			},
			shots: []string{
				"",
				"TestGolden_KeyHelp",
			},
		},
		// x で開く詳細モーダルの描画を固定する。個数とタイトルバーが無く、性能・性質と説明が
		// 並ぶことを覆う。入力ゲートと overlay 重ねを含む本番経路で撮る
		{
			name: "ItemActionDetail",
			build: func(world w.World) ([]es.State[w.World], error) {
				if _, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3); err != nil {
					return nil, err
				}
				return []es.State[w.World]{&gs.ItemActionState{}}, nil
			},
			actions: []inputmapper.ActionID{
				inputmapper.ActionOpenItemDetail, // 調べるタブ先頭アイテムの詳細モーダルを開く
			},
			shots: []string{
				"",
				"TestGolden_ItemActionDetail",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Len(t, tc.shots, len(tc.actions)+1, "shots は Action 数+1 フレームぶん要る")

			replay.PlayScenario(t,
				func(world w.World) []es.State[w.World] {
					built, err := tc.build(world)
					require.NoError(t, err)
					return built
				},
				tc.actions,
				func(frame int, _ w.World, screen *ebiten.Image) {
					if name := tc.shots[frame]; name != "" {
						vrt.AssertFrameGolden(t, name, screen)
					}
				},
			)
		})
	}
}
