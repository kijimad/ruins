//go:build !js || !wasm

package states

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/logger"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/save"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// save を参照する UI とファクトリをこのデスクトップ限定ファイルへ集約する。
// WASM ビルドは saveload_wasm.go の空実装を使い、states から save への参照が消えてバイナリの依存グラフから save が外れる。
// 保存できるデスクトップでも、体験版モードのときは採否関数がセーブ・ロードを落とす。

// loadMainMenuItem はメインメニューのロード項目を返す。ok が真なら項目を採用する。体験版では出さない
func loadMainMenuItem(world w.World) (mainMenuItem, bool) {
	if world.Resources.Config.Demo {
		return mainMenuItem{}, false
	}
	return mainMenuItem{
		Label:      query.T(world, "Load"),
		Transition: es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewLoadMenuState}},
	}, true
}

// dungeonSaveMenuChoice はダンジョンメニューのセーブ項目を返す。ok が真なら項目を採用する。体験版では出さない
func dungeonSaveMenuChoice(world w.World) (Choice, bool) {
	if world.Resources.Config.Demo {
		return Choice{}, false
	}
	return Choice{Label: query.T(world, "Save game"), Run: pushChoice(NewSaveMenuState)}, true
}

// NewSaveMenuState は手動セーブ画面を作る。固定4スロットで、主人公名とタイムスタンプを表示する
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

// NewLoadMenuState はロード画面を作る。手動4スロットとオートセーブ4スロットをセクション分けで表示する
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
