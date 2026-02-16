package states

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/actions"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// 各ステートのファクトリー関数を集約したファイル

// NewDungeonMenuState は新しいDungeonMenuStateインスタンスを作成するファクトリー関数
func NewDungeonMenuState() es.State[w.World] {
	persistentState := NewPersistentMessageState(nil)

	persistentState.messageData = messagedata.NewSystemMessage("").
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

	return persistentState
}

// NewCraftMenuState は新しいCraftMenuStateインスタンスを作成するファクトリー関数
func NewCraftMenuState() es.State[w.World] {
	return &CraftMenuState{}
}

// NewInventoryMenuState は新しいInventoryMenuStateインスタンスを作成するファクトリー関数
func NewInventoryMenuState() es.State[w.World] {
	return &InventoryMenuState{}
}

// NewEquipMenuState は新しいEquipMenuStateインスタンスを作成するファクトリー関数
func NewEquipMenuState() es.State[w.World] {
	return &EquipMenuState{}
}

// NewDebugMenuState は新しいDebugMenuStateインスタンスを作成するファクトリー関数
func NewDebugMenuState() es.State[w.World] {
	messageState := &MessageState{}

	messageState.messageData = messagedata.NewSystemMessage("").
		WithChoice("回復薬スポーン(インベントリ)", func(world w.World) error {
			_, err := worldhelper.SpawnItem(world, "回復薬", 1, gc.ItemLocationInPlayerBackpack)
			if err != nil {
				return fmt.Errorf("error spawning item: %w", err)
			}
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		}).
		WithChoice("手榴弾スポーン(インベントリ)", func(world w.World) error {
			_, err := worldhelper.SpawnItem(world, "手榴弾", 1, gc.ItemLocationInPlayerBackpack)
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
		WithChoice("ダンジョン開始(広場)", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewDungeonState(1, WithBuilderType(mapplanner.PlannerTypeTownPlaza)),
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
				NewStateFuncs: []es.StateFactory[w.World]{func() es.State[w.World] { return NewMessageState(testMessageData) }}})
			return nil
		}).
		WithChoice("アイテム入手イベント", func(world w.World) error {
			// アイテムを実際にインベントリに追加
			_ = worldhelper.ChangeStackableCount(world, "鉄", 1)
			_ = worldhelper.ChangeStackableCount(world, "木の棒", 1)
			_ = worldhelper.ChangeStackableCount(world, "フェライトコア", 2)

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
				NewStateFuncs: []es.StateFactory[w.World]{func() es.State[w.World] { return NewMessageState(itemMessageData) }}})
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
				NewStateFuncs: []es.StateFactory[w.World]{func() es.State[w.World] { return NewMessageState(longMessageData) }}})
			return nil
		}).
		WithChoice("連鎖メッセージテスト", func(_ w.World) error {
			chainMessageData := messagedata.NewSystemMessage("戦闘開始。").
				SystemMessage("剣と剣がぶつかり合う。").
				SystemMessage("勝利した。")

			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{func() es.State[w.World] { return NewMessageState(chainMessageData) }}})
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
				NewStateFuncs: []es.StateFactory[w.World]{func() es.State[w.World] { return NewMessageState(choiceMessageData) }}})
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
				NewStateFuncs: []es.StateFactory[w.World]{func() es.State[w.World] { return NewMessageState(testMessageData) }}})
			return nil
		}).
		WithChoice("デバッグ表示切り替え", func(_ w.World) error {
			cfg := config.Get()
			cfg.ShowMapDebug = !cfg.ShowMapDebug
			cfg.ShowAIDebug = !cfg.ShowAIDebug
			cfg.NoEncounter = !cfg.NoEncounter
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		}).
		WithChoice("ゲーム開始メッセージ", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewGameStartMessageState}})
			return nil
		}).
		WithChoice("背景付きメッセージテスト", func(_ w.World) error {
			testMessage := messagedata.NewDialogMessage("これは背景付きメッセージのテストです。\nroom1.pngが背景に表示されています。", "システム")
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{
					func() es.State[w.World] {
						return NewMessageState(testMessage, WithBackgroundKey("bg", "hospital1"))
					},
				}})
			return nil
		}).
		WithChoice("オープニング", func(_ w.World) error {
			// 1ページ目: 町医者 - 診断
			page1 := &messagedata.MessageData{Speaker: "町医者"}
			page1.AddText("お母様の容態ですが...\n").
				AddText("残念ながら、").
				AddKeyword("地髄欠乏症").
				AddText("です。\n").
				AddText("現代でも治療の難しい病の1つとされています。")

			// 2ページ目: 町医者 - 闇医者を勧める
			page2 := &messagedata.MessageData{Speaker: "町医者"}
			page2.AddText("ただ、可能性はあります。\n西の果ての").
				AddKeyword("ハシマ").
				AddText("では").
				AddKeyword("地髄").
				AddText("研究が進んでいて、\n腕の立つ医者がいます。\n\n").
				AddText("訪ねてみるといいでしょう。")

			// 3ページ目: 場面転換 - ハシマへ
			page3 := &messagedata.MessageData{Speaker: ""}
			page3.AddText("* * *\n\n").
				AddText("ハシマの診療所 ----")

			// 4ページ目: 闇医者 - 街の説明
			page4 := &messagedata.MessageData{Speaker: "闇医者"}
			page4.AddText("よくここまで来たな。大変な街じゃろう。\n").
				AddText("どいつもこいつも一攫千金を目指してギラついておる。\n").
				AddText("まさに大陸のゴールドラッシュの様相じゃ。")

			// 5ページ目: 闇医者 - 治療法と費用
			page5 := &messagedata.MessageData{Speaker: "闇医者"}
			page5.AddText("さて、あんたの母親じゃが、\nわしが経験論的に編み出した手法によって\n治療できると確信しとる!\n\n").
				AddText("超高純度の精製地髄を投与して\nショックを与えれば、目覚める。\n\n").
				AddText("それに必要な地髄量は...").
				AddKeyword("1000万CZ").
				AddText("分だ。\n\n")

			// 6ページ目: 闇医者 - 採掘を勧める
			page6 := &messagedata.MessageData{Speaker: "闇医者"}
			page6.AddText("一般人が用意できる量ではないが、方法はある。\n\n").
				AddText("...地髄を直接採取するのだ。\n\n").
				AddKeyword("ハシマ").
				AddText("のすぐ近くに").
				AddKeyword("遺跡").
				AddText("がある。\n\n").
				AddText("深層まで潜れば必要な量が手に入るじゃろう...\n")

			// page1→page2（町医者シーン）
			page1.NextMessages = []*messagedata.MessageData{page2}

			// page3→page4→page5→page6（闇医者シーン）
			page3.NextMessages = []*messagedata.MessageData{page4}
			page4.NextMessages = []*messagedata.MessageData{page5}
			page5.NextMessages = []*messagedata.MessageData{page6}

			// page2終了時に闇医者シーンへ遷移
			page2.OnComplete = func() {
				messageState.SetTransition(es.Transition[w.World]{
					Type: es.TransPush,
					NewStateFuncs: []es.StateFactory[w.World]{
						func() es.State[w.World] {
							return NewMessageState(page3, WithBackgroundKey("bg", "hospital1"))
						}},
				})
			}

			// 最初に町医者シーンを表示
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{
					func() es.State[w.World] {
						return NewMessageState(page1, WithBackgroundKey("bg", "hospital2"))
					},
				}})
			return nil
		}).
		WithChoice("CZ収集エンディング", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewCZCollectionEndingState()},
			})
			return nil
		}).
		WithChoice("調停者エンディング", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransPush,
				NewStateFuncs: []es.StateFactory[w.World]{NewDungeonCompleteEndingState},
			})
			return nil
		}).
		WithChoice("次の階層に進む", func(world w.World) error {
			currentDepth := world.Resources.Dungeon.Depth

			// 次の階層に遷移
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{
					NewDungeonState(currentDepth + 1),
				},
			})
			return nil
		}).
		WithChoice("ステージ上部の橋にワープ", func(world w.World) error {
			// プレイヤーエンティティを取得
			player, err := worldhelper.GetPlayerEntity(world)
			if err != nil {
				return fmt.Errorf("プレイヤーが見つかりません: %w", err)
			}

			// GridElementコンポーネントを取得
			gridComp := world.Components.GridElement.Get(player)
			if gridComp == nil {
				return fmt.Errorf("プレイヤーのGridElementコンポーネントが見つかりません")
			}
			grid := gridComp.(*gc.GridElement)

			// ポータルエンティティを検索
			var targetX, targetY gc.Tile
			found := false

			world.Manager.Join(
				world.Components.Interactable,
				world.Components.GridElement,
			).Visit(ecs.Visit(func(entity ecs.Entity) {
				interactable := world.Components.Interactable.Get(entity).(*gc.Interactable)
				_, ok := interactable.Data.(gc.PortalInteraction)
				if !ok {
					return
				}

				// ポータルを発見
				portalGrid := world.Components.GridElement.Get(entity).(*gc.GridElement)
				targetX = portalGrid.X
				targetY = portalGrid.Y
				found = true
			}))

			if !found {
				return fmt.Errorf("ポータルが見つかりません")
			}

			// プレイヤーの座標を更新（ポータルの1マス下にワープ）
			grid.X = targetX
			grid.Y = targetY + 1

			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransPop,
			})
			return nil
		}).
		WithChoice("閉じる", func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{
				Type: es.TransPop,
			})
			return nil
		})

	return messageState
}

