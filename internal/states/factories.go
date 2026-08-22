package states

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/activity"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/logger"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 各ステートのファクトリー関数を集約したファイル

// NewDungeonMenuState はダンジョンメニューを選択メニューとして作る
func NewDungeonMenuState() (es.State[w.World], error) {
	return NewChoiceMenu(dungeonMenuChoices), nil
}

func dungeonMenuChoices(world w.World) (string, []Choice) {
	return "", []Choice{
		{Label: query.T(world, "Inventory"), Run: pushChoice(NewItemActionState(verbExamine))},
		{Label: query.T(world, "Character"), Run: pushChoice(NewCharacterState)},
		{Label: query.T(world, "Save game"), Run: pushChoice(NewSaveMenuState)},
		{Label: query.T(world, "Quit"), Run: func(_ w.World) (es.Transition[w.World], error) {
			return es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: []es.StateFactory[w.World]{NewMainMenuState}}, nil
		}},
		{Label: query.T(world, "Close"), Run: func(_ w.World) (es.Transition[w.World], error) {
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}},
	}
}

// NewCraftMenuState は新しいCraftMenuStateインスタンスを作成するファクトリー関数
func NewCraftMenuState() (es.State[w.World], error) {
	return &CraftMenuState{}, nil
}

// NewCharacterState は画面タブメニューのStateを作成するファクトリー関数。主人公を表示する
func NewCharacterState() (es.State[w.World], error) {
	return &CharacterState{}, nil
}

// DungeonStateOption はDungeonStateのオプション設定関数
type DungeonStateOption func(*DungeonState)

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

// newGameOverworldState は新規ゲーム開始用のオーバーワールド探索ステートを返す。
// 街を含むオーバーワールドを RunSeed から決定的生成し、プレイヤーは街から始まる。
// キャラ作成・デモ・デバッグ開始で共通に使い、開始点を1箇所に集約する。RunSeed は都度引く。
// 帯形状はマスタ DungeonOverworld が持つので、プレイ固有の RunSeed だけを渡す。
func newGameOverworldState(world w.World) es.StateFactory[w.World] {
	return NewOverworldState(mapplanner.PlannerTypeOverworldField, dungeon.DungeonOverworld, &overworld.NewGameParams{
		RunSeed: world.Resources.Config.RNG.Uint64(),
	})
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

	// ゲームオーバーメッセージを作成する。翻訳は world を要するので OnStart で構築する
	messageState.build = func(world w.World) *messagedata.MessageData {
		return messagedata.NewSystemMessage(query.T(world, "You died.")).
			WithChoice(query.T(world, "Return to main menu"), func(_ w.World) error {
				messageState.SetTransition(es.Transition[w.World]{
					Type:          es.TransReplace,
					NewStateFuncs: []es.StateFactory[w.World]{NewMainMenuState}})
				return nil
			})
	}

	return messageState, nil
}

// NewRunResultState は run の死の結果画面を作成するファクトリー関数。
// RunOutcome と RunStats を読み、死因とスコアと統計を表示する
func NewRunResultState() (es.State[w.World], error) {
	messageState := &MessageState{}

	messageState.build = func(world w.World) *messagedata.MessageData {
		var dist, days, turns int
		if o := query.GetRunOutcome(world); o != nil {
			dist = o.ReachedDist
			days = o.Days
			turns = o.Turns
		}
		var kills, items int
		var sales consts.Currency
		if s := query.GetRunStats(world); s != nil {
			kills = s.EnemiesKilled
			items = s.ItemsScavenged
			sales = s.SalesTotal
		}
		text := query.T(world, "You died.") + "\n\n" +
			query.T(world, "Distance reached: %d", dist) + "\n" +
			query.T(world, "Days: %d", days) + "\n" +
			query.T(world, "Turns: %d", turns) + "\n" +
			query.T(world, "Enemies killed: %d", kills) + "\n" +
			query.T(world, "Items scavenged: %d", items) + "\n" +
			query.T(world, "Sales: %d", sales)
		return messagedata.NewSystemMessage(text).
			WithChoice(query.T(world, "Return to main menu"), func(_ w.World) error {
				messageState.SetTransition(es.Transition[w.World]{
					Type:          es.TransReplace,
					NewStateFuncs: []es.StateFactory[w.World]{NewMainMenuState}})
				return nil
			})
	}

	return messageState, nil
}

