package states

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
)

// デバッグメニューの選択肢とスポーン補助をまとめる。開発用の動作確認を1箇所に集約する。

// debugEnterPlanners はデバッグでプランナー単位に生成して試すフロアプランナーの一覧。
// マップ生成の見た目を試す用途なので、遺跡定義でなくプランナー、すなわち大部屋・小部屋などで選ぶ。
var debugEnterPlanners = []mapplanner.PlannerType{
	mapplanner.PlannerTypeBigRoom,
	mapplanner.PlannerTypeSmallRoom,
	mapplanner.PlannerTypeCave,
	mapplanner.PlannerTypeRuins,
	mapplanner.PlannerTypeForest,
}

// NewDebugMenuState は新しいDebugMenuStateインスタンスを作成するファクトリー関数
func NewDebugMenuState() (es.State[w.World], error) {
	return NewChoiceMenu(debugMenuChoices), nil
}

// debugMenuChoices はデバッグメニューの選択肢を返す。開発用のスポーンやメッセージ確認をまとめる
func debugMenuChoices(_ w.World) (string, []Choice) {
	debugName := dungeon.DungeonDebug.Name()
	choices := []Choice{
		{Label: "回復薬スポーン(インベントリ)", Run: popAfter(func(world w.World) error {
			_, err := lifecycle.SpawnBackpackItem(world, "回復薬", 1)
			return err
		})},
		{Label: "レイガンスポーン(インベントリ)", Run: popAfter(func(world w.World) error {
			_, err := lifecycle.SpawnBackpackItem(world, "レイガン", 1)
			return err
		})},
		{Label: "ゲームオーバー", Run: pushChoice(NewGameOverMessageState)},
		{Label: "全ダンジョン踏破", Run: popAfter(func(world w.World) error {
			for _, name := range dungeon.GetAllDungeonNames() {
				query.GetGameProgress(world).MarkDungeonCleared(name)
			}
			return nil
		})},
		{Label: "オーバーワールド開始", Run: func(world w.World) (es.Transition[w.World], error) {
			return es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: []es.StateFactory[w.World]{newGameOverworldState(world)}}, nil
		}},
	}

	// プランナー単位でデバッグ遺跡へ入る選択肢を平坦に追加する。TransReplace ではなく TransPop で
	// ゲームへ戻し、DungeonState.Update が enterDungeonWith を指定プランナーで通す
	for _, pt := range debugEnterPlanners {
		choices = append(choices, Choice{Label: "デバッグ遺跡を生成 " + pt.Name, Run: popAfter(func(world w.World) error {
			return lifecycle.RequestStateChange(world, gc.WarpDungeonEnterWithPlannerEvent(debugName, pt.Name))
		})})
	}

	// 街用NPC・収納箱をスポーン地点の隣に固定配置したデバッグステージへ入る。
	// 街の会話・売買・収納を入ってすぐテストできる
	choices = append(choices, Choice{Label: "デバッグステージを生成", Run: popAfter(func(world w.World) error {
		return lifecycle.RequestStateChange(world, gc.WarpDungeonEnterWithPlannerEvent(dungeon.DungeonDebugTown.Name(), mapplanner.PlannerTypeDebugTown.Name))
	})})

	choices = append(choices,
		Choice{Label: "メッセージ表示テスト", Run: pushMessage(messagedata.NewSystemMessage("ゲームが自動保存されました。\n\n進行状況は安全に記録されています。"))},
		Choice{Label: "アイテム入手イベント", Run: func(world w.World) (es.Transition[w.World], error) {
			for name, count := range map[string]int{"鉄": 1, "木の棒": 1, "フェライトコア": 2} {
				if err := lifecycle.ChangeStackableCount(world, name, count); err != nil {
					return es.Transition[w.World]{}, fmt.Errorf("アイテム追加に失敗: %w", err)
				}
			}
			md := &messagedata.MessageData{Speaker: ""}
			md.AddText("宝箱を発見した。\n\n鉄を手に入れた。\n木の棒を手に入れた。\nフェライトコアを2個手に入れた。\n")
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMessageState(md) }}}, nil
		}},
		Choice{Label: "連鎖メッセージテスト", Run: pushMessage(messagedata.NewSystemMessage("戦闘開始。").SystemMessage("剣と剣がぶつかり合う。").SystemMessage("勝利した。"))},
		Choice{Label: "長いメッセージテスト", Run: pushMessage(messagedata.NewSystemMessage("これは非常に長いメッセージのテストです。\n\nメッセージウィンドウは自動的にサイズを調整し、\n長いテキストでも適切に表示されることを確認しています。\n\n複数行のテキストと改行が正しく処理されること、\nそしてウィンドウの背景やボーダーが適切に描画されることを\nこのテストで検証できます。\n\n日本語のテキストも問題なく表示されるはずです。\n句読点、記号、数字123なども含めて確認してみましょう。"))},
		Choice{Label: "選択肢分岐メッセージテスト", Run: pushMessage(messagedata.NewDialogMessage("敵に遭遇した。", "").
			WithChoiceMessage("戦う", messagedata.NewSystemMessage("戦闘した。")).
			WithChoiceMessage("交渉する", messagedata.NewSystemMessage("交渉した。")).
			WithChoiceMessage("逃走する", messagedata.NewSystemMessage("逃走した。")))},
		Choice{Label: "背景付きメッセージテスト", Run: func(_ w.World) (es.Transition[w.World], error) {
			md := messagedata.NewDialogMessage("これは背景付きメッセージのテストです。", "システム")
			md.BackgroundKey = "hospital1"
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMessageState(md) }}}, nil
		}},
		Choice{Label: "デバッグ表示切り替え", Run: popAfter(func(world w.World) error {
			world.Config.ShowMapDebug = !world.Config.ShowMapDebug
			world.Config.ShowAIDebug = !world.Config.ShowAIDebug
			world.Config.NoEncounter = !world.Config.NoEncounter
			return nil
		})},
		Choice{Label: "オープニング", Run: pushChoice(NewOpeningState)},
		Choice{Label: "全クリアイベント", Run: pushChoice(NewAllClearEventState)},
		Choice{Label: "名前入力", Run: pushChoice(NewCharacterNamingState)},
		Choice{Label: "職業選択", Run: pushChoice(NewCharacterJobState("Ash"))},
		Choice{Label: "隊員スポーン", Run: popAfter(func(world w.World) error {
			player, err := query.GetPlayerEntity(world)
			if err != nil {
				return err
			}
			abilities := gc.Abilities{
				Vitality:  gc.Ability{Base: 10},
				Strength:  gc.Ability{Base: 8},
				Sensation: gc.Ability{Base: 7},
				Dexterity: gc.Ability{Base: 6},
				Agility:   gc.Ability{Base: 9},
				Defense:   gc.Ability{Base: 5},
			}
			if _, err := lifecycle.SpawnSquadMember(world, player, "隊員", abilities, "general"); err != nil {
				return fmt.Errorf("隊員スポーンに失敗: %w", err)
			}
			return nil
		})},
		Choice{Label: "敵スポーン:火の玉(hostile)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "火の玉") })},
		Choice{Label: "敵スポーン:苔亀(neutral)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "苔亀") })},
		Choice{Label: "敵スポーン:ネズミ(cowardly)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "ネズミ") })},
		Choice{Label: "敵スポーン:鉄の番兵(stationary)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "鉄の番兵") })},
		Choice{Label: "敵スポーン:毒蜘蛛(wallHug)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "毒蜘蛛") })},
		Choice{Label: "敵スポーン:スライム(swarm)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "スライム") })},
		Choice{Label: "敵スポーン:骸骨兵(patrol)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "骸骨兵") })},
		Choice{Label: "敵スポーン:野犬(territorial)", Run: stayAfter(func(world w.World) error { return spawnEnemyNearPlayer(world, "野犬") })},
		Choice{Label: "Propスポーン:moving_stone(PassCost)", Run: stayAfter(func(world w.World) error { return spawnPropNearPlayer(world, "moving_stone") })},
		Choice{Label: "Propスポーン:bonfire(光源)", Run: stayAfter(func(world w.World) error { return spawnPropNearPlayer(world, "bonfire") })},
		Choice{Label: "Propスポーン:barrel(破壊可能)", Run: stayAfter(func(world w.World) error { return spawnPropNearPlayer(world, "barrel") })},
		Choice{Label: "Propスポーン:construction_sign(通行不可)", Run: stayAfter(func(world w.World) error { return spawnPropNearPlayer(world, "construction_sign") })},
		Choice{Label: "Propスポーン:木箱(収納・アイテム入り)", Run: stayAfter(spawnStorageWithItems)},
		Choice{Label: "コンポーネント一覧", Run: pushChoice(NewComponentDebugState)},
		Choice{Label: TextClose, Run: func(_ w.World) (es.Transition[w.World], error) {
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}},
	)
	return "", choices
}

// spawnPropNearPlayer はプレイヤーの隣にPropをスポーンする
func spawnPropNearPlayer(world w.World, name string) error {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	playerGrid := world.Components.GridElement.Get(player)
	_, err = lifecycle.SpawnProp(world, name, playerGrid.X+2, playerGrid.Y)
	return err
}

// spawnStorageWithItems はプレイヤーの隣にアイテム入り木箱をスポーンする
func spawnStorageWithItems(world w.World) error {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	playerGrid := world.Components.GridElement.Get(player)
	storageEntity, err := lifecycle.SpawnProp(world, "木箱", playerGrid.X+2, playerGrid.Y)
	if err != nil {
		return err
	}

	// アイテムを収納に格納する
	items := []struct {
		name  string
		count int
	}{
		{"回復薬", 3},
		{"手榴弾", 1},
		{"たいまつ", 1},
	}
	for _, item := range items {
		if _, err := lifecycle.SpawnStorageItem(world, item.name, item.count, storageEntity); err != nil {
			return fmt.Errorf("収納アイテムのスポーンに失敗: %w", err)
		}
	}
	return nil
}

// spawnEnemyNearPlayer はプレイヤーから少し離れた位置に敵をスポーンする
func spawnEnemyNearPlayer(world w.World, name string) error {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	playerGrid := world.Components.GridElement.Get(player)
	_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: playerGrid.X + 8, Y: playerGrid.Y}, name)
	return err
}