// DungeonStateOption はDungeonStateのオプション設定関数
type DungeonStateOption func(*DungeonState)

// WithSeed はシード値を設定するオプション
func WithSeed(seed uint64) DungeonStateOption {
	return func(ds *DungeonState) {
		ds.Seed = &seed
	}
}

// WithBuilderType はマップビルダータイプを設定するオプション
func WithBuilderType(builderType mapplanner.PlannerType) DungeonStateOption {
	return func(ds *DungeonState) {
		ds.BuilderType = builderType
	}
}

// NewDungeonState はDungeonStateインスタンスを作成するファクトリー関数
// デフォルトではBuilderTypeはPlannerTypeRandomになる
func NewDungeonState(depth int, opts ...DungeonStateOption) es.StateFactory[w.World] {
	return func() es.State[w.World] {
		ds := &DungeonState{
			Depth:       depth,
			BuilderType: mapplanner.PlannerTypeRandom,
		}
		for _, opt := range opts {
			opt(ds)
		}
		return ds
	}
}

// NewTownState は街のステートを作成するファクトリー関数
// 街は常に深度0、BuilderTypeはTown固定
func NewTownState(opts ...DungeonStateOption) es.StateFactory[w.World] {
	allOpts := make([]DungeonStateOption, 0, 1+len(opts))
	allOpts = append(allOpts, WithBuilderType(mapplanner.PlannerTypeTown))
	allOpts = append(allOpts, opts...)

	return NewDungeonState(0, allOpts...)
}