// NewAllClearEventState は全ダンジョンクリア時のイベントStateを作成するファクトリー関数
func NewAllClearEventState() (es.State[w.World], error) {
	messageState := &MessageState{}

	messageState.build = func(world w.World) *messagedata.MessageData {
		return messagedata.NewSystemMessage(query.T(world, "You conquered all the ruins.\n\nThe ancient presence sleeping at the bottom of the great hole has finally quieted.")).
			WithChoice(query.T(world, "Close"), func(_ w.World) error {
				messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
				return nil
			})
	}
	return messageState, nil
}

// NewSaveMenuState は手動セーブ画面を作成するファクトリー関数。
// 固定4スロットで、主人公名とタイムスタンプを表示する。
func NewSaveMenuState() (es.State[w.World], error) {
	saveManager, err := save.NewSerializationManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create save manager: %w", err)
	}
	// スロットラベルはファイル IO を伴うので初回 Fetch で一度だけ組んでキャッシュする。
	// world は Fetch から渡る。保存後は TransSwitch でステートを開き直して再構築する
	var choices []Choice
	return NewChoiceMenu(func(world w.World) (string, []Choice) {
		if choices == nil {
			for i := 1; i <= 4; i++ {
				slotName := fmt.Sprintf("slot%d", i)
				choices = append(choices, Choice{Label: formatSaveSlotLabel(world, saveManager, slotName), Run: func(world w.World) (es.Transition[w.World], error) {
					if err := saveManager.SaveWorld(world, slotName); err != nil {
						return es.Transition[w.World]{}, fmt.Errorf("save failed: %w", err)
					}
					return es.Transition[w.World]{Type: es.TransSwitch, NewStateFuncs: []es.StateFactory[w.World]{NewSaveMenuState}}, nil
				}})
			}
			choices = append(choices, backChoice(world))
		}
		return query.T(world, "Save"), choices
	}), nil
}

// NewLoadMenuState はロード画面を作成するファクトリー関数。
// 手動4スロットとオートセーブ4スロットをセクション分けで表示する。
func NewLoadMenuState() (es.State[w.World], error) {
	saveManager, err := save.NewSerializationManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create save manager: %w", err)
	}
	// ラベルはファイル IO を伴うので初回 Fetch で一度だけ組んでキャッシュする
	var choices []Choice
	return NewChoiceMenu(func(world w.World) (string, []Choice) {
		if choices == nil {
			choices = append(choices, Choice{Label: query.T(world, "Manual save"), Header: true})
			for i := 1; i <= 4; i++ {
				choices = append(choices, loadSlotChoice(world, saveManager, fmt.Sprintf("slot%d", i)))
			}
			choices = append(choices, Choice{Label: query.T(world, "Auto save"), Header: true})
			autoSaves, err := saveManager.ListAutoSaves()
			if err != nil {
				logger.New(logger.CategorySave).Error("failed to list auto saves", "error", err.Error())
				autoSaves = nil
			}
			if len(autoSaves) > 4 {
				autoSaves = autoSaves[:4]
			}
			for i := range 4 {
				if i < len(autoSaves) {
					choices = append(choices, loadSlotChoice(world, saveManager, autoSaves[i]))
				} else {
					choices = append(choices, Choice{Label: "  ---", Header: true})
				}
			}
			choices = append(choices, backChoice(world))
		}
		return query.T(world, "Load game"), choices
	}), nil
}

// backChoice は戻る選択肢を返す。選択メニューで共通に使う
func backChoice(world w.World) Choice {
	return Choice{Label: query.T(world, "Back"), Run: func(_ w.World) (es.Transition[w.World], error) {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}}
}

// loadSlotChoice はロードスロット1つ分の選択肢を返す。空スロットは選べない見出し行にする
func loadSlotChoice(world w.World, saveManager *save.SerializationManager, slotName string) Choice {
	if !saveManager.SaveFileExists(slotName) {
		return Choice{Label: "  ---", Header: true}
	}
	return Choice{Label: formatSaveSlotLabel(world, saveManager, slotName), Run: func(world w.World) (es.Transition[w.World], error) {
		if err := saveManager.LoadWorld(world, slotName); err != nil {
			// ロード失敗はアプリ全体を落とさない。RestoreWorldFromJSON の probe 検証で本番ワールドは
			// 無傷なので、エラーはログに残してメニューへ戻るだけにする。ゲームループへ返すと
			// main の log.Fatal まで波及してプロセスごと落ちてしまう
			logger.New(logger.CategorySave).Error("failed to load save", "slot", slotName, "error", err.Error())
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}
		// 復元済みの現在地から再生成せずに復帰する
		return es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: []es.StateFactory[w.World]{newResumeStateFactory(world)}}, nil
	}}
}

