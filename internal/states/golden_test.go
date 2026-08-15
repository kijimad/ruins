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

func TestGolden_MainMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.MainMenuState{}))
}

func TestGolden_SettingsMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.MainMenuState{}, &gs.SettingsMenuState{}))
}

func TestGolden_LanguageMenu(t *testing.T) {
	t.Parallel()
	s, err := gs.NewLanguageMenuState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(&gs.MainMenuState{}, s))
}

func TestGolden_CharacterNaming(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.CharacterNamingState{}))
}

func TestGolden_CharacterJob(t *testing.T) {
	t.Parallel()
	s, err := gs.NewCharacterJobState("Ash")()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(s))
}

// newGoldenBackdrop はメニュー系 golden の背景に使うオーバーワールド状態を作る。
// 街がオーバーワールドの地物になり専用の街ステートが無くなったため、旧 NewTownState の
// 代わりに開始チャンクを背景として使う。決定的な RunSeed で golden を安定させる。
func newGoldenBackdrop(t *testing.T) es.State[w.World] {
	t.Helper()
	s, err := gs.NewOverworldState(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1), &overworld.NewGameParams{RunSeed: 42})()
	require.NoError(t, err)
	return s
}

// TestGolden_OverworldMap は N キーで開く種別俯瞰図の描画を固定する。記号の色表・凡例・
// 現在地マーカー・荒れ地の文字非重畳を含む描画経路を覆い、配色や記号の集約を変えたときの
// 退行を捕らえる。俯瞰図は帯から純関数で算出するので、決定的 RunSeed で golden が安定する。
func TestGolden_OverworldMap(t *testing.T) {
	t.Parallel()
	// 下段の帯を映さず記号地図UIだけを撮る。backdrop は band データの供給元として積む
	backdrop := newGoldenBackdrop(t)
	vrt.AssertTopStateGolden(t, vrt.States(backdrop, &gs.OverworldMapState{}))
}

// TestGolden_ItemAction は動詞タブ画面を固定する。調べるタブでバックパックの
// アイテムを名前のみで一覧する経路を覆う。
func TestGolden_ItemAction(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
		_, err := lifecycle.SpawnBackpackItem(world, "healing_potion", 3)
		require.NoError(t, err)
		return []es.State[w.World]{&gs.ItemActionState{}}
	})
}

// TestGolden_Character は画面タブメニューを固定する。装備タブでプレイヤーの
// スロット一覧を1カラムで並べる経路を覆う。
func TestGolden_Character(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.CharacterState{}))
}

func TestGolden_CraftMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
		// 回復薬の材料を持たせ、合成可能な行にチェックが付く様子を確認する
		_, err := lifecycle.SpawnBackpackItem(world, "green_herb", 1)
		require.NoError(t, err)
		_, err = lifecycle.SpawnBackpackItem(world, "yellow_herb", 1)
		require.NoError(t, err)
		return []es.State[w.World]{&gs.CraftMenuState{}}
	})
}

func TestGolden_ShopMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.ShopMenuState{}))
}

func TestGolden_SaveMenu(t *testing.T) {
	t.Parallel()
	s, err := gs.NewSaveMenuState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(s))
}

func TestGolden_LoadMenu(t *testing.T) {
	t.Parallel()
	s, err := gs.NewLoadMenuState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(s))
}

func TestGolden_DebugMenu(t *testing.T) {
	t.Parallel()
	s, err := gs.NewDebugMenuState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(s))
}

func TestGolden_ComponentDebug(t *testing.T) {
	t.Parallel()
	s, err := gs.NewComponentDebugState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(s))
}

// ダンジョンとオーバーワールドのワールド描画は3D命令列VRTへ移した。
// render3d_golden_test.go の TestGolden_Render3DSnapshot / TestRender3DImages を参照する。

// TestGolden_CubePanel はキューブ内部のコントロールパネルの描画を固定する。
// 現ステージを内部にし重量物を1つ置いて、総重量が出る状態でパネルを描く。
func TestGolden_CubePanel(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
		// 内部を現ステージにする。パネルの OnStart はここから総重量を算出する
		query.GetDungeon(world).CurrentStage = gc.NewCubeInteriorStage()
		// 内部の床へ重量物を1つ置き、総重量が非ゼロで出るようにする
		item := world.ECS.NewEntity()
		world.Components.Weight.Add(item, &gc.Weight{Milligram: 5 * consts.MilligramPerKg})
		world.Components.LocationOnField.Add(item, &gc.LocationOnField{})
		world.Components.StageBound.Add(item, &gc.StageBound{Key: gc.NewCubeInteriorStage()})
		return []es.State[w.World]{&gs.CubePanelState{}}
	})
}

