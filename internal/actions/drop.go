package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// DropActivity はActivityInterfaceの実装
type DropActivity struct {
	// Target は捨てるアイテムのエンティティ
	Target ecs.Entity
}

// Info はActivityInterfaceの実装
func (da *DropActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "ドロップ",
		Description:     "アイテムを足元に置く",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 50,
		TotalRequiredAP: 50,
	}
}

// String はActivityInterfaceの実装
func (da *DropActivity) String() string {
	return "Drop"
}

// Validate はアイテムドロップアクティビティの検証を行う
func (da *DropActivity) Validate(act *Activity, world w.World) error {
	// Targetがバックパック内にあることを確認
	if !da.Target.HasComponent(world.Components.ItemLocationInPlayerBackpack) {
		return fmt.Errorf("アイテムがバックパック内にありません")
	}

	// プレイヤーの位置情報が必要
	gridElement := world.Components.GridElement.Get(act.Actor)
	if gridElement == nil {
		return fmt.Errorf("位置情報が見つかりません")
	}

	return nil
}

// Start はアイテムドロップ開始時の処理を実行する
func (da *DropActivity) Start(act *Activity, _ w.World) error {
	act.Logger.Debug("アイテムドロップ開始", "actor", act.Actor, "target", da.Target)
	return nil
}

// DoTurn はアイテムドロップアクティビティの1ターン分の処理を実行する
func (da *DropActivity) DoTurn(act *Activity, world w.World) error {
	// アイテムドロップ処理を実行
	if err := da.performDropActivity(act, world); err != nil {
		act.Cancel(fmt.Sprintf("アイテムドロップエラー: %s", err.Error()))
		return err
	}

	// ドロップ処理完了
	act.Complete()
	return nil
}

// Finish はアイテムドロップ完了時の処理を実行する
func (da *DropActivity) Finish(act *Activity, _ w.World) error {
	act.Logger.Debug("アイテムドロップアクティビティ完了", "actor", act.Actor)
	return nil
}

// Canceled はアイテムドロップキャンセル時の処理を実行する
func (da *DropActivity) Canceled(act *Activity, _ w.World) error {
	act.Logger.Debug("アイテムドロップキャンセル", "actor", act.Actor, "reason", act.CancelReason)
	return nil
}

// performDropActivity は実際のアイテムドロップ処理を実行する
func (da *DropActivity) performDropActivity(act *Activity, world w.World) error {
	// プレイヤー位置を取得
	gridElement := world.Components.GridElement.Get(act.Actor)
	if gridElement == nil {
		return fmt.Errorf("位置情報が見つかりません")
	}

	playerGrid := gridElement.(*gc.GridElement)

	// アイテム情報を取得
	formattedName := worldhelper.FormatItemName(world, da.Target)

	// バックパックから削除してフィールドに移動
	worldhelper.MoveToField(world, da.Target, act.Actor)

	// グリッド位置を設定
	da.Target.AddComponent(world.Components.GridElement, &gc.GridElement{
		X: playerGrid.X,
		Y: playerGrid.Y,
	})

	gamelog.New(gamelog.FieldLog).
		ItemName(formattedName).
		Append(" を捨てた。").
		Log()

	return nil
}
