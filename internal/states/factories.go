package states

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/logger"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 各ステートのファクトリー関数を集約したファイル

// pushChoice は指定ファクトリの state を push する Choice.Run を返す。選択メニューの共通部品
func pushChoice(factory es.StateFactory[w.World]) func(w.World) (es.Transition[w.World], error) {
	return func(_ w.World) (es.Transition[w.World], error) {
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{factory}}, nil
	}
}

// NewDungeonMenuState はダンジョンメニューを選択メニューとして作る
func NewDungeonMenuState() (es.State[w.World], error) {
	return NewChoiceMenu(dungeonMenuChoices), nil
}

// dungeonMenuChoices はダンジョンメニューの選択肢を返す
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

// popAfter は fn を実行して閉じる Choice.Run を返す
func popAfter(fn func(w.World) error) func(w.World) (es.Transition[w.World], error) {
	return func(world w.World) (es.Transition[w.World], error) {
		if err := fn(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
}

// stayAfter は fn を実行してメニューに留まる Choice.Run を返す。敵やPropの連続スポーンに使う
func stayAfter(fn func(w.World) error) func(w.World) (es.Transition[w.World], error) {
	return func(world w.World) (es.Transition[w.World], error) {
		return es.Transition[w.World]{Type: es.TransNone}, fn(world)
	}
}

// pushMessage は指定メッセージ画面を push する Choice.Run を返す
func pushMessage(md *messagedata.MessageData) func(w.World) (es.Transition[w.World], error) {
	return pushChoice(func() (es.State[w.World], error) { return NewMessageState(md) })
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

	// 街用NPC・収納箱を配置したデバッグステージへ入る。街の会話・売買・収納を1画面でテストできる
	choices = append(choices, Choice{Label: "デバッグステージ:街用NPC+収納箱", Run: popAfter(func(world w.World) error {
		return lifecycle.RequestStateChange(world, gc.WarpDungeonEnterWithPlannerEvent(debugName, mapplanner.PlannerTypeDebugAll.Name))
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
		WithChoice(TextClose, func(_ w.World) error {
			messageState.SetTransition(es.Transition[w.World]{Type: es.TransPop})
			return nil
		})

	messageState.messageData = messageData
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

// addLoadSlot はロードメニューにスロットを追加する。
// データが存在するスロットは選択可能、存在しないスロットは "---" で選択不可にする。
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
