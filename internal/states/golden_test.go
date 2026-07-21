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
	s, err := gs.NewOverworldState(mapplanner.PlannerTypeOverworldField, &overworld.NewGameParams{RunSeed: 42, ChunkW: 30, ChunkH: 20, K: 3})()
	require.NoError(t, err)
	return s
}

func TestGolden_InventoryMenu(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	vrt.AssertStateGolden(t, vrt.States(town, &gs.InventoryMenuState{}))
}

func TestGolden_EquipMenu(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	vrt.AssertStateGolden(t, vrt.States(town, &gs.EquipMenuState{}))
}

func TestGolden_CraftMenu(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	vrt.AssertStateGolden(t, vrt.States(town, &gs.CraftMenuState{}))
}

func TestGolden_ShopMenu(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	vrt.AssertStateGolden(t, vrt.States(town, &gs.ShopMenuState{}))
}

func TestGolden_SaveMenu(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	s, err := gs.NewSaveMenuState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(town, s))
}

func TestGolden_LoadMenu(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	s, err := gs.NewLoadMenuState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(town, s))
}

func TestGolden_DebugMenu(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	s, err := gs.NewDebugMenuState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(town, s))
}

func TestGolden_ComponentDebug(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	s, err := gs.NewComponentDebugState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(town, s))
}

func TestGolden_SquadMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
		playerEntity, err := query.GetPlayerEntity(world)
		require.NoError(t, err)

		_, err = lifecycle.SpawnDefaultSquadMember(world, playerEntity)
		require.NoError(t, err)

		town := newGoldenBackdrop(t)
		squad, err := gs.NewSquadMenuState()
		require.NoError(t, err)
		return []es.State[w.World]{
			town,
			squad,
		}
	})
}

func TestGolden_FormationMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
		playerEntity, err := query.GetPlayerEntity(world)
		require.NoError(t, err)

		_, err = lifecycle.SpawnDefaultSquadMember(world, playerEntity)
		require.NoError(t, err)

		town := newGoldenBackdrop(t)
		formation, err := gs.NewFormationMenuState()
		require.NoError(t, err)
		return []es.State[w.World]{
			town,
			formation,
		}
	})
}

func TestGolden_Dungeon(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:          1,
		DefinitionName: dungeon.DungeonDebug.Name,
		BuilderType:    mapplanner.PlannerTypeSmallRoom,
	}))
}

func TestGolden_Overworld(t *testing.T) {
	t.Parallel()
	s, err := gs.NewOverworldState(mapplanner.PlannerTypeOverworldField, &overworld.NewGameParams{RunSeed: 42, ChunkW: 30, ChunkH: 20, K: 3})()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(s))
}

// TestGolden_OverworldFrost は寒波前線の氷オーバーレイの描画を固定する。
// 総ターン数を進めて前線を可視帯へ入れ、西側が凍結壁として濃く覆われる様子を見る。
func TestGolden_OverworldFrost(t *testing.T) {
	t.Parallel()
	s, err := gs.NewOverworldState(mapplanner.PlannerTypeOverworldField, &overworld.NewGameParams{RunSeed: 42, ChunkW: 30, ChunkH: 20, K: 3})()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
		// 前線が帯へ食い込むところまでターンを進める。updateFront が FrontEastAbsX を導出する
		query.GetGameTime(world).TotalTurns = 300
		return []es.State[w.World]{s}
	})
}

func TestGolden_LookAround(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:          1,
		DefinitionName: dungeon.DungeonDebug.Name,
		BuilderType:    mapplanner.PlannerTypeSmallRoom,
	}, &gs.LookAroundState{}))
}

func TestGolden_GameOver(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	s, err := gs.NewGameOverMessageState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(town, s))
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
	town := newGoldenBackdrop(t)
	msgState, err := gs.NewMessageState(messageData)
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(town, msgState))
}

func TestGolden_Status(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	s, err := gs.NewStatusState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(town, s))
}

func TestGolden_MemberStatus(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
		playerEntity, err := query.GetPlayerEntity(world)
		require.NoError(t, err)

		member, err := lifecycle.SpawnDefaultSquadMember(world, playerEntity)
		require.NoError(t, err)

		town := newGoldenBackdrop(t)
		status, err := gs.NewMemberStatusState(member)
		require.NoError(t, err)
		return []es.State[w.World]{town, status}
	})
}

func TestGolden_TavernMenu(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	s, err := gs.NewTavernMenuState()
	require.NoError(t, err)
	vrt.AssertStateGolden(t, vrt.States(town, s))
}

func TestGolden_Shooting(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:          1,
		DefinitionName: dungeon.DungeonDebug.Name,
		BuilderType:    mapplanner.PlannerTypeSmallRoom,
	}, &gs.ShootingState{}))
}

func TestGolden_Pickup(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:          1,
		DefinitionName: dungeon.DungeonDebug.Name,
		BuilderType:    mapplanner.PlannerTypeSmallRoom,
	}, &gs.PickupState{}))
}

func TestGolden_Place(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, vrt.States(&gs.DungeonState{
		Depth:          1,
		DefinitionName: dungeon.DungeonDebug.Name,
		BuilderType:    mapplanner.PlannerTypeSmallRoom,
	}, &gs.PlaceState{}))
}

func TestGolden_PersistentMessage(t *testing.T) {
	t.Parallel()
	town := newGoldenBackdrop(t)
	messageData := messagedata.NewDialogMessage(
		"永続メッセージのVRTテストです。",
		"テスト",
	)
	vrt.AssertStateGolden(t, vrt.States(town, gs.NewPersistentMessageState(messageData)))
}

func TestGolden_StorageMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, func(world w.World) []es.State[w.World] {
		storageEntity, err := lifecycle.SpawnProp(world, "木箱", 3, 3)
		require.NoError(t, err)

		_, err = lifecycle.SpawnStorageItem(world, "回復薬", 1, storageEntity)
		require.NoError(t, err)

		town := newGoldenBackdrop(t)
		storageState, stateErr := gs.NewStorageMenuState(storageEntity)
		require.NoError(t, stateErr)

		return []es.State[w.World]{
			town,
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
		def := &dungeons[i]
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
