package states

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 各ステートのファクトリー関数を集約したファイル

// NewDungeonMenuState は新しいDungeonMenuStateインスタンスを作成するファクトリー関数
func NewDungeonMenuState() (es.State[w.World], error) {
	persistentState := NewPersistentMessageState(nil)

	persistentState.messageData = messagedata.NewSystemMessage("").
		WithChoice("状態", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewStatusState},
			})
			return nil
		}).
		WithChoice("所持", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewInventoryMenuState},
			})
			return nil
		}).
		WithChoice("装備", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewEquipMenuState},
			})
			return nil
		}).
		WithChoice("命令", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewSquadMenuState},
			})
			return nil
		}).
		WithChoice("部隊", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewFormationMenuState},
			})
			return nil
		}).
		WithChoice("書込", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewSaveMenuState},
			})
			return nil
		}).
		WithChoice("終了", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type:          es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{NewMainMenuState},
			})
			return nil
		}).
		WithChoice("閉じる", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type: es.TransPop,
			})
			return nil
		})

	return persistentState, nil
}

// NewCraftMenuState は新しいCraftMenuStateインスタンスを作成するファクトリー関数
func NewCraftMenuState() (es.State[w.World], error) {
	return &CraftMenuState{}, nil
}

// NewInventoryMenuState は新しいInventoryMenuStateインスタンスを作成するファクトリー関数
func NewInventoryMenuState() (es.State[w.World], error) {
	return &InventoryMenuState{}, nil
}

// NewEquipMenuState は新しいEquipMenuStateインスタンスを作成するファクトリー関数
func NewEquipMenuState() (es.State[w.World], error) {
	return &EquipMenuState{}, nil
}

