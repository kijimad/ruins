package states_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

// newGoldenBackdrop はメニュー系 golden の背景に使うオーバーワールド状態を作る。
// 街がオーバーワールドの地物になり専用の街ステートが無くなったため、旧 NewTownState の
// 代わりに開始チャンクを背景として使う。決定的な RunSeed で golden を安定させる。
func newGoldenBackdrop() (es.State[w.World], error) {
	return gs.NewOverworldState(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1), &overworld.NewGameParams{RunSeed: 42})()
}

// TestGolden はステートの実描画をフルスタックで固定する VRT をまとめて回す。
// 各ケースは build が返すステート列を実際のプレイどおり描いて golden と比較する。
// ゴールデン名は t.Name() のスラッシュを均すため testdata/TestGolden_<name>.png になる。
func TestGolden(t *testing.T) {
	t.Parallel()

	// build は描画するステート列を組む。fixture の error はそのまま返し、ループで require する。
	// *testing.T を取らないことで thelper の誤検知を避ける。
	cases := []struct {
		name  string
		build func(world w.World) ([]es.State[w.World], error)
	}{
		{"MainMenu", func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.MainMenuState{}}, nil
		}},
		{"SettingsMenu", func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.MainMenuState{}, &gs.SettingsMenuState{}}, nil
		}},
		{"LanguageMenu", func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewLanguageMenuState()
			return []es.State[w.World]{&gs.MainMenuState{}, s}, err
		}},
		{"CharacterNaming", func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.CharacterNamingState{}}, nil
		}},
		{"CharacterJob", func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewCharacterJobState("Ash")()
			return []es.State[w.World]{s}, err
		}},
		// OverworldMap は N キーで開く種別俯瞰図の描画を固定する。記号の色表・凡例・現在地マーカー・
		// 荒れ地の文字非重畳を含む描画経路を覆う。実際のプレイどおり下段の世界の上に地図UIを重ねて撮る。
		{"OverworldMap", func(w.World) ([]es.State[w.World], error) {
			backdrop, err := newGoldenBackdrop()
			if err != nil {
				return nil, err
			}
			return []es.State[w.World]{backdrop, &gs.OverworldMapState{}}, nil
		}},
		// ItemAction は動詞タブ画面を固定する。調べるタブでバックパックのアイテムを名前のみで一覧する経路を覆う。
		{"ItemAction", func(world w.World) ([]es.State[w.World], error) {
			if _, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3); err != nil {
				return nil, err
			}
			return []es.State[w.World]{&gs.ItemActionState{}}, nil
		}},
		// Character は画面タブメニューを固定する。装備タブでプレイヤーのスロット一覧を1カラムで並べる経路を覆う。
		{"Character", func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.CharacterState{}}, nil
		}},
		{"CraftMenu", func(world w.World) ([]es.State[w.World], error) {
			// 回復薬の材料を持たせ、合成可能な行にチェックが付く様子を確認する
			if _, err := lifecycle.SpawnBackpackItem(world, "green_herb", 1); err != nil {
				return nil, err
			}
			if _, err := lifecycle.SpawnBackpackItem(world, "yellow_herb", 1); err != nil {
				return nil, err
			}
			return []es.State[w.World]{&gs.CraftMenuState{}}, nil
		}},
		{"ShopMenu", func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.ShopMenuState{}}, nil
		}},
		{"SaveMenu", func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewSaveMenuState()
			return []es.State[w.World]{s}, err
		}},
		{"LoadMenu", func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewLoadMenuState()
			return []es.State[w.World]{s}, err
		}},
		{"DebugMenu", func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewDebugMenuState()
			return []es.State[w.World]{s}, err
		}},
		{"ComponentDebug", func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewComponentDebugState()
			return []es.State[w.World]{s}, err
		}},
		// CubePanel はキューブ内部のコントロールパネルの描画を固定する。
		// 現ステージを内部にし重量物を1つ置いて、総重量が出る状態でパネルを描く。
		{"CubePanel", func(world w.World) ([]es.State[w.World], error) {
			// 内部を現ステージにする。パネルの OnStart はここから総重量を算出する
			query.GetDungeon(world).CurrentStage = gc.NewCubeInteriorStage()
			// 内部の床へ重量物を1つ置き、総重量が非ゼロで出るようにする
			item := world.ECS.NewEntity()
			world.Components.Weight.Add(item, &gc.Weight{Milligram: 5 * consts.MilligramPerKg})
			world.Components.LocationOnField.Add(item, &gc.LocationOnField{})
			world.Components.StageBound.Add(item, &gc.StageBound{Key: gc.NewCubeInteriorStage()})
			return []es.State[w.World]{&gs.CubePanelState{}}, nil
		}},
		// LookAround は実際のプレイどおり、3D世界とHUDの上にカーソルと情報パネルを重ねて撮る。
		{"LookAround", func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.DungeonState{
				Depth:          1,
				DefinitionName: dungeon.DungeonDebug.Name(),
				BuilderType:    mapplanner.PlannerTypeSmallRoom,
			}, &gs.LookAroundState{}}, nil
		}},
		{"GameOver", func(w.World) ([]es.State[w.World], error) {
			s, err := gs.NewGameOverMessageState()
			return []es.State[w.World]{s}, err
		}},
		{"Message", func(w.World) ([]es.State[w.World], error) {
			messageData := messagedata.NewDialogMessage(
				"これはメッセージウィンドウのVRTテストです。\n\n表示状態の確認用メッセージです。",
				"VRTテスト",
			).WithChoice(
				"選択肢1", func(_ w.World) error { return nil },
			).WithChoice(
				"選択肢2", func(_ w.World) error { return nil },
			)
			s, err := gs.NewMessageState(messageData)
			return []es.State[w.World]{s}, err
		}},
		// Shooting は実際のプレイどおり、3D世界とHUDの上に照準と射撃パネルを重ねて撮る。
		{"Shooting", func(w.World) ([]es.State[w.World], error) {
			return []es.State[w.World]{&gs.DungeonState{
				Depth:          1,
				DefinitionName: dungeon.DungeonDebug.Name(),
				BuilderType:    mapplanner.PlannerTypeSmallRoom,
			}, &gs.ShootingState{}}, nil
		}},
		{"PersistentMessage", func(w.World) ([]es.State[w.World], error) {
			messageData := messagedata.NewDialogMessage("永続メッセージのVRTテストです。", "テスト")
			return []es.State[w.World]{gs.NewPersistentMessageState(messageData)}, nil
		}},
		{"StorageMenu", func(world w.World) ([]es.State[w.World], error) {
			storageEntity, err := lifecycle.SpawnProp(world, "wooden_crate", 3, 3)
			if err != nil {
				return nil, err
			}
			if _, err := lifecycle.SpawnStorageItem(world, "healing_potion", 1, storageEntity); err != nil {
				return nil, err
			}
			s, err := gs.NewStorageMenuState(storageEntity)
			return []es.State[w.World]{s}, err
		}},
		// ChoiceMenuMany は共通の選択メニューが多数の選択肢でもモーダルに収まりページ送りすることを覆う。
		{"ChoiceMenuMany", func(w.World) ([]es.State[w.World], error) {
			choices := make([]gs.Choice, 0, 30)
			for i := range 30 {
				choices = append(choices, gs.Choice{Label: fmt.Sprintf("項目 %d", i+1)})
			}
			menu := gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) { return "選択", choices })
			return []es.State[w.World]{menu}, nil
		}},
		// ChoiceMenuHeaders は共通の選択メニューの見出し行とページ表示なしの短い一覧を覆う。
		{"ChoiceMenuHeaders", func(w.World) ([]es.State[w.World], error) {
			choices := []gs.Choice{
				{Label: "武器", Header: true},
				{Label: "木刀"},
				{Label: "レイガン"},
				{Label: "防具", Header: true},
				{Label: "革の鎧"},
				{Label: "戻る"},
			}
			menu := gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) { return "装備選択", choices })
			return []es.State[w.World]{menu}, nil
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
				states, err := tc.build(world)
				require.NoError(t, err)
				return states
			})
		})
	}
}

