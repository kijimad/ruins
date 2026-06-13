package states_test

import (
	"os"
	"testing"

	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"
)

func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

func TestGolden_MainMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, &gs.MainMenuState{})
}

func TestGolden_CharacterNaming(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, &gs.CharacterNamingState{})
}

func TestGolden_CharacterJob(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewCharacterJobState("Ash")())
}

func TestGolden_Town(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()())
}

func TestGolden_InventoryMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), &gs.InventoryMenuState{})
}

func TestGolden_EquipMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), &gs.EquipMenuState{})
}

func TestGolden_CraftMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), &gs.CraftMenuState{})
}

func TestGolden_ShopMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), &gs.ShopMenuState{})
}

func TestGolden_AutoSell(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), gs.NewAutoSellState()())
}

func TestGolden_SaveMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), gs.NewSaveMenuState())
}

func TestGolden_LoadMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), gs.NewLoadMenuState())
}

func TestGolden_DebugMenu(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), gs.NewDebugMenuState())
}

func TestGolden_DungeonSelect(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), gs.NewDungeonSelectState())
}

func TestGolden_Dungeon(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, &gs.DungeonState{
		Depth:       1,
		BuilderType: mapplanner.PlannerTypeSmallRoom,
	})
}

func TestGolden_LookAround(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, &gs.DungeonState{
		Depth:       1,
		BuilderType: mapplanner.PlannerTypeSmallRoom,
	}, &gs.LookAroundState{})
}

func TestGolden_GameOver(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), gs.NewGameOverMessageState())
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
	vrt.AssertStateGolden(t, gs.NewTownState()(), gs.NewMessageState(messageData))
}

func TestGolden_Status(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, gs.NewTownState()(), gs.NewStatusState())
}

func TestGolden_Shooting(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, &gs.DungeonState{
		Depth:       1,
		BuilderType: mapplanner.PlannerTypeSmallRoom,
	}, &gs.ShootingState{})
}

func TestGolden_Pickup(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, &gs.DungeonState{
		Depth:       1,
		BuilderType: mapplanner.PlannerTypeSmallRoom,
	}, &gs.PickupState{})
}

func TestGolden_Place(t *testing.T) {
	t.Parallel()
	vrt.AssertStateGolden(t, &gs.DungeonState{
		Depth:       1,
		BuilderType: mapplanner.PlannerTypeSmallRoom,
	}, &gs.PlaceState{})
}

func TestGolden_PersistentMessage(t *testing.T) {
	t.Parallel()
	messageData := messagedata.NewDialogMessage(
		"永続メッセージのVRTテストです。",
		"テスト",
	)
	vrt.AssertStateGolden(t, gs.NewTownState()(), gs.NewPersistentMessageState(messageData))
}
