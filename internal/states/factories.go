package states

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/activity"
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

func dungeonMenuChoices(_ w.World) (string, []Choice) {
	return "", []Choice{
		{Label: "所持", Run: pushChoice(NewItemActionState(verbExamine))},
		{Label: "部隊", Run: pushChoice(NewCharacterState)},
		{Label: "書込", Run: pushChoice(NewSaveMenuState)},
		{Label: "終了", Run: func(_ w.World) (es.Transition[w.World], error) {
			return es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: []es.StateFactory[w.World]{NewMainMenuState}}, nil
		}},
		{Label: TextClose, Run: func(_ w.World) (es.Transition[w.World], error) {
			return es.Transition[w.World]{Type: es.TransPop}, nil
		}},
	}
}

// NewCraftMenuState は新しいCraftMenuStateインスタンスを作成するファクトリー関数
func NewCraftMenuState() (es.State[w.World], error) {
	return &CraftMenuState{}, nil
}

// NewCharacterState は画面タブメニューのStateを作成するファクトリー関数。主人公から始まる
func NewCharacterState() (es.State[w.World], error) {
	return &CharacterState{}, nil
}

// NewCharacterStateForMember は指定メンバーを表示対象にした画面タブメニューを作成する。
// 画面内で切り替えて主人公や他の仲間も見られる
func NewCharacterStateForMember(member ecs.Entity) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &CharacterState{target: member}, nil
	}
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
		RunSeed: world.Config.RNG.Uint64(),
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
		return nil, fmt.Errorf("セーブマネージャーの作成に失敗: %w", err)
	}
	choices := make([]Choice, 0, 5)
	for i := 1; i <= 4; i++ {
		slotName := fmt.Sprintf("slot%d", i)
		choices = append(choices, Choice{Label: formatSaveSlotLabel(saveManager, slotName), Run: func(world w.World) (es.Transition[w.World], error) {
			if err := saveManager.SaveWorld(world, slotName); err != nil {
				return es.Transition[w.World]{}, fmt.Errorf("save failed: %w", err)
			}
			// 保存後はメニューを開き直してラベルを更新する
			return es.Transition[w.World]{Type: es.TransSwitch, NewStateFuncs: []es.StateFactory[w.World]{NewSaveMenuState}}, nil
		}})
	}
	choices = append(choices, backChoice())
	return NewChoiceMenu(func(_ w.World) (string, []Choice) { return "セーブ", choices }), nil
}

// NewLoadMenuState はロード画面を作成するファクトリー関数。
// 手動4スロットとオートセーブ4スロットをセクション分けで表示する。
func NewLoadMenuState() (es.State[w.World], error) {
	saveManager, err := save.NewSerializationManager()
	if err != nil {
		return nil, fmt.Errorf("セーブマネージャーの作成に失敗: %w", err)
	}
	choices := make([]Choice, 0, 12)
	choices = append(choices, Choice{Label: "手動セーブ", Header: true})
	for i := 1; i <= 4; i++ {
		choices = append(choices, loadSlotChoice(saveManager, fmt.Sprintf("slot%d", i)))
	}
	choices = append(choices, Choice{Label: "オートセーブ", Header: true})
	autoSaves, err := saveManager.ListAutoSaves()
	if err != nil {
		return nil, fmt.Errorf("オートセーブ一覧の取得に失敗: %w", err)
	}
	if len(autoSaves) > 4 {
		autoSaves = autoSaves[:4]
	}
	for i := range 4 {
		if i < len(autoSaves) {
			choices = append(choices, loadSlotChoice(saveManager, autoSaves[i]))
		} else {
			choices = append(choices, Choice{Label: "  ---", Header: true})
		}
	}
	choices = append(choices, backChoice())
	return NewChoiceMenu(func(_ w.World) (string, []Choice) { return "ロード", choices }), nil
}

// backChoice は戻る選択肢を返す。選択メニューで共通に使う
func backChoice() Choice {
	return Choice{Label: "戻る", Run: func(_ w.World) (es.Transition[w.World], error) {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}}
}

// loadSlotChoice はロードスロット1つ分の選択肢を返す。空スロットは選べない見出し行にする
func loadSlotChoice(saveManager *save.SerializationManager, slotName string) Choice {
	if !saveManager.SaveFileExists(slotName) {
		return Choice{Label: "  ---", Header: true}
	}
	return Choice{Label: formatSaveSlotLabel(saveManager, slotName), Run: func(world w.World) (es.Transition[w.World], error) {
		if err := saveManager.LoadWorld(world, slotName); err != nil {
			// ロード失敗はアプリ全体を落とさない。RestoreWorldFromJSON の probe 検証で本番ワールドは
			// 無傷なので、エラーはログに残してメニューへ戻るだけにする。ゲームループへ返すと
			// main の log.Fatal まで波及してプロセスごと落ちてしまう
			logger.New(logger.CategorySave).Error("セーブのロードに失敗した", "slot", slotName, "error", err.Error())
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

// NewTavernMenuState は酒場の雇用画面のStateを作成するファクトリー関数
func NewTavernMenuState() (es.State[w.World], error) {
	return &TavernMenuState{}, nil
}

// NewShopMenuState は新しいShopMenuStateインスタンスを作成するファクトリー関数
func NewShopMenuState() (es.State[w.World], error) {
	return &ShopMenuState{}, nil
}

// NewStorageMenuState は収納メニューStateを作成する
func NewStorageMenuState(storageEntity ecs.Entity) (es.State[w.World], error) {
	return &StorageMenuState{storageEntity: storageEntity}, nil
}

// NewInteractionMenuState はインタラクションメニューStateを作成する
func NewInteractionMenuState(world w.World) (es.State[w.World], error) {
	if len(GetInteractionActions(world)) == 0 {
		messageState := &MessageState{}
		messageState.messageData = messagedata.NewSystemMessage("実行可能なアクションがありません。")
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
				return es.Transition[w.World]{}, fmt.Errorf("プレイヤーの取得に失敗: %w", err)
			}
			if _, err := activity.ExecuteInteraction(playerEntity, action.Target, action.Interaction, world); err != nil {
				return es.Transition[w.World]{}, fmt.Errorf("アクション実行失敗: %w", err)
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