// NewDebugMenuState は新しいDebugMenuStateインスタンスを作成するファクトリー関数
func NewDebugMenuState() (es.State[w.World], error) {
	messageState := &MessageState{}

	messageState.messageData = messagedata.NewSystemMessage("").
		WithChoice("回復薬スポーン(インベントリ)", func(world w.World) error {
			_, err := lifecycle.SpawnBackpackItem(world, "回復薬", 1)
			if err != nil {
				return fmt.Errorf("error spawning item: %w", err)
			}
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		}).
		WithChoice("レイガンスポーン(インベントリ)", func(world w.World) error {
			_, err := lifecycle.SpawnBackpackItem(world, "レイガン", 1)
			if err != nil {
				return fmt.Errorf("error spawning item: %w", err)
			}
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		}).
		WithChoice("ゲームオーバー", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewGameOverMessageState},
			})
			return nil
		}).
		WithChoice("ダンジョン選択", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewDungeonSelectState},
			})
			return nil
		}).
		WithChoice("全ダンジョン踏破", func(world w.World) error {
			for _, name := range dungeon.GetAllDungeonNames() {
				query.GetGameProgress(world).MarkDungeonCleared(name)
			}
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		}).
		WithChoice("ダンジョン開始(大部屋)", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewDungeonState(1, WithBuilderType(mapplanner.PlannerTypeBigRoom)),
				}})
			return nil
		}).
		WithChoice("ダンジョン開始(小部屋)", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewDungeonState(1, WithBuilderType(mapplanner.PlannerTypeSmallRoom)),
				}})
			return nil
		}).
		WithChoice("ダンジョン開始(洞窟)", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewDungeonState(1, WithBuilderType(mapplanner.PlannerTypeCave)),
				}})
			return nil
		}).
		WithChoice("ダンジョン開始(廃墟)", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewDungeonState(1, WithBuilderType(mapplanner.PlannerTypeRuins)),
				}})
			return nil
		}).
		WithChoice("ダンジョン開始(森)", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewDungeonState(1, WithBuilderType(mapplanner.PlannerTypeForest)),
				}})
			return nil
		}).
		WithChoice("ダンジョン開始(小さな町)", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewDungeonState(1, WithBuilderType(mapplanner.PlannerTypeSmallTown)),
				}})
			return nil
		}).
		WithChoice("ダンジョン開始(ボス部屋)", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewDungeonState(1, WithBuilderType(mapplanner.PlannerTypeBossFloor)),
				}})
			return nil
		}).
		WithChoice("市街地開始", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewTownState(),
				}})
			return nil
		}).
		WithChoice("メッセージ表示テスト", func(_ w.World) error {
			testMessageData := messagedata.NewSystemMessage("ゲームが自動保存されました。\n\n進行状況は安全に記録されています。")
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMessageState(testMessageData) }}})
			return nil
		}).
		WithChoice("アイテム入手イベント", func(world w.World) error {
			// アイテムを実際にインベントリに追加
			if err := lifecycle.ChangeStackableCount(world, "鉄", 1); err != nil {
				return fmt.Errorf("アイテム追加に失敗: %w", err)
			}
			if err := lifecycle.ChangeStackableCount(world, "木の棒", 1); err != nil {
				return fmt.Errorf("アイテム追加に失敗: %w", err)
			}
			if err := lifecycle.ChangeStackableCount(world, "フェライトコア", 2); err != nil {
				return fmt.Errorf("アイテム追加に失敗: %w", err)
			}

			// アイテム入手完了後の表示用メッセージを生成
			messageText := "宝箱を発見した。\n\n" +
				"鉄を手に入れた。\n" +
				"木の棒を手に入れた。\n" +
				"フェライトコアを2個手に入れた。\n"

			itemMessageData := &messagedata.MessageData{
				Speaker: "",
			}
			itemMessageData.AddText(messageText)
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMessageState(itemMessageData) }}})
			return nil
		}).
		WithChoice("長いメッセージテスト", func(_ w.World) error {
			longText := `これは非常に長いメッセージのテストです。

メッセージウィンドウは自動的にサイズを調整し、
長いテキストでも適切に表示されることを確認しています。

複数行のテキストと改行が正しく処理されること、
そしてウィンドウの背景やボーダーが適切に描画されることを
このテストで検証できます。

日本語のテキストも問題なく表示されるはずです。
句読点、記号、数字123なども含めて確認してみましょう。`

			longMessageData := messagedata.NewSystemMessage(longText)
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMessageState(longMessageData) }}})
			return nil
		}).
		WithChoice("連鎖メッセージテスト", func(_ w.World) error {
			chainMessageData := messagedata.NewSystemMessage("戦闘開始。").
				SystemMessage("剣と剣がぶつかり合う。").
				SystemMessage("勝利した。")

			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMessageState(chainMessageData) }}})
			return nil
		}).
		WithChoice("選択肢分岐メッセージテスト", func(_ w.World) error {
			battleMessage := messagedata.NewSystemMessage("戦闘した。")
			negotiateMessage := messagedata.NewSystemMessage("交渉した。")
			escapeMessage := messagedata.NewSystemMessage("逃走した。")

			choiceMessageData := messagedata.NewDialogMessage("敵に遭遇した。", "").
				WithChoiceMessage("戦う", battleMessage).
				WithChoiceMessage("交渉する", negotiateMessage).
				WithChoiceMessage("逃走する", escapeMessage)

			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMessageState(choiceMessageData) }}})
			return nil
		}).
		WithChoice("選択肢処理テスト", func(_ w.World) error {
			choiceAction1 := func() {
				println("実行: 1")
			}
			choiceAction2 := func() {
				println("実行: 2")
			}

			onCompleteAction := func() {
				println("Complete Action")
			}

			result1 := messagedata.NewSystemMessage("選択肢1を選びました。").
				SystemMessage("何かの処理が実行されました。").
				WithOnComplete(onCompleteAction)

			result2 := messagedata.NewSystemMessage("選択肢2を選びました。").
				SystemMessage("別の処理が実行されました。").
				WithOnComplete(onCompleteAction)

			testMessageData := messagedata.NewDialogMessage("処理のテストです。選択肢を選んでください。", "システム").
				WithChoiceMessage("処理1を実行", result1).
				WithChoiceMessage("処理2を実行", result2)

			testMessageData.Choices[0].Action = func(_ w.World) error { choiceAction1(); return nil }
			testMessageData.Choices[1].Action = func(_ w.World) error { choiceAction2(); return nil }

			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMessageState(testMessageData) }}})
			return nil
		}).
		WithChoice("デバッグ表示切り替え", func(world w.World) error {
			world.Config.ShowMapDebug = !world.Config.ShowMapDebug
			world.Config.ShowAIDebug = !world.Config.ShowAIDebug
			world.Config.NoEncounter = !world.Config.NoEncounter
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		}).
		WithChoice("背景付きメッセージテスト", func(_ w.World) error {
			testMessage := messagedata.NewDialogMessage("これは背景付きメッセージのテストです。", "システム")
			testMessage.BackgroundKey = "hospital1"
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{
					func() (es.State[w.World], error) {
						return NewMessageState(testMessage)
					},
				}})
			return nil
		}).
		WithChoice("オープニング", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewOpeningState},
			})
			return nil
		}).
		WithChoice("全クリアイベント", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewAllClearEventState,
				},
			})
			return nil
		}).
		WithChoice("次の階層に進む", func(world w.World) error {
			currentDepth := query.GetDungeon(world).Depth

			// 次の階層に遷移
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewDungeonState(currentDepth + 1),
				},
			})
			return nil
		}).
		WithChoice("名前入力", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewCharacterNamingState},
			})
			return nil
		}).
		WithChoice("職業選択", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewCharacterJobState("Ash")},
			})
			return nil
		}).
		WithChoice("探索結果", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{NewAutoSellState()},
			})
			return nil
		}).
		WithChoice("隊員スポーン", func(world w.World) error {
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
			_, err = lifecycle.SpawnSquadMember(world, player, "隊員", abilities, "general")
			if err != nil {
				return fmt.Errorf("隊員スポーンに失敗: %w", err)
			}
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		}).
		WithChoice("敵スポーン:火の玉(hostile)", func(world w.World) error {
			return spawnEnemyNearPlayer(world, "火の玉")
		}).
		WithChoice("敵スポーン:苔亀(neutral)", func(world w.World) error {
			return spawnEnemyNearPlayer(world, "苔亀")
		}).
		WithChoice("敵スポーン:苔亀(wander)", func(world w.World) error {
			return spawnEnemyNearPlayer(world, "苔亀")
		}).
		WithChoice("敵スポーン:ネズミ(cowardly)", func(world w.World) error {
			return spawnEnemyNearPlayer(world, "ネズミ")
		}).
		WithChoice("敵スポーン:鉄の番兵(stationary)", func(world w.World) error {
			return spawnEnemyNearPlayer(world, "鉄の番兵")
		}).
		WithChoice("敵スポーン:毒蜘蛛(wallHug)", func(world w.World) error {
			return spawnEnemyNearPlayer(world, "毒蜘蛛")
		}).
		WithChoice("敵スポーン:スライム(swarm)", func(world w.World) error {
			return spawnEnemyNearPlayer(world, "スライム")
		}).
		WithChoice("敵スポーン:骸骨兵(patrol)", func(world w.World) error {
			return spawnEnemyNearPlayer(world, "骸骨兵")
		}).
		WithChoice("敵スポーン:野犬(territorial)", func(world w.World) error {
			return spawnEnemyNearPlayer(world, "野犬")
		}).
		WithChoice("Propスポーン:moving_stone(PassCost)", func(world w.World) error {
			return spawnPropNearPlayer(world, "moving_stone")
		}).
		WithChoice("Propスポーン:bonfire(光源)", func(world w.World) error {
			return spawnPropNearPlayer(world, "bonfire")
		}).
		WithChoice("Propスポーン:barrel(破壊可能)", func(world w.World) error {
			return spawnPropNearPlayer(world, "barrel")
		}).
		WithChoice("Propスポーン:construction_sign(通行不可)", func(world w.World) error {
			return spawnPropNearPlayer(world, "construction_sign")
		}).
		WithChoice("Propスポーン:木箱(収納・アイテム入り)", spawnStorageWithItems).
		WithChoice("コンポーネント一覧", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewComponentDebugState},
			})
			return nil
		}).
		WithChoice("閉じる", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransPop,
			})
			return nil
		})

	return messageState, nil
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
	_, err = lifecycle.SpawnEnemy(world, int(playerGrid.X)+8, int(playerGrid.Y), name)
	return err
}