// newResumeStateFactory はロード復元時の復帰先ステートを保存内容から選ぶ。
// 現ステージが帯データを持つ、すなわちオーバーワールドなら OverworldState で復帰して帯を
// 再構築し、通常ダンジョンなら DungeonState で復帰する。定義名・深度から再生成はしない。
func newResumeStateFactory(world w.World) es.StateFactory[w.World] {
	if query.IsOnOverworld(world) {
		// ロード復元。帯形状は SeamlessBand から復元するので params は nil。種別はマスタを渡す
		return NewOverworldState(mapplanner.PlannerTypeOverworldField, dungeon.DungeonOverworld, nil)
	}
	d := query.GetDungeon(world)
	return NewDungeonState(d.CurrentStage.Depth, WithDefinitionName(d.CurrentStage.Name), WithResume())
}

// formatSaveSlotLabel はセーブスロットの表示ラベルを生成する。
// データがあればプレイヤー名と日時を、無ければダッシュを返す。
func formatSaveSlotLabel(world w.World, saveManager *save.SerializationManager, slotName string) string {
	if !saveManager.SaveFileExists(slotName) {
		return "---"
	}

	playerName, nameErr := saveManager.GetSavePlayerName(slotName)
	timestamp, tsErr := saveManager.GetSaveFileTimestamp(slotName)

	if nameErr == nil && tsErr == nil {
		return fmt.Sprintf("  %s  %s", playerName, timestamp.Format("01/02 15:04"))
	}
	return query.T(world, "  Has data")
}

// NewMessageState は組み立て済みメッセージから MessageState を作成する。
// 構築を build へ一本化するため、受け取った messageData を返すだけの build で包む
func NewMessageState(messageData *messagedata.MessageData) (es.State[w.World], error) {
	return &MessageState{
		build: func(_ w.World) *messagedata.MessageData { return messageData },
	}, nil
}

// NewShopMenuState は新しいShopMenuStateインスタンスを作成するファクトリー関数。
// merchant はこの店の在庫を持つ商人。売買はこの商人の収納を出し入れする
func NewShopMenuState(merchant ecs.Entity) (es.State[w.World], error) {
	return &ShopMenuState{merchant: merchant}, nil
}

// NewStorageMenuState は収納メニューStateを作成する
func NewStorageMenuState(storageEntity ecs.Entity) (es.State[w.World], error) {
	return &StorageMenuState{storageEntity: storageEntity}, nil
}

// NewAuctionMenuState は出荷場所のメニューStateを作成する
func NewAuctionMenuState(stationEntity ecs.Entity) (es.State[w.World], error) {
	return &AuctionMenuState{stationEntity: stationEntity}, nil
}

// NewInteractionMenuState はインタラクションメニューStateを作成する
func NewInteractionMenuState(world w.World) (es.State[w.World], error) {
	if len(GetInteractionActions(world)) == 0 {
		messageState := &MessageState{}
		messageState.build = func(world w.World) *messagedata.MessageData {
			return messagedata.NewSystemMessage(query.T(world, "No actions available."))
		}
		return messageState, nil
	}
	return NewChoiceMenu(interactionChoices), nil
}

// interactionActionChoices は交流アクション列を選択肢へ変換する。選ぶと実行してダンジョンへ戻る。
// キャンセルは選択肢でなく Esc で閉じる。他メニューと操作を揃える
func interactionActionChoices(actions []InteractionAction) []Choice {
	choices := make([]Choice, 0, len(actions))
	for _, action := range actions {
		choices = append(choices, Choice{Label: action.Label, Run: func(world w.World) (es.Transition[w.World], error) {
			playerEntity, err := query.GetPlayerEntity(world)
			if err != nil {
				return es.Transition[w.World]{}, fmt.Errorf("failed to get player: %w", err)
			}
			if _, err := activity.ExecuteInteraction(playerEntity, action.Target, action.Interaction, world); err != nil {
				return es.Transition[w.World]{}, fmt.Errorf("failed to execute action: %w", err)
			}
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}})
	}
	return choices
}

