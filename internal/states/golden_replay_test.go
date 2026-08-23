package states_test

import (
	"fmt"
	"image"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/vrt/replay"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"

	"github.com/stretchr/testify/require"
)

// replayStep はリプレイの1手。shot が真なら action を適用した直後のフレームを撮る。
// action のゼロ値は入力なしで1フレーム進める待ちの手で、state を組んだ直後の画は
// 待ち1手に shot を付けて撮る。golden 名はケース名から導出し、1ケースで複数撮るときは
// suffix で区別して TestGolden_<ケース名>_<suffix> になる。単発は suffix を空にする。
// 全タブを撮るときは ActionMenuTabNext を並べ、各手に suffix でタブ名を付ける
type replayStep struct {
	action inputmapper.ActionID
	shot   bool
	suffix string
}

// TestGolden はステートの実描画を本番の MainGame ループで駆動して固定する VRT をまとめて回す。
// 各ケースは build が返すステート列を組み、steps の Action 列で操作し、shot の手の直後の
// フレームを golden と比較する。組んだ直後の静止画も操作後の画も同じリプレイ機構で撮る。
// カーソルの移動先やメニューの開閉が壊れると、遷移のテストが通っても見た目で落ちる。
//
// 撮るのは互いに異なる画だけにする。既存 golden と同一の画を
// 別名で残しても、資産が増えて README のギャラリーが重複するだけで検出力は上がらない
func TestGolden(t *testing.T) {
	t.Parallel()

	// build は fixture の error をそのまま返し、サブテストで require する。
	// *testing.T を取らないことで thelper の誤検知を避ける
	cases := []struct {
		name  string
		build func(world w.World) ([]es.State[w.World], error)
		steps []replayStep
	}{
		// ================ 組んだ直後の画を待ち1手で撮るケース ================
		{
			name: "MainMenu",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.MainMenuState{}}, nil
			},
			steps: []replayStep{{shot: true}},
		},
		{
			name: "SettingsMenu",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.MainMenuState{}, &gs.SettingsMenuState{}}, nil
			},
			steps: []replayStep{{shot: true}},
		},
		{
			name: "CharacterNaming",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.CharacterNamingState{}}, nil
			},
			steps: []replayStep{{shot: true}},
		},
		{
			name: "CharacterJob",
			build: func(w.World) ([]es.State[w.World], error) {
				s, err := gs.NewCharacterJobState("Ash")()
				return []es.State[w.World]{s}, err
			},
			steps: []replayStep{{shot: true}},
		},
		// OverworldMap は N キーで開く種別俯瞰図の描画を固定する。記号の色表・凡例・現在地マーカー・
		// 荒れ地の文字非重畳を含む描画経路を覆う。実際のプレイどおり下段の世界の上に地図UIを重ねて撮る。
		{
			name: "OverworldMap",
			build: func(w.World) ([]es.State[w.World], error) {
				// 背景は開始チャンクのオーバーワールド。決定的な RunSeed で golden を安定させる
				backdrop, err := gs.NewOverworldState(mapplanner.PlannerTypeOverworldField,
					dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1), &overworld.NewGameParams{RunSeed: 42})()
				if err != nil {
					return nil, err
				}
				return []es.State[w.World]{backdrop, &gs.OverworldMapState{}}, nil
			},
			steps: []replayStep{{shot: true}},
		},
		// ItemAction は動詞タブ画面の全タブを撮る。動詞ごとにバックパックのアイテムが
		// 適用可否で絞られる。回復薬は調べる・置く・食べる・使う・出品に出て、読むには出ない
		{
			name: "ItemAction",
			build: func(world w.World) ([]es.State[w.World], error) {
				if _, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3); err != nil {
					return nil, err
				}
				return []es.State[w.World]{&gs.ItemActionState{}}, nil
			},
			steps: []replayStep{
				{shot: true, suffix: "Inspect"}, // 初期タブ
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Drop"},
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Eat"},
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Read"},
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Use"},
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "List"},
			},
		},
		// Character は人物画面の全タブを撮る。装備は編集タブ、以降は読み取り専用の情報タブ。
		// スキルはページ送りとカテゴリ見出しを含む
		{
			name: "Character",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.CharacterState{}}, nil
			},
			steps: []replayStep{
				{shot: true, suffix: "Equipment"}, // 初期タブ
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Abilities"},
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Skills"},
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Effects"},
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Health"},
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Basic"},
			},
		},
		// RunStats は統計テーブル画面を撮る。character と同じテーブル枠に統計が並ぶ。
		// 死の結果画面 NewRunResultState は見出しが違うだけの同じテーブルなので撮り分けない
		{
			name: "RunStats",
			build: func(world w.World) ([]es.State[w.World], error) {
				// 見栄えのする統計値を仕込む。経過1240ターンで日数が2日目に乗る
				if s := query.GetRunStats(world); s != nil {
					s.EnemiesKilled = 12
					s.ItemsScavenged = 8
					s.SalesTotal = 3400
				}
				if gt := query.GetGameTime(world); gt != nil {
					gt.TotalTurns = 1240
				}
				st, err := gs.NewRunStatsState()
				if err != nil {
					return nil, err
				}
				return []es.State[w.World]{st}, nil
			},
			steps: []replayStep{
				{shot: true, suffix: "Table"},
			},
		},
		{
			name: "CraftMenu",
			build: func(world w.World) ([]es.State[w.World], error) {
				// 回復薬の材料を持たせ、合成可能な行にチェックが付く様子を確認する
				if _, err := lifecycle.SpawnBackpackItem(world, "green_herb", 1); err != nil {
					return nil, err
				}
				if _, err := lifecycle.SpawnBackpackItem(world, "yellow_herb", 1); err != nil {
					return nil, err
				}
				return []es.State[w.World]{&gs.CraftMenuState{}}, nil
			},
			steps: []replayStep{
				{shot: true, suffix: "Consumables"}, // 初期タブ
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Weapons"},
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Armor"},
			},
		},
		{
			name: "ShopMenu",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.ShopMenuState{}}, nil
			},
			steps: []replayStep{
				{shot: true, suffix: "Buy"}, // 初期タブ
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Sell"},
			},
		},
		{
			name: "SaveMenu",
			build: func(w.World) ([]es.State[w.World], error) {
				s, err := gs.NewSaveMenuState()
				return []es.State[w.World]{s}, err
			},
			steps: []replayStep{{shot: true}},
		},
		{
			name: "LoadMenu",
			build: func(w.World) ([]es.State[w.World], error) {
				s, err := gs.NewLoadMenuState()
				return []es.State[w.World]{s}, err
			},
			steps: []replayStep{{shot: true}},
		},
		{
			name: "DebugMenu",
			build: func(w.World) ([]es.State[w.World], error) {
				s, err := gs.NewDebugMenuState()
				return []es.State[w.World]{s}, err
			},
			steps: []replayStep{{shot: true}},
		},
		{
			name: "ComponentDebug",
			build: func(w.World) ([]es.State[w.World], error) {
				s, err := gs.NewComponentDebugState()
				return []es.State[w.World]{s}, err
			},
			steps: []replayStep{{shot: true}},
		},
		// CubePanel はキューブ内部のコントロールパネルの描画を固定する。
		// 現ステージを内部にし重量物を1つ置いて、総重量が出る状態でパネルを描く。
		{
			name: "CubePanel",
			build: func(world w.World) ([]es.State[w.World], error) {
				// 内部を現ステージにする。パネルの OnStart はここから総重量を算出する
				query.GetDungeon(world).CurrentStage = gc.NewCubeInteriorStage()
				// 内部の床へ重量物を1つ置き、総重量が非ゼロで出るようにする
				item := world.ECS.NewEntity()
				world.Components.Weight.Add(item, &gc.Weight{Milligram: 5 * consts.MilligramPerKg})
				world.Components.LocationOnField.Add(item, &gc.LocationOnField{})
				world.Components.StageBound.Add(item, &gc.StageBound{Key: gc.NewCubeInteriorStage()})
				return []es.State[w.World]{&gs.CubePanelState{}}, nil
			},
			steps: []replayStep{{shot: true}},
		},
		// LookAround は実際のプレイどおり、3D世界とHUDの上にカーソルと情報パネルを重ねて撮る。
		// カーソルを足元から離した画も撮る。プレイヤーの真下では投影のずれが最小になり、
		// 離した位置でこそカーソル枠が実際のタイルに乗っているかが分かる
		{
			name: "LookAround",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.DungeonState{
					Depth:          1,
					DefinitionName: dungeon.DungeonDebug.Name(),
					BuilderType:    mapplanner.PlannerTypeSmallRoom,
				}, &gs.LookAroundState{}}, nil
			},
			steps: []replayStep{
				{shot: true},
				{action: inputmapper.ActionMoveNorth},
				{action: inputmapper.ActionMoveNorth},
				{action: inputmapper.ActionMoveEast},
				{action: inputmapper.ActionMoveEast, shot: true, suffix: "Away"},
			},
		},
		{
			name: "GameOver",
			build: func(w.World) ([]es.State[w.World], error) {
				s, err := gs.NewGameOverMessageState()
				return []es.State[w.World]{s}, err
			},
			steps: []replayStep{{shot: true}},
		},
		{
			name: "Message",
			build: func(w.World) ([]es.State[w.World], error) {
				messageData := messagedata.NewDialogMessage(
					"This is a VRT test of the message window.\n\nA message to check the display state.",
					"VRT Test",
				).WithChoice(
					"Choice 1", func(_ w.World) error { return nil },
				).WithChoice(
					"Choice 2", func(_ w.World) error { return nil },
				)
				s, err := gs.NewMessageState(messageData)
				return []es.State[w.World]{s}, err
			},
			steps: []replayStep{{shot: true}},
		},
		// Shooting は実際のプレイどおり、3D世界とHUDの上に照準と射撃パネルを重ねて撮る。
		{
			name: "Shooting",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.DungeonState{
					Depth:          1,
					DefinitionName: dungeon.DungeonDebug.Name(),
					BuilderType:    mapplanner.PlannerTypeSmallRoom,
				}, &gs.ShootingState{}}, nil
			},
			steps: []replayStep{{shot: true}},
		},
		{
			name: "PersistentMessage",
			build: func(w.World) ([]es.State[w.World], error) {
				messageData := messagedata.NewDialogMessage("This is a VRT test of the persistent message.", "Test")
				return []es.State[w.World]{gs.NewPersistentMessageState(messageData)}, nil
			},
			steps: []replayStep{{shot: true}},
		},
		{
			name: "StorageMenu",
			build: func(world w.World) ([]es.State[w.World], error) {
				storageEntity, err := lifecycle.SpawnProp(world, "wooden_crate", 3, 3)
				if err != nil {
					return nil, err
				}
				if _, err := lifecycle.SpawnStorageItem(world, "healing_potion", 1, storageEntity); err != nil {
					return nil, err
				}
				s, err := gs.NewStorageMenuState(storageEntity)
				return []es.State[w.World]{s}, err
			},
			steps: []replayStep{
				{shot: true, suffix: "Retrieve"}, // 初期タブ
				{action: inputmapper.ActionMenuTabNext, shot: true, suffix: "Store"},
			},
		},
		// ChoiceMenuMany は共通の選択メニューが多数の選択肢でもモーダルに収まりページ送りすることを覆う。
		{
			name: "ChoiceMenuMany",
			build: func(w.World) ([]es.State[w.World], error) {
				choices := make([]gs.Choice, 0, 30)
				for i := range 30 {
					choices = append(choices, gs.Choice{Label: fmt.Sprintf("Item %d", i+1)})
				}
				menu := gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) { return "Select", choices })
				return []es.State[w.World]{menu}, nil
			},
			steps: []replayStep{{shot: true}},
		},
		// ChoiceMenuHeaders は共通の選択メニューの見出し行とページ表示なしの短い一覧を覆う。
		{
			name: "ChoiceMenuHeaders",
			build: func(w.World) ([]es.State[w.World], error) {
				choices := []gs.Choice{
					{Label: "Weapon", Header: true},
					{Label: "Wooden Sword"},
					{Label: "Ray Gun"},
					{Label: "Armor", Header: true},
					{Label: "Leather Armor"},
					{Label: "Back"},
				}
				menu := gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) { return "Equipment", choices })
				return []es.State[w.World]{menu}, nil
			},
			steps: []replayStep{{shot: true}},
		},
		// Overworld は新規ゲーム開始直後のオーバーワールド実画面を固定する。
		{
			name: "Overworld",
			build: func(w.World) ([]es.State[w.World], error) {
				s, err := gs.NewOverworldState(
					mapplanner.PlannerTypeOverworldField,
					dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1),
					&overworld.NewGameParams{},
				)()
				return []es.State[w.World]{s}, err
			},
			steps: []replayStep{{shot: true}},
		},
		// Dungeon は遺跡へ入った直後のダンジョン実画面を固定する。
		// プレイヤーは上り階段の上に湧く実スポーンのまま撮る。
		{
			name: "Dungeon",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.DungeonState{
					Depth:          1,
					DefinitionName: dungeon.DungeonDebug.Name(),
					BuilderType:    mapplanner.PlannerTypeSmallRoom,
				}}, nil
			},
			steps: []replayStep{{shot: true}},
		},
		// ================ 操作した結果の画を撮るケース ================
		// 設定メニューのカーソルが Back へ移った画を撮る。開いた直後と閉じたあとの画は
		// SettingsMenu・MainMenu と完全に同一なので撮らない。
		// 閉じたあと1段になることは replay の遷移テストが押さえている
		{
			name: "SettingsMenuClose",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.MainMenuState{}, &gs.SettingsMenuState{}}, nil
			},
			steps: []replayStep{
				{action: inputmapper.ActionMenuDown, shot: true}, // カーソルが Language から Back へ移った画を撮る
				{action: inputmapper.ActionMenuSelect},           // Back を決定して設定メニューを閉じる
			},
		},
		// メインメニューでカーソルが Settings に載った画を撮る。組んだ直後の画は先頭行に
		// カーソルがある状態しか撮れないので、移動後の見た目はここでしか固定できない。
		// push した先の画は SettingsMenu と同一なので撮らない
		{
			name: "MainMenuOpenSettings",
			build: func(w.World) ([]es.State[w.World], error) {
				return []es.State[w.World]{&gs.MainMenuState{}}, nil
			},
			steps: []replayStep{
				{action: inputmapper.ActionMenuDown},             // Start から Demo へ
				{action: inputmapper.ActionMenuDown},             // Demo から Load へ
				{action: inputmapper.ActionMenuDown, shot: true}, // Load から Settings へ。移った画を撮る
				{action: inputmapper.ActionMenuSelect},           // Settings を開いて push する
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
				{action: inputmapper.ActionOpenKeyHelp}, // ヘルプを push する。反映は次フレーム
				{shot: true},                            // 押し込まれたヘルプを撮る
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
				{action: inputmapper.ActionOpenItemDetail, shot: true}, // 調べるタブ先頭アイテムの詳細モーダルを開いた画を撮る
			},
		},
		// 装備選択ポップアップが実プレイ経路で開く state 遷移を固定する。装備タブで空の武器
		// スロットを選ぶと開く。武器スロット1は初期装備の松明で埋まるので1つ下げて空のスロット2で開く。
		// ポップアップの視覚の詳細は widget golden TestGolden_EquipSelect が厳密に固定する
		{
			name: "EquipSelectFlow",
			build: func(world w.World) ([]es.State[w.World], error) {
				if _, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1); err != nil {
					return nil, err
				}
				if _, err := lifecycle.SpawnBackpackItem(world, "ray_gun", 1); err != nil {
					return nil, err
				}
				return []es.State[w.World]{&gs.CharacterState{}}, nil
			},
			steps: []replayStep{
				{action: inputmapper.ActionMenuDown},               // 武器スロット1から空のスロット2へ
				{action: inputmapper.ActionMenuSelect, shot: true}, // 空スロットで装備選択を開いた画を撮る
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 各撮影の golden 名をケース名と suffix から先に確定し、重複を弾く。
			// 複数撮るケースは suffix で区別する
			seen := map[string]bool{}
			actions := make([]inputmapper.ActionID, len(tc.steps))
			for i, s := range tc.steps {
				actions[i] = s.action
				if !s.shot {
					continue
				}
				name := "TestGolden_" + tc.name
				if s.suffix != "" {
					name += "_" + s.suffix
				}
				require.False(t, seen[name], "golden 名 %s が重複している。複数撮るなら suffix で区別する", name)
				seen[name] = true
			}
			replay.PlayScenario(t,
				func(world w.World) []es.State[w.World] {
					built, err := tc.build(world)
					require.NoError(t, err)
					return built
				},
				actions,
				func(frame int, _ w.World, img *image.NRGBA) {
					// フレーム f は steps[f] の action を適用した直後の画。カーソル移動や
					// タブ送りは同一フレームで効くのでその手で撮る。state を push する手は
					// 次フレームで反映されるので、末尾に待ち手を足してそこで撮る。
					// 末尾の settle フレームには対応する step が無いので撮らない
					if frame >= len(tc.steps) || !tc.steps[frame].shot {
						return
					}
					name := "TestGolden_" + tc.name
					if suffix := tc.steps[frame].suffix; suffix != "" {
						name += "_" + suffix
					}
					vrt.AssertFrameGolden(t, name, img)
				},
			)
		})
	}
}