// DungeonStateOption はDungeonStateのオプション設定関数
type DungeonStateOption func(*DungeonState)

// WithBuilderType はマップビルダータイプを設定するオプション
func WithBuilderType(builderType mapplanner.PlannerType) DungeonStateOption {
	return func(ds *DungeonState) {
		ds.BuilderType = builderType
	}
}

// WithDefinitionName はダンジョン定義名を設定するオプション
func WithDefinitionName(name string) DungeonStateOption {
	return func(ds *DungeonState) {
		ds.DefinitionName = name
	}
}

// WithResume はセーブから復帰するモードにするオプション。
// マップの再生成・プレイヤー再配置を行わず、復元済みのワールドをそのまま使う
func WithResume() DungeonStateOption {
	return func(ds *DungeonState) {
		ds.Resume = true
	}
}

// WithEscapeTarget は脱出時の遷移先を設定するオプション。
// 設定すると自動精算(AutoSell→Town)を通さず、TransReplace で指定先へ戻す（マクロ移動から潜行するとき MacroMap を渡す）
func WithEscapeTarget(target es.StateFactory[w.World]) DungeonStateOption {
	return func(ds *DungeonState) {
		ds.EscapeTarget = target
	}
}

// NewDungeonState はDungeonStateインスタンスを作成するファクトリー関数
// デフォルトではBuilderTypeはPlannerTypeRandomになる
func NewDungeonState(depth int, opts ...DungeonStateOption) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		ds := &DungeonState{
			Depth:       depth,
			BuilderType: mapplanner.PlannerTypeRandom,
		}
		for _, opt := range opts {
			opt(ds)
		}
		return ds, nil
	}
}