// interactionChoices は範囲内の交流アクションを選択肢にする。交流メニューで使う
func interactionChoices(world w.World) (string, []Choice) {
	return "", interactionActionChoices(GetInteractionActions(world))
}

// sameTileActionChoices は足元タイルの手動アクションを選択肢にする。ダンジョンの相互作用キーで使う
func sameTileActionChoices(world w.World) (string, []Choice) {
	return "", interactionActionChoices(GetSameTileManualActions(world))
}

// NewMerchantDialogState は商人との会話ステートを作成。merchant はこの商人の実体で、店を開くとき在庫の持ち主として渡す
func NewMerchantDialogState(speakerName string, merchant ecs.Entity) (es.State[w.World], error) {
	persistentState := &PersistentMessageState{}

	persistentState.build = func(world w.World) *messagedata.MessageData {
		return messagedata.NewDialogMessage("", speakerName).
			AddText(query.T(world, "Want to make a deal?\n\nI've got good stuff.")).
			WithChoice(query.T(world, "Look"), func(_ w.World) error {
				persistentState.SetTransition(es.Transition[w.World]{
					Type: es.TransPush,
					NewStateFuncs: []es.StateFactory[w.World]{
						func() (es.State[w.World], error) { return NewShopMenuState(merchant) },
					},
				})
				return nil
			}).
			WithChoice(query.T(world, "No business"), func(_ w.World) error {
				persistentState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
				return nil
			})
	}

	return persistentState, nil
}

// NewDoctorDialogState は怪しい科学者との会話ステートを作成
func NewDoctorDialogState(speakerName string) (es.State[w.World], error) {
	persistentState := &PersistentMessageState{}

	persistentState.build = func(world w.World) *messagedata.MessageData {
		return messagedata.NewDialogMessage("", speakerName).
			AddText(query.T(world, "Heh heh... I'll reconstruct matter with my secret technique.\n\nBring me core and materials!")).
			WithChoice(query.T(world, "I want to craft"), func(_ w.World) error {
				persistentState.SetTransition(es.Transition[w.World]{
					Type:          es.TransPush,
					NewStateFuncs: []es.StateFactory[w.World]{NewCraftMenuState},
				})
				return nil
			}).
			WithChoice(query.T(world, "No business"), func(_ w.World) error {
				persistentState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
				return nil
			})
	}

	return persistentState, nil
}

// NewOpeningState はオープニングを表示するStateを作成するファクトリー関数
// 完了後はポップする。後続ステートが必要な場合は呼び出し側でスタックに積む
func NewOpeningState() (es.State[w.World], error) {
	messageState := &MessageState{}
	// 翻訳は world を要するので OnStart でページを組む
	messageState.build = func(world w.World) *messagedata.MessageData {
		// 1. 黒背景: 荒野の大穴
		page1a := &messagedata.MessageData{Speaker: "", BackgroundKey: "black1"}
		page1a.AddText(query.T(world, "A vast wasteland, with a single great hole gaping open."))

		// 2. 穴背景: 空ページ（背景だけ見せる）→ 遺跡の説明
		blank := &messagedata.MessageData{Speaker: "", BackgroundKey: "hole1"}
		page1b := &messagedata.MessageData{Speaker: ""}
		page1b.AddText(query.T(world, "At the bottom of the hole lie ruins of an ancient civilization.\n")).
			AddText(query.T(world, "Treasure comes out. Monsters too. Half of those who dive never return.\n")).
			AddText(query.T(world, "Around the hole a town has formed of divers, sellers, and buyers."))

		// 3. 酒場背景: 空ページ（背景だけ見せる）→ 拾い屋の噂
		blankBar := &messagedata.MessageData{Speaker: "", BackgroundKey: "bar1"}
		page2 := &messagedata.MessageData{Speaker: ""}
		page2.AddText(query.T(world, "\"Heard about it? Another bottom-seeker vanished.\"\n")).
			AddText(query.T(world, "\"How many is that now?\"\n")).
			AddText(query.T(world, "\"Who knows. I quit counting long ago.\"\n\n")).
			AddMarkup(query.T(world, "\"So, the next <keyword>scavenger</keyword> has shown up... a bottom-seeker too, they say.\"\n"))

		return messagedata.ChainMessages(page1a, blank, page1b, blankBar, page2)
	}
	return messageState, nil
}