// 寒波前線の氷オーバーレイも3D命令列VRTへ移した。render3d_golden_test.go の OverworldFrost シーンを参照する。

func TestGolden_LookAround(t *testing.T) {
	t.Parallel()
	// ワールドは映さずカーソルと情報パネルだけを撮る。DungeonState はタイルデータの供給元として積む
	vrt.AssertTopStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:          1,
		DefinitionName: dungeon.DungeonDebug.Name(),
		BuilderType:    mapplanner.PlannerTypeSmallRoom,
	}, &gs.LookAroundState{}))
}

func TestGolden_GameOver(t *testing.T) {
	t.Parallel()
	s, err := gs.NewGameOverMessageState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(s))
}

func TestGolden_Message(t *testing.T) {
	t.Parallel()
	messageData := messagedata.NewDialogMessage(
		"これはメッセージウィンドウのVRTテストです。\n\n表示状態の確認用メッセージです。",
		"VRTテスト",
	).WithChoice(
		"選択肢1", func(_ w.World) error { return nil },
	).WithChoice(
		"選択肢2", func(_ w.World) error { return nil },
	)
	msgState, err := gs.NewMessageState(messageData)
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(msgState))
}

func TestGolden_Shooting(t *testing.T) {
	t.Parallel()
	// ワールドは映さず照準と射撃パネルだけを撮る。DungeonState は敵データの供給元として積む
	vrt.AssertTopStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:          1,
		DefinitionName: dungeon.DungeonDebug.Name(),
		BuilderType:    mapplanner.PlannerTypeSmallRoom,
	}, &gs.ShootingState{}))
}

func TestGolden_PersistentMessage(t *testing.T) {
	t.Parallel()
	messageData := messagedata.NewDialogMessage(
		"永続メッセージのVRTテストです。",
		"テスト",
	)
	vrt.AssertStateGolden(t, vrt.States(gs.NewPersistentMessageState(messageData)))
}

func TestGolden_StorageMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
		storageEntity, err := lifecycle.SpawnProp(world, "wooden_crate", 3, 3)
		require.NoError(t, err)

		_, err = lifecycle.SpawnStorageItem(world, "healing_potion", 1, storageEntity)
		require.NoError(t, err)

		storageState, stateErr := gs.NewStorageMenuState(storageEntity)
		require.NoError(t, stateErr)

		return []es.State[w.World]{
			storageState,
		}
	})
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

				pngData := vrt.RenderStatePNG(t, vrt.States(&gs.MapGenVisualizerState{
					PlannerType:   pt,
					Seed:          result.seed,
					SnapshotIndex: i,
				}))
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

// TestGolden_ChoiceMenuMany は共通の選択メニューが多数の選択肢でもモーダルに収まりページ送りすることを覆う。
// 各メニュー個別でなく共通実装 ChoiceMenu を一度だけ検証する
func TestGolden_ChoiceMenuMany(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(_ w.World) []es.State[w.World] {
		choices := make([]gs.Choice, 0, 30)
		for i := range 30 {
			choices = append(choices, gs.Choice{Label: fmt.Sprintf("項目 %d", i+1)})
		}
		menu := gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) { return "選択", choices })
		return []es.State[w.World]{menu}
	})
}

// TestGolden_ChoiceMenuHeaders は共通の選択メニューの見出し行とページ表示なしの短い一覧を覆う
func TestGolden_ChoiceMenuHeaders(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(_ w.World) []es.State[w.World] {
		choices := []gs.Choice{
			{Label: "武器", Header: true},
			{Label: "木刀"},
			{Label: "レイガン"},
			{Label: "防具", Header: true},
			{Label: "革の鎧"},
			{Label: "戻る"},
		}
		menu := gs.NewChoiceMenu(func(_ w.World) (string, []gs.Choice) { return "ロード", choices })
		return []es.State[w.World]{menu}
	})
}