// NewDemoStartState はデモ用の初期化ステートを作成するファクトリー関数
// キャラクター作成をスキップしてデフォルトのプレイヤーを生成し、TownStateに遷移する
func NewDemoStartState() (es.State[w.World], error) {
	return &DemoStartState{}, nil
}

// NewTownState は街のステートを作成するファクトリー関数
func NewTownState(opts ...DungeonStateOption) es.StateFactory[w.World] {
	allOpts := make([]DungeonStateOption, 0, 2+len(opts))
	allOpts = append(allOpts, WithBuilderType(mapplanner.PlannerTypeTown))
	allOpts = append(allOpts, WithDefinitionName(dungeon.DungeonTown.Name))
	allOpts = append(allOpts, opts...)

	// 街は常に深度0
	return NewDungeonState(0, allOpts...)
}

// NewDungeonSelectState はダンジョン選択画面のStateを作成するファクトリー関数
func NewDungeonSelectState() (es.State[w.World], error) {
	return &DungeonSelectState{}, nil
}

// NewMainMenuState は新しいMainMenuStateインスタンスを作成するファクトリー関数
func NewMainMenuState() (es.State[w.World], error) {
	return &MainMenuState{}, nil
}

// NewSettingsMenuState は新しいSettingsMenuStateインスタンスを作成するファクトリー関数
func NewSettingsMenuState() (es.State[w.World], error) {
	return &SettingsMenuState{}, nil
}

