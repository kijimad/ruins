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

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
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

func TestGolden_CharacterNaming(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.CharacterNamingState{}))
}

func TestGolden_CharacterJob(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewCharacterJobState("Ash")()))
}

func TestGolden_Town(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()()))
}

func TestGolden_InventoryMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), &gs.InventoryMenuState{}))
}

func TestGolden_EquipMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), &gs.EquipMenuState{}))
}

func TestGolden_CraftMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), &gs.CraftMenuState{}))
}

func TestGolden_ShopMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), &gs.ShopMenuState{}))
}

func TestGolden_AutoSell(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), gs.NewAutoSellState()()))
}

func TestGolden_SaveMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), gs.NewSaveMenuState()))
}

func TestGolden_LoadMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), gs.NewLoadMenuState()))
}

func TestGolden_DebugMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), gs.NewDebugMenuState()))
}

func TestGolden_ComponentDebug(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), gs.NewComponentDebugState()))
}

func TestGolden_DungeonSelect(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), gs.NewDungeonSelectState()))
}

func TestGolden_Dungeon(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:       1,
		BuilderType: mapplanner.PlannerTypeSmallRoom,
	}))
}

func TestGolden_LookAround(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:       1,
		BuilderType: mapplanner.PlannerTypeSmallRoom,
	}, &gs.LookAroundState{}))
}

func TestGolden_GameOver(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), gs.NewGameOverMessageState()))
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
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), gs.NewMessageState(messageData)))
}

func TestGolden_Status(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), gs.NewStatusState()))
}

func TestGolden_Shooting(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:       1,
		BuilderType: mapplanner.PlannerTypeSmallRoom,
	}, &gs.ShootingState{}))
}

func TestGolden_Pickup(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:       1,
		BuilderType: mapplanner.PlannerTypeSmallRoom,
	}, &gs.PickupState{}))
}

func TestGolden_Place(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:       1,
		BuilderType: mapplanner.PlannerTypeSmallRoom,
	}, &gs.PlaceState{}))
}

func TestGolden_PersistentMessage(t *testing.T) {
	t.Parallel()
	messageData := messagedata.NewDialogMessage(
		"永続メッセージのVRTテストです。",
		"テスト",
	)
	vrt.AssertStateGolden(t, vrt.States(gs.NewTownState()(), gs.NewPersistentMessageState(messageData)))
}

func TestGolden_StorageMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
		storageEntity, err := worldhelper.SpawnProp(world, "木箱", 3, 3)
		require.NoError(t, err)

		_, err = worldhelper.SpawnStorageItem(world, "回復薬", 1, storageEntity)
		require.NoError(t, err)

		return []es.State[w.World]{
			gs.NewTownState()(),
			gs.NewStorageMenuState(storageEntity),
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
	for _, def := range dungeon.GetAllDungeons() {
		for _, pw := range def.PlannerPool {
			if _, exists := tableMap[pw.PlannerType.Name]; !exists {
				tableMap[pw.PlannerType.Name] = tableInfo{
					EnemyTableName: def.EnemyTableName,
					ItemTableName:  def.ItemTableName,
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

// buildMapGenChain はBuildChainを使ってRecording付きチェーンを構築する。
// 実装と同じチェーン構築ロジックを共有する。
// 接続性エラー時は本番と同様にシードを変えてリトライする。
// アセット未作成などでチェーン構築に失敗した場合はスキップしてnilを返す
func buildMapGenChain(t *testing.T, pt mapplanner.PlannerType) *mapplanner.PlannerChain {
	t.Helper()
	world := vrt.InitVRTWorld(t)
	for attempt := 0; attempt < mapplanner.MaxPlanRetries; attempt++ {
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
		return chain
	}
	t.Skipf("PlannerType %s のプラン生成が%d回失敗しました", pt.Name, mapplanner.MaxPlanRetries)
	return nil
}

// TestGolden_MapGenSnapshot は全PlannerTypeの全フェーズのスナップショットをJSONで検証する。
// テーブル名はダンジョン定義から取得する
func TestGolden_MapGenSnapshot(t *testing.T) {
	t.Parallel()

	for _, pt := range collectPlannerTypes() {
		chain := buildMapGenChain(t, pt)
		for i, snap := range chain.Snapshots {
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

// TestMapGenImages は全PlannerTypeの各フェーズのVRT画像を生成する。
// 対応するスナップショットJSONの内容が変わった場合のみ画像を再生成する。
// ピクセル比較は行わず、目視確認用の参照画像として保存する
func TestMapGenImages(t *testing.T) {
	t.Parallel()

	for _, pt := range collectPlannerTypes() {
		chain := buildMapGenChain(t, pt)
		for i, snap := range chain.Snapshots {
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
					Seed:          mapGenSeed,
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