// NewMainMenuState は新しいMainMenuStateインスタンスを作成するファクトリー関数
func NewMainMenuState() es.State[w.World] {
	return &MainMenuState{}
}

// NewGameOverMessageState はゲームオーバー用のMessageStateを作成するファクトリー関数
func NewGameOverMessageState() es.State[w.World] {
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

	return messageState
}

// NewDungeonCompleteEndingState はダンジョン踏破エンディングのStateを作成するファクトリー関数
func NewDungeonCompleteEndingState() es.State[w.World] {
	messageState := &MessageState{}

	// ゲームクリアメッセージを作成（選択肢付き）
	messageData := messagedata.NewSystemMessage(`地中から大気まで濃密な地髄に溢れた。
長い眠りについていた人々は目覚めだした。

不毛だった大地には風が吹き、若草が芽吹きだした。

人々は忘れかけた希望の感覚に
酔いしれるのであった...。

----------
[TRUE END]
----------`).
		WithChoice("閉じる", func(_ w.World) error {
			// 町に遷移
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransReplace,
				NewStateFuncs: []es.StateFactory[w.World]{NewTownState()},
			})
			return nil
		})

	// MessageStateにMessageDataを設定
	messageState.messageData = messageData

	return messageState
}

// NewCZCollectionEndingState はCZ収集エンディングのStateFactoryを作成する
func NewCZCollectionEndingState() es.StateFactory[w.World] {
	return func() es.State[w.World] {
		// 1ページ目: 治療費を集めた主人公
		ending1 := &messagedata.MessageData{Speaker: ""}
		ending1.AddText(
			`遺跡への潜行を繰り返し、

ついに1000万CZを集めた。`)

		// 2ページ目: 医師の反応
		ending2 := &messagedata.MessageData{Speaker: "医師"}
		ending2.AddText(
			`「...本当に集めてきたのですか。

これだけの高純度地髄を、こんな短期間で...。」`)

		// 3ページ目: 治療開始
		ending3 := &messagedata.MessageData{Speaker: "医師"}
		ending3.AddText(
			`すぐに治療を始めます。
地髄の精製と投与には時間がかかりますが、

お母さんは必ず目を覚ますでしょう。`)

		// 4ページ目: 回復
		ending4 := &messagedata.MessageData{Speaker: ""}
		ending4.AddText(
			`数日後...

母は目を覚ました。
虚ろに落ちる前の、穏やかな表情で。`)

		// 5ページ目: エンディング
		ending5 := &messagedata.MessageData{Speaker: ""}
		ending5.AddText(
			`命を賭けた潜行の日々は終わった。

しかし、遺跡の最深部には
まだ誰も到達していない。

いつかまた、あの場所に戻る日が
来るかもしれない。

------------
[NORMAL END]
------------`)

		messageState := NewMessageState(ending1, WithBackgroundKey("bg", "hospital1"))

		// 最後のメッセージに街への遷移を追加
		ending5.WithChoice("閉じる", func(_ w.World) error {
			if ms, ok := messageState.(*MessageState); ok {
				ms.SetTransition(es.Transition[w.World]{
					Type:          es.TransReplace,
					NewStateFuncs: []es.StateFactory[w.World]{NewTownState()},
				})
			}
			return nil
		})

		ending1.NextMessages = []*messagedata.MessageData{ending2}
		ending2.NextMessages = []*messagedata.MessageData{ending3}
		ending3.NextMessages = []*messagedata.MessageData{ending4}
		ending4.NextMessages = []*messagedata.MessageData{ending5}

		return messageState
	}
}

