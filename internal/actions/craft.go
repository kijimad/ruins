package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

// CraftActivity はActivityInterfaceの実装
type CraftActivity struct{}

// Info はActivityInterfaceの実装
func (ca *CraftActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "クラフト",
		Description:     "アイテムを作成する",
		Interruptible:   true,
		Resumable:       false,
		ActionPointCost: 100,
		TotalRequiredAP: 1500,
	}
}

// String はActivityInterfaceの実装
func (ca *CraftActivity) String() string {
	return "Craft"
}

// Validate はクラフトアクティビティの検証を行う
// Validate はActivityInterfaceの実装
func (ca *CraftActivity) Validate(act *Activity, world w.World) error {
	// クラフト対象（レシピ）が必要
	if act.Target == nil {
		return fmt.Errorf("クラフトにはレシピが必要です")
	}

	// レシピが実際に存在するかチェック
	targetEntity := *act.Target

	// Recipeコンポーネントを持つかチェック
	if !targetEntity.HasComponent(world.Components.Recipe) {
		return fmt.Errorf("クラフト対象がレシピではありません")
	}

	// レシピ名を取得
	recipeName := ""
	if nameComp := world.Components.Name.Get(targetEntity); nameComp != nil {
		name := nameComp.(*gc.Name)
		recipeName = name.Name
	} else {
		return fmt.Errorf("レシピ名が取得できません")
	}

	// クラフト可能性をチェック
	canCraft, err := worldhelper.CanCraft(world, recipeName)
	if err != nil {
		return fmt.Errorf("レシピエラー: %w", err)
	}
	if !canCraft {
		return fmt.Errorf("必要な材料やツールが不足しています")
	}

	// TODO: より詳細なクラフト可能チェック
	// - 必要なツールがあるか
	// - 作業場所が適切か
	// - クラフトスキルが十分か

	return nil
}

// Start はクラフト開始時の処理を実行する
// Start はActivityInterfaceの実装
func (ca *CraftActivity) Start(act *Activity, world w.World) error {
	act.Logger.Debug("クラフト開始", "actor", act.Actor, "target", *act.Target, "duration", act.TurnsLeft)

	// クラフト開始メッセージ
	targetEntity := *act.Target
	recipeName := "アイテム"
	if nameComp := world.Components.Name.Get(targetEntity); nameComp != nil {
		name := nameComp.(*gc.Name)
		recipeName = name.Name
	}

	// プレイヤーの場合のみクラフト開始メッセージを表示
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append("クラフトを開始した: ").
			ItemName(recipeName).
			Log()
	}

	return nil
}

// DoTurn はクラフトアクティビティの1ターン分の処理を実行する
// DoTurn はActivityInterfaceの実装
func (ca *CraftActivity) DoTurn(act *Activity, world w.World) error {
	// クラフト条件を再チェック
	if err := ca.Validate(act, world); err != nil {
		act.Cancel(fmt.Sprintf("クラフト条件が満たされません: %s", err.Error()))
		return err
	}

	// 基本のターン処理
	if act.TurnsLeft <= 0 {
		act.Complete()
		return nil
	}

	// 1ターン進行
	act.TurnsLeft--
	act.Logger.Debug("クラフト進行",
		"turns_left", act.TurnsLeft,
		"progress", act.GetProgressPercent())

	// クラフト処理
	if err := ca.performCrafting(act, world); err != nil {
		act.Logger.Warn("クラフト処理エラー", "error", err.Error())
	}

	// 完了チェック
	if act.TurnsLeft <= 0 {
		act.Complete()
		return nil
	}

	return nil
}

// Finish はクラフト完了時の処理を実行する
// Finish はActivityInterfaceの実装
func (ca *CraftActivity) Finish(act *Activity, world w.World) error {
	act.Logger.Debug("クラフト完了", "actor", act.Actor)

	// 実際のクラフト実行
	targetEntity := *act.Target
	recipeName := ""
	if nameComp := world.Components.Name.Get(targetEntity); nameComp != nil {
		name := nameComp.(*gc.Name)
		recipeName = name.Name
	} else {
		return fmt.Errorf("レシピ名が取得できません")
	}

	_, err := worldhelper.Craft(world, recipeName)
	if err != nil {
		act.Logger.Error("クラフト実行エラー", "error", err.Error())
		return fmt.Errorf("クラフトに失敗しました: %w", err)
	}

	// 完了メッセージ

	// プレイヤーの場合のみクラフト完了メッセージを表示
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append("クラフトを完了した: ").
			ItemName(recipeName).
			Log()
	}

	return nil
}

// Canceled はクラフトキャンセル時の処理を実行する
// Canceled はActivityInterfaceの実装
func (ca *CraftActivity) Canceled(act *Activity, world w.World) error {
	// プレイヤーの場合のみ中断時のメッセージを表示
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append("クラフトが中断された: ").
			Append(act.CancelReason).
			Log()
	}

	act.Logger.Debug("クラフト中断", "reason", act.CancelReason, "progress", act.GetProgressPercent())
	return nil
}

// performCrafting はクラフト処理を実行する
func (ca *CraftActivity) performCrafting(act *Activity, world w.World) error {
	// アクターの存在チェック
	if act.Actor == 0 {
		return fmt.Errorf("クラフトするエンティティが指定されていません")
	}

	// TODO: クラフト進行による効果実装
	// - スキル経験値の獲得
	// - 中間処理の実行

	// プレイヤーの場合のみ進行ログを出力（25%毎）
	if isPlayerActivity(act, world) {
		progress := act.GetProgressPercent()
		if progress >= 25.0 && progress < 26.0 {
			gamelog.New(gamelog.FieldLog).
				Append("材料を準備している...").
				Log()
		} else if progress >= 50.0 && progress < 51.0 {
			gamelog.New(gamelog.FieldLog).
				Append("作業を進めている...").
				Log()
		} else if progress >= 75.0 && progress < 76.0 {
			gamelog.New(gamelog.FieldLog).
				Append("仕上げに入っている...").
				Log()
		}
	}

	return nil
}
