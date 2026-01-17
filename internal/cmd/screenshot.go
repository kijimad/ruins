package cmd

import (
	"context"
	"fmt"

	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/vrt"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/urfave/cli/v3"
)

// CmdScreenshot はスクリーンショットを撮影するコマンド
var CmdScreenshot = &cli.Command{
	Name:        "screenshot",
	Usage:       "screenshot",
	Description: "screenshot game",
	Action:      runScreenshot,
	Flags:       []cli.Flag{},
}

func runScreenshot(_ context.Context, cmd *cli.Command) error {
	mode := cmd.Args().Get(0)
	if mode == "" {
		return fmt.Errorf("引数が不足している。ステート名が必要")
	}

	// 固定seed値を使用したtown dungeon state
	const townSeedVal = 1
	townStateFactory := gs.NewTownState(gs.WithSeed(townSeedVal))

	switch mode {
	case gs.CraftMenuState{}.String():
		return vrt.RunTestGame(mode, townStateFactory(), &gs.CraftMenuState{})
	case "DebugMenu":
		return vrt.RunTestGame(mode, townStateFactory(), gs.NewDebugMenuState())
	case gs.DungeonState{}.String():
		// 固定seed値を使用する
		const seedVal = 1
		return vrt.RunTestGame(mode, &gs.DungeonState{
			Depth:       1,
			Seed:        seedVal,
			BuilderType: mapplanner.PlannerTypeSmallRoom,
		})
	case gs.EquipMenuState{}.String():
		return vrt.RunTestGame(mode, townStateFactory(), &gs.EquipMenuState{})
	case "GameOver":
		return vrt.RunTestGame(mode, townStateFactory(), gs.NewGameOverMessageState())
	case "Town":
		return vrt.RunTestGame(mode, townStateFactory())
	case gs.InventoryMenuState{}.String():
		return vrt.RunTestGame(mode, townStateFactory(), &gs.InventoryMenuState{})
	case "LoadMenu":
		return vrt.RunTestGame(mode, townStateFactory(), gs.NewLoadMenuState())
	case gs.MainMenuState{}.String():
		return vrt.RunTestGame(mode, &gs.MainMenuState{})
	case gs.MessageState{}.String():
		messageData := messagedata.NewDialogMessage(
			"これはメッセージウィンドウのVRTテストです。\n\n表示状態の確認用メッセージです。",
			"VRTテスト",
		).WithChoice(
			"選択肢1", func(_ w.World) error { return nil },
		).WithChoice(
			"選択肢2", func(_ w.World) error { return nil },
		)
		return vrt.RunTestGame(mode, townStateFactory(), gs.NewMessageState(messageData))
	case "SaveMenu":
		return vrt.RunTestGame(mode, townStateFactory(), gs.NewSaveMenuState())
	case gs.ShopMenuState{}.String():
		return vrt.RunTestGame(mode, townStateFactory(), &gs.ShopMenuState{})
	default:
		return fmt.Errorf("スクリーンショット実行時に対応してないステートが指定された: %s", mode)
	}
}