// NewGameStartMessageState はゲーム開始時の目的を説明するMessageStateを作成するファクトリー関数
func NewGameStartMessageState() es.State[w.World] {
	messageState := &MessageState{}

	// メッセージを作成
	messageData := messagedata.NewDialogMessage(`「あんた、遺跡の『珠狙い』だろ?
外からこの街に来る異常に若い連中はみんなそうさ。
向こう見ずで破滅的で、...
どうしようもない事情を持ってる。」

「あんたは...、そうか、母親が...。
言っちゃ悪いが、そういう奴らはここでは珍しくない。
どんな事情があるにせよ、遺跡で辿る結末は1つさ。」`, "老兵")

	// MessageStateにMessageDataを設定
	messageState.messageData = messageData

	return messageState
}

// NewSaveMenuState は新しいSaveMenuStateインスタンスを作成するファクトリー関数
func NewSaveMenuState() es.State[w.World] {
	messageState := &MessageState{}

	// セーブマネージャーで現在のスロット状態を取得
	saveManager := save.NewSerializationManager("./saves")

	messageData := messagedata.NewSystemMessage("")

	// 各スロットの状態を確認して選択肢を動的に生成
	for i := 1; i <= 3; i++ {
		slotName := fmt.Sprintf("slot%d", i)
		var label string

		if saveManager.SaveFileExists(slotName) {
			if timestamp, err := saveManager.GetSaveFileTimestamp(slotName); err == nil {
				label = fmt.Sprintf("スロット%d [%s]", i, timestamp.Format("01/02 15:04"))
			} else {
				label = fmt.Sprintf("スロット%d [データあり]", i)
			}
		} else {
			label = fmt.Sprintf("スロット%d [空]", i)
		}

		messageData = messageData.WithChoice(label, func(world w.World) error {
			if err := saveManager.SaveWorld(world, slotName); err != nil {
				return fmt.Errorf("save failed: %w", err)
			}
			// セーブ後は同じセーブメニューを再作成してメニューを維持
			messageState.SetTransition(es.Transition[w.World]{
				Type:          es.TransSwitch,
				NewStateFuncs: []es.StateFactory[w.World]{NewSaveMenuState}})
			return nil
		})
	}

	messageData = messageData.WithChoice("戻る", func(_ w.World) error {
		messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		return nil
	})

	messageState.messageData = messageData
	return messageState
}

// NewLoadMenuState は新しいLoadMenuStateインスタンスを作成するファクトリー関数
func NewLoadMenuState() es.State[w.World] {
	messageState := &MessageState{}

	// セーブマネージャーで現在のスロット状態を取得
	saveManager := save.NewSerializationManager("./saves")

	messageData := messagedata.NewSystemMessage("")

	// 各スロットの状態を確認して選択肢を動的に生成
	hasValidSlot := false
	for i := 1; i <= 3; i++ {
		slotName := fmt.Sprintf("slot%d", i)
		var label string

		if saveManager.SaveFileExists(slotName) {
			hasValidSlot = true
			if timestamp, err := saveManager.GetSaveFileTimestamp(slotName); err == nil {
				label = fmt.Sprintf("スロット%d [%s]", i, timestamp.Format("01/02 15:04"))
			} else {
				label = fmt.Sprintf("スロット%d [データあり]", i)
			}

			messageData = messageData.WithChoice(label, func(world w.World) error {
				// ロードを実行
				err := saveManager.LoadWorld(world, slotName)
				if err != nil {
					println("Load failed:", err.Error())
					messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
					return err
				}
				// 遷移（街マップを生成してプレイヤーを配置）
				stateFactory := NewTownState()
				messageState.SetTransition(es.Transition[w.World]{
					Type:          es.TransReplace,
					NewStateFuncs: []es.StateFactory[w.World]{stateFactory}})
				return nil
			})
		}
	}

	// 有効なセーブデータが存在しない場合の処理
	if !hasValidSlot {
		messageData = messageData.WithChoice("セーブデータがありません", func(_ w.World) error {
			// 何もしない（選択不可を示すためのダミー選択肢）
			return nil
		})
	}

	messageData = messageData.WithChoice("戻る", func(_ w.World) error {
		messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		return nil
	})

	// MessageStateにMessageDataを設定
	messageState.messageData = messageData

	return messageState
}

