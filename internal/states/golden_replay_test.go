package states_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/mapplanner"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/vrt/replay"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

// replayStep はリプレイの1手。action を適用した直後のフレームを shot の golden 名で撮る。
// shot が空の手は撮らない
type replayStep struct {
	action inputmapper.ActionID
	shot   string
}

// TestGolden の静止画は state を組んだ直後の1枚を撮る。ここは操作した結果の画を撮る点が違う。
// 実在メニューを本番の MainGame ループで Action 列から駆動し、節目のフレームを golden で固定する。
// カーソルの移動先やメニューの開閉が壊れると、遷移のテストが通っても見た目で落ちる。
//
// 撮るのは静止画の golden では作れない画だけにする。既存 golden と同一の画を
// 別名で残しても、資産が増えて README のギャラリーが重複するだけで検出力は上がらない
func TestGoldenReplay(t *testing.T) {
	t.Parallel()

	// build は fixture の error をそのまま返し、サブテストで require する。
	// *testing.T を取らないことで thelper の誤検知を避ける。golden_test.go の build と同じ規約
	cases := []struct {
		name  string
		build func(world w.World) ([]es.State[w.World], error)
		steps []replayStep
	}{
		// 設定メニューのカーソルが Back へ移った画を撮る。開いた直後と閉じたあとの画は
		// TestGolden_SettingsMenu・TestGolden_MainMenu と完全に同一なので撮らない。
		// 閉じたあと1段になることは replay の遷移テストが押さえている
		{
			name: "SettingsMenuClose",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.MainMenuState{}, &gs.SettingsMenuState{}}, nil
			},
			steps: []replayStep{
				{action: inputmapper.ActionMenuDown, shot: "TestGolden_ReplaySettingsBack"}, // カーソルが Language から Back へ移った画
				{action: inputmapper.ActionMenuSelect},                                      // Back を決定して設定メニューを閉じる
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
			steps: []replayStep{
				{action: inputmapper.ActionMenuDown},                                                   // Start から Demo へ
				{action: inputmapper.ActionMenuDown},                                                   // Demo から Load へ
				{action: inputmapper.ActionMenuDown, shot: "TestGolden_ReplayMainMenuSettingsFocused"}, // Load から Settings へ
				{action: inputmapper.ActionMenuSelect},                                                 // Settings を開いて push する
			},
		},
		// ? で開くキー一覧ヘルプの描画を固定する。ヘルプの golden はこの1枚に絞る。
		// 文脈は最もキーが多いダンジョンにし、数字連結や記号の表記崩れまで一覧で検出する
		{
			name: "KeyHelp",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.DungeonState{
					Depth:          1,
					DefinitionName: dungeon.DungeonDebug.Name(),
					BuilderType:    mapplanner.PlannerTypeSmallRoom,
				}}, nil
			},
			steps: []replayStep{
				{action: inputmapper.ActionOpenKeyHelp, shot: "TestGolden_KeyHelp"}, // キー一覧ヘルプを push した画
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
			steps: []replayStep{
				{action: inputmapper.ActionOpenItemDetail, shot: "TestGolden_ItemActionDetail"}, // 調べるタブ先頭アイテムの詳細モーダルを開いた画
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actions := make([]inputmapper.ActionID, len(tc.steps))
			for i, s := range tc.steps {
				actions[i] = s.action
			}
			replay.PlayScenario(t,
				func(world w.World) []es.State[w.World] {
					built, err := tc.build(world)
					require.NoError(t, err)
					return built
				},
				actions,
				func(frame int, _ w.World, screen *ebiten.Image) {
					// フレーム 0 は開始直後で、どの手もまだ適用されていないので撮らない。
					// フレーム i は steps[i-1] の適用直後にあたる
					if frame == 0 {
						return
					}
					if name := tc.steps[frame-1].shot; name != "" {
						vrt.AssertFrameGolden(t, name, screen)
					}
				},
			)
		})
	}
}