// NewGameOverMessageState はゲームオーバー用のMessageStateを作成するファクトリー関数
func NewGameOverMessageState() (es.State[w.World], error) {
	messageState := &MessageState{}

	// ゲームオーバーメッセージを作成（選択肢付き）
	messageData := messagedata.NewSystemMessage("死亡した。").
		WithChoice("メインメニューに戻る", func(_ w.World) error {
			// メインメニューに遷移
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{NewMainMenuState}})
			return nil
		})

	// MessageStateにMessageDataを設定
	messageState.messageData = messageData

	return messageState, nil
}

// NewAllClearEventState は全ダンジョンクリア時のイベントStateを作成するファクトリー関数
func NewAllClearEventState() (es.State[w.World], error) {
	messageState := &MessageState{}

	messageData := messagedata.NewSystemMessage("すべての遺跡を踏破した。\n\n大穴の底に眠っていた古代の気配が、ようやく静まった。").
		WithChoice("閉じる", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		})

	messageState.messageData = messageData
	return messageState, nil
}

// NewSaveMenuState は手動セーブ画面を作成するファクトリー関数。
// 固定4スロットで、主人公名とタイムスタンプを表示する。
func NewSaveMenuState() (es.State[w.World], error) {
	messageState := &MessageState{}
	saveManager, err := save.NewSerializationManager()
	if err != nil {
		return nil, fmt.Errorf("セーブマネージャーの作成に失敗: %w", err)
	}
	messageData := messagedata.NewSystemMessage("")

	for i := 1; i <= 4; i++ {
		slotName := fmt.Sprintf("slot%d", i)
		label := formatSaveSlotLabel(saveManager, slotName)

		messageData.WithChoice(label, func(world w.World) error {
			if err := saveManager.SaveWorld(world, slotName); err != nil {
				return fmt.Errorf("save failed: %w", err)
			}
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransSwitch,
				NewStateFuncs: []es.StateFactory[w.World]{NewSaveMenuState}})
			return nil
		})
	}

	messageData.WithChoice("戻る", func(_ w.World) error {
		messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		return nil
	})

	messageState.messageData = messageData
	return messageState, nil
}

// NewLoadMenuState はロード画面を作成するファクトリー関数。
// 手動4スロットとオートセーブ4スロットをセクション分けで表示する。
func NewLoadMenuState() (es.State[w.World], error) {
	messageState := &MessageState{}
	saveManager, err := save.NewSerializationManager()
	if err != nil {
		return nil, fmt.Errorf("セーブマネージャーの作成に失敗: %w", err)
	}
	messageData := messagedata.NewSystemMessage("")

	// 手動セーブセクション
	messageData.WithChoice("手動セーブ", nil)
	for i := 1; i <= 4; i++ {
		slotName := fmt.Sprintf("slot%d", i)
		addLoadSlot(messageData, messageState, saveManager, slotName)
	}

	// オートセーブセクション
	messageData.WithChoice("オートセーブ", nil)
	autoSaves, err := saveManager.ListAutoSaves()
	if err != nil {
		return nil, fmt.Errorf("オートセーブ一覧の取得に失敗: %w", err)
	}
	if len(autoSaves) > 4 {
		autoSaves = autoSaves[:4]
	}
	for i := range 4 {
		if i < len(autoSaves) {
			addLoadSlot(messageData, messageState, saveManager, autoSaves[i])
		} else {
			messageData.WithChoice("  ---", nil)
		}
	}

	messageData.WithChoice("戻る", func(_ w.World) error {
		messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		return nil
	})

	messageState.messageData = messageData
	return messageState, nil
}

