package states_test

import (
	"os"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
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

		item, err := worldhelper.SpawnItem(world, "回復薬", 1, gc.LocationTypeOnField)
		require.NoError(t, err)
		worldhelper.MoveToStorage(world, item, storageEntity)

		return []es.State[w.World]{
			gs.NewTownState()(),
			gs.NewStorageMenuState(storageEntity),
		}
	})
}