// NewMessageState はメッセージデータを受け取って新しいMessageStateを作成するファクトリー関数
// オプションを指定することで背景画像などをカスタマイズできる
func NewMessageState(messageData *messagedata.MessageData, opts ...MessageStateOption) es.State[w.World] {
	return &MessageState{
		messageData: messageData,
		options:     opts,
	}
}

// NewShopMenuState は新しいShopMenuStateインスタンスを作成するファクトリー関数
func NewShopMenuState() es.State[w.World] {
	return &ShopMenuState{}
}

// NewInteractionMenuState はインタラクションメニューStateを作成する
func NewInteractionMenuState(world w.World) es.State[w.World] {
	messageState := &MessageState{}

	// プレイヤー周辺の実行可能なアクションを取得
	interactionActions := GetInteractionActions(world)

	if len(interactionActions) == 0 {
		// アクションがない場合
		messageState.messageData = messagedata.NewSystemMessage("実行可能なアクションがありません。")
		return messageState
	}

	// アクションメニューを構築
	messageState.messageData = messagedata.NewSystemMessage("")

	for _, action := range interactionActions {
		// TODO(kijima): ループ変数の対策は不要になったので消す
		// クロージャで変数をキャプチャ
		capturedAction := action
		messageState.messageData = messageState.messageData.WithChoice(capturedAction.Label, func(world w.World) error {
			// プレイヤーエンティティの取得
			playerEntity, err := worldhelper.GetPlayerEntity(world)
			if err != nil {
				return fmt.Errorf("failed to get player entity: %w", err)
			}

			// アクションを実行
			params := actions.ActionParams{
				Actor:  playerEntity,
				Target: &capturedAction.Target,
			}
			if err := executeActivity(world, capturedAction.Activity, params); err != nil {
				return fmt.Errorf("アクション実行失敗: %w", err)
			}

			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		})
	}

	// キャンセル用の「閉じる」選択肢を追加
	messageState.messageData = messageState.messageData.WithChoice("キャンセル", func(_ w.World) error {
		messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		return nil
	})

	return messageState
}

// NewMerchantDialogState は商人との会話ステートを作成
func NewMerchantDialogState(speakerName string) es.State[w.World] {
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

	return persistentState
}

// NewDoctorDialogState は怪しい科学者との会話ステートを作成
func NewDoctorDialogState(speakerName string) es.State[w.World] {
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

	return persistentState
}

// NewDarkDoctorDialogState は闇医者との会話ステートを作成
func NewDarkDoctorDialogState(speakerName string, world w.World) es.State[w.World] {
	persistentState := NewPersistentMessageState(nil)

	// プレイヤーの所持金を確認
	player, _ := worldhelper.GetPlayerEntity(world)
	requiredAmount := 10000000 // 1000万CZ
	hasEnoughMoney := worldhelper.HasCurrency(world, player, requiredAmount)

	persistentState.messageData = messagedata.NewDialogMessage("", speakerName).
		AddText(`治療費1000万CZを用意できたかい?`)

	// 1000万CZ以上持っている場合のみ「はい」選択肢を表示
	if hasEnoughMoney {
		persistentState.messageData = persistentState.messageData.WithChoice("はい", func(world w.World) error {
			// 通貨を消費
			if worldhelper.ConsumeCurrency(world, player, requiredAmount) {
				persistentState.SetTransition(es.Transition[w.World]{
					Type:          es.TransSwitch,
					NewStateFuncs: []es.StateFactory[w.World]{NewCZCollectionEndingState()},
				})
			}
			return nil
		})
	}

	persistentState.messageData = persistentState.messageData.WithChoice("まだだ", func(_ w.World) error {
		persistentState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		return nil
	})

	return persistentState
}