const mapGenSeed = uint64(12345)

// collectPlannerTypes は全PlannerTypeにダンジョン定義のテーブル名を設定して返す。
// ダンジョン定義に含まれるPlannerTypeにはそのダンジョンのテーブル名を設定し、
// 含まれないもの（テンプレート系など）はテーブルなしでテストする
func collectPlannerTypes() []mapplanner.PlannerType {
	// ダンジョン定義からPlannerType名→テーブル名のマッピングを構築する
	type tableInfo struct {
		EnemyTableName string
		ItemTableName  string
	}
	tableMap := map[string]tableInfo{}
	dungeons := dungeon.GetAllDungeons()
	for i := range dungeons {
		def := dungeons[i]
		for _, pw := range def.PlannerPool() {
			if _, exists := tableMap[pw.PlannerType.Name]; !exists {
				tableMap[pw.PlannerType.Name] = tableInfo{
					EnemyTableName: def.EnemyTableName(),
					ItemTableName:  def.ItemTableName(),
				}
			}
		}
	}

	result := make([]mapplanner.PlannerType, len(mapplanner.AllPlannerTypes))
	copy(result, mapplanner.AllPlannerTypes)
	for i := range result {
		if info, ok := tableMap[result[i].Name]; ok {
			result[i].EnemyTableName = info.EnemyTableName
			result[i].ItemTableName = info.ItemTableName
		}
		result[i].Depth = 1
	}
	return result
}