// addLoadSlot はロードメニューにスロットを追加する。
// データが存在するスロットは選択可能、存在しないスロットは "---" で選択不可にする。
func addLoadSlot(messageData *messagedata.MessageData, messageState *MessageState, saveManager *save.SerializationManager, slotName string) {
	if !saveManager.SaveFileExists(slotName) {
		messageData.WithChoice("  ---", nil)
		return
	}

	label := formatSaveSlotLabel(saveManager, slotName)
	messageData.WithChoice(label, func(world w.World) error {
		err := saveManager.LoadWorld(world, slotName)
		if err != nil {
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return err
		}
		// 復元済みの現在地（ダンジョン定義名・深度。町も深度0のダンジョンとして扱う）から
		// 再生成せずに復帰する
		dungeonState := query.GetDungeon(world)
		resume := NewDungeonState(
			dungeonState.Depth,
			WithDefinitionName(dungeonState.DefinitionName),
			WithResume(),
		)
		messageState.SetTransition(es.Transition[w.World]{
			Type:          es.TransReplace,
			NewStateFuncs: []es.StateFactory[w.World]{resume}})
		return nil
	})
}

// formatSaveSlotLabel はセーブスロットの表示ラベルを生成する。
// データがある場合は "プレイヤー名  日時" 、ない場合は "---" を返す。
func formatSaveSlotLabel(saveManager *save.SerializationManager, slotName string) string {
	if !saveManager.SaveFileExists(slotName) {
		return "---"
	}

	playerName, nameErr := saveManager.GetSavePlayerName(slotName)
	timestamp, tsErr := saveManager.GetSaveFileTimestamp(slotName)

	if nameErr == nil && tsErr == nil {
		return fmt.Sprintf("  %s  %s", playerName, timestamp.Format("01/02 15:04"))
	}
	return "  データあり"
}

// NewMessageState はメッセージデータを受け取って新しいMessageStateを作成するファクトリー関数
func NewMessageState(messageData *messagedata.MessageData) (es.State[w.World], error) {
	return &MessageState{
		messageData: messageData,
	}, nil
}

// NewSquadMenuState は隊員管理画面のStateを作成するファクトリー関数
func NewSquadMenuState() (es.State[w.World], error) {
	return &SquadMenuState{}, nil
}

// NewMemberStatusState は隊員ステータス詳細画面のStateを作成するファクトリー関数
func NewMemberStatusState(member ecs.Entity) (es.State[w.World], error) {
	return &MemberStatusState{member: member}, nil
}

// NewFormationMenuState は隊編成画面のStateを作成するファクトリー関数
func NewFormationMenuState() (es.State[w.World], error) {
	return &FormationMenuState{}, nil
}

// NewTavernMenuState は酒場の雇用画面のStateを作成するファクトリー関数
func NewTavernMenuState() (es.State[w.World], error) {
	return &TavernMenuState{}, nil
}

// NewShopMenuState は新しいShopMenuStateインスタンスを作成するファクトリー関数
func NewShopMenuState() (es.State[w.World], error) {
	return &ShopMenuState{}, nil
}

// NewMacroMapState はマクロ移動（ルート網）画面のStateを作成するファクトリー関数
func NewMacroMapState() (es.State[w.World], error) {
	return &MacroMapState{}, nil
}

// NewStorageMenuState は収納メニューStateを作成する
func NewStorageMenuState(storageEntity ecs.Entity) (es.State[w.World], error) {
	return &StorageMenuState{storageEntity: storageEntity}, nil
}

// NewInteractionMenuState はインタラクションメニューStateを作成する
func NewInteractionMenuState(world w.World) (es.State[w.World], error) {
	interactionActions := GetInteractionActions(world)

	if len(interactionActions) == 0 {
		messageState := &MessageState{}
		messageState.messageData = messagedata.NewSystemMessage("実行可能なアクションがありません。")
		return messageState, nil
	}

	return newActionChoiceMenu(interactionActions), nil
}

