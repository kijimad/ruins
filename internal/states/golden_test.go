package states_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// buildMapGenChain はBuildChainを使ってRecording付きチェーンを構築する。
// 実装と同じチェーン構築ロジックを共有する
func buildMapGenChain(t *testing.T, world w.World, pt mapplanner.PlannerType) *mapplanner.PlannerChain {
	t.Helper()
	chain, err := mapplanner.BuildChain(world, consts.MapTileWidth, consts.MapTileHeight, mapGenSeed, pt)
	require.NoError(t, err)
	chain.Recording = true
	require.NoError(t, chain.Plan())
	return chain
}

// TestGolden_MapGenSnapshot はダンジョン定義ごとに各プランナータイプの全フェーズのスナップショットをJSONで検証する
func TestGolden_MapGenSnapshot(t *testing.T) {
	t.Parallel()

	world := vrt.InitVRTWorld(t)
	for _, def := range dungeon.GetAllDungeons() {
		for _, pw := range def.PlannerPool {
			pt := pw.PlannerType
			pt.EnemyTableName = def.EnemyTableName
			pt.ItemTableName = def.ItemTableName
			pt.Depth = 1

			chain := buildMapGenChain(t, world, pt)
			for i, snap := range chain.Snapshots {
				t.Run(fmt.Sprintf("%s/%s/Phase%d_%s", def.Name, pt.Name, i, snap.Label), func(t *testing.T) {
					t.Parallel()
					data, err := json.MarshalIndent(snap, "", "  ")
					require.NoError(t, err)

					g := goldie.New(t, goldie.WithNameSuffix(".json"))
					g.Assert(t, t.Name(), data)
				})
			}
		}
	}
}

// TestMapGenImages はダンジョン定義ごとに各プランナータイプの各フェーズのVRT画像を生成する。
// 一致率の検証は行わず、目視確認用の参照画像として保存する
func TestMapGenImages(t *testing.T) {
	t.Parallel()

	world := vrt.InitVRTWorld(t)
	for _, def := range dungeon.GetAllDungeons() {
		for _, pw := range def.PlannerPool {
			pt := pw.PlannerType
			pt.EnemyTableName = def.EnemyTableName
			pt.ItemTableName = def.ItemTableName
			pt.Depth = 1

			chain := buildMapGenChain(t, world, pt)
			for i, snap := range chain.Snapshots {
				t.Run(fmt.Sprintf("%s/%s/Phase%d_%s", def.Name, pt.Name, i, snap.Label), func(t *testing.T) {
					t.Parallel()
					pngData := vrt.RenderStatePNG(t, vrt.States(&gs.MapGenVisualizerState{
						PlannerType: pt,
						Seed:        mapGenSeed,
						PhaseIndex:  i,
					}))

					dir := filepath.Join("testdata", "MapGenImages", def.Name, pt.Name)
					require.NoError(t, os.MkdirAll(dir, 0o755))
					path := filepath.Join(dir, fmt.Sprintf("Phase%d_%s.png", i, snap.Label))
					require.NoError(t, os.WriteFile(path, pngData, 0o644))
				})
			}
		}
	}
}