// mapGenResult はbuildMapGenChainの結果を保持する
type mapGenResult struct {
	chain *mapplanner.PlannerChain
	seed  uint64
}

// buildMapGenChain はBuildChainを使ってRecording付きチェーンを構築する。
// 実装と同じチェーン構築ロジックを共有する。
// 接続性エラー時は本番と同様にシードを変えてリトライする。
// アセット未作成などでチェーン構築に失敗した場合はスキップしてnilを返す
func buildMapGenChain(t *testing.T, pt mapplanner.PlannerType) *mapGenResult {
	t.Helper()
	world := vrt.InitVRTWorld(t)
	for attempt := range mapplanner.MaxPlanRetries {
		currentSeed := mapGenSeed + uint64(attempt*1000)
		chain, err := mapplanner.BuildChain(world, consts.MapTileWidth, consts.MapTileHeight, currentSeed, pt)
		if err != nil {
			t.Skipf("PlannerType %s のチェーン構築をスキップ: %v", pt.Name, err)
			return nil
		}
		chain.Recording = true
		if err := chain.Plan(); err != nil {
			if errors.Is(err, mapplanner.ErrConnectivity) {
				continue
			}
			t.Skipf("PlannerType %s のプラン生成をスキップ: %v", pt.Name, err)
			return nil
		}
		return &mapGenResult{chain: chain, seed: currentSeed}
	}
	t.Skipf("PlannerType %s のプラン生成が%d回失敗しました", pt.Name, mapplanner.MaxPlanRetries)
	return nil
}

// TestGolden_MapGenSnapshot は全PlannerTypeの全フェーズのスナップショットをJSONで検証する。
// テーブル名はダンジョン定義から取得する
func TestGolden_MapGenSnapshot(t *testing.T) {
	t.Parallel()

	for _, pt := range collectPlannerTypes() {
		result := buildMapGenChain(t, pt)
		if result == nil {
			continue
		}
		for i, snap := range result.chain.Snapshots {
			t.Run(fmt.Sprintf("%s/Phase%d_%s", pt.Name, i, snap.Label), func(t *testing.T) {
				t.Parallel()
				data, err := json.MarshalIndent(snap, "", "  ")
				require.NoError(t, err)

				g := goldie.New(t, goldie.WithNameSuffix(".json"))
				g.Assert(t, t.Name(), data)
			})
		}
	}
}

// TestGolden_FacilitySamples は施設種別ごとに建物候補を 3x3 で並べたギャラリーを描く。
// 生成＆選別パイプラインの段3(人間の採否)の視覚基盤。生成規則が変わると golden が変わるので
// updategolden で更新し、並んだ候補を見比べて良し悪しを判断する。フォグ無しで内装まで見える。
// TestMapGenImages は全PlannerTypeの各フェーズのVRT画像を生成する。
// 対応するスナップショットJSONの内容が変わった場合のみ画像を再生成する。
// ピクセル比較は行わず、目視確認用の参照画像として保存する
func TestMapGenImages(t *testing.T) {
	t.Parallel()

	for _, pt := range collectPlannerTypes() {
		result := buildMapGenChain(t, pt)
		if result == nil {
			continue
		}
		for i, snap := range result.chain.Snapshots {
			t.Run(fmt.Sprintf("%s/Phase%d_%s", pt.Name, i, snap.Label), func(t *testing.T) {
				t.Parallel()

				currentJSON, err := json.MarshalIndent(snap, "", "  ")
				require.NoError(t, err)

				g := goldie.New(t, goldie.WithNameSuffix(".png"))
				imgPath := g.GoldenFileName(t, t.Name())
				subName := strings.TrimPrefix(t.Name(), "TestMapGenImages/")
				jsonPath := filepath.Join("testdata", "TestGolden_MapGenSnapshot", subName+".json")

				if !imgNeedsUpdate(imgPath, jsonPath, currentJSON) {
					return
				}

				pngData := vrt.RenderPNG(t, vrt.States(&gs.MapGenVisualizerState{
					PlannerType:   pt,
					Seed:          result.seed,
					SnapshotIndex: i,
				}), nil)
				require.NoError(t, g.Update(t, t.Name(), pngData))
				t.Logf("画像を更新: %s", imgPath)
			})
		}
	}
}

// imgNeedsUpdate は画像が存在しないかJSONの内容が変わった場合にtrueを返す。
// goldie はJSONの末尾に改行を付加するためTrimSpaceで比較する
func imgNeedsUpdate(imgPath, jsonPath string, currentJSON []byte) bool {
	if _, err := os.Stat(imgPath); err != nil {
		return true
	}
	goldenJSON, err := os.ReadFile(jsonPath)
	if err != nil {
		return true
	}
	return !bytes.Equal(bytes.TrimSpace(currentJSON), bytes.TrimSpace(goldenJSON))
}