// newActionChoiceMenu はInteractionActionのリストから選択メニューを作成する
func newActionChoiceMenu(actions []InteractionAction) es.State[w.World] {
	messageState := &MessageState{}
	messageState.messageData = messagedata.NewSystemMessage("")

	for _, action := range actions {
		messageState.messageData.WithChoice(action.Label, func(world w.World) error {
			playerEntity, err := query.GetPlayerEntity(world)
			if err != nil {
				return fmt.Errorf("failed to get player entity: %w", err)
			}

			if _, err := activity.ExecuteInteraction(playerEntity, action.Target, action.Interaction, world); err != nil {
				return fmt.Errorf("アクション実行失敗: %w", err)
			}

			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		})
	}

	messageState.messageData.WithChoice("キャンセル", func(_ w.World) error {
		messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		return nil
	})

	return messageState
}

// NewMerchantDialogState は商人との会話ステートを作成
func NewMerchantDialogState(speakerName string) (es.State[w.World], error) {
	persistentState := NewPersistentMessageState(nil)

	persistentState.messageData = messagedata.NewDialogMessage("", speakerName).
		AddText(`何か取引しないかい?

いい物揃ってるよ。`).
		WithChoice("見る", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewShopMenuState},
			})
			return nil
		}).
		WithChoice("用は無い", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		})

	return persistentState, nil
}

// NewTavernKeeperDialogState は酒場の主人との会話ステートを作成
func NewTavernKeeperDialogState(speakerName string) (es.State[w.World], error) {
	persistentState := NewPersistentMessageState(nil)

	persistentState.messageData = messagedata.NewDialogMessage("", speakerName).
		AddText(`うちには腕の立つ連中が集まってるよ。

隊員を雇うかい?`).
		WithChoice("雇う", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewTavernMenuState},
			})
			return nil
		}).
		WithChoice("用は無い", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		})

	return persistentState, nil
}

// NewDoctorDialogState は怪しい科学者との会話ステートを作成
func NewDoctorDialogState(speakerName string) (es.State[w.World], error) {
	persistentState := NewPersistentMessageState(nil)

	persistentState.messageData = messagedata.NewDialogMessage("", speakerName).
		AddText(`フフフ...わしの秘密の技術で物質再構築してやろう

地髄と素材を持ってくるのじゃ!`).
		WithChoice("合成したい", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewCraftMenuState},
			})
			return nil
		}).
		WithChoice("用は無い", func(_ w.World) error {
			persistentState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		})

	return persistentState, nil
}

// NewOpeningState はオープニングを表示するStateを作成するファクトリー関数
// 完了後はポップする。後続ステートが必要な場合は呼び出し側でスタックに積む
func NewOpeningState() (es.State[w.World], error) {
	// 1. 黒背景: 荒野の大穴
	page1a := &messagedata.MessageData{Speaker: "", BackgroundKey: "black1"}
	page1a.AddText("見渡すかぎりの荒野に、大穴がひとつ、口を開けている。")

	// 2. 穴背景: 空ページ（背景だけ見せる）→ 遺跡の説明
	blank := &messagedata.MessageData{Speaker: "", BackgroundKey: "hole1"}
	page1b := &messagedata.MessageData{Speaker: ""}
	page1b.AddText("穴の底には古代文明の遺跡がある。\n").
		AddText("宝が出る。怪物も出る。潜った者の半分は帰ってこない。\n").
		AddText("穴のまわりには潜る者、売る者、買う者で街ができた。")

	// 3. 酒場背景: 空ページ（背景だけ見せる）→ 拾い屋の噂
	blankBar := &messagedata.MessageData{Speaker: "", BackgroundKey: "bar1"}
	page2 := &messagedata.MessageData{Speaker: ""}
	page2.AddText("「聞いたか。底狙いの奴、また一人消えたってよ。」\n").
		AddText("「何人目だ。」\n").
		AddText("「さあな。数えるのはとっくにやめた。」\n\n").
		AddText("「でさ、次の").
		AddKeyword("拾い屋").
		AddText("が来たんだが...そいつも底狙いだと。」\n")

	first := messagedata.ChainMessages(page1a, blank, page1b, blankBar, page2)
	return NewMessageState(first)
}
