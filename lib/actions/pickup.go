package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/lib/components"
	"github.com/kijimaD/ruins/lib/gamelog"
	w "github.com/kijimaD/ruins/lib/world"
	"github.com/kijimaD/ruins/lib/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// PickupActivity はActivityInterfaceの実装
type PickupActivity struct{}

// Info はActivityInterfaceの実装
func (pa *PickupActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "拾得",
		Description:     "アイテムを拾得する",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 50,
		TotalRequiredAP: 50,
	}
}

// String はActivityInterfaceの実装
func (pa *PickupActivity) String() string {
	return "Pickup"
}

// Validate はアイテム拾得アクティビティの検証を行う
// Validate はActivityInterfaceの実装
func (pa *PickupActivity) Validate(act *Activity, world w.World) error {
	// プレイヤーの位置情報が必要
	gridElement := world.Components.GridElement.Get(act.Actor)
	if gridElement == nil {
		return fmt.Errorf("位置情報が見つかりません")
	}

	playerGrid := gridElement.(*gc.GridElement)
	playerTileX := int(playerGrid.X)
	playerTileY := int(playerGrid.Y)

	// 同じタイルにフィールドアイテムがあるかチェック
	hasItems := false
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationOnField,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(itemEntity ecs.Entity) {
		itemGrid := world.Components.GridElement.Get(itemEntity).(*gc.GridElement)
		if int(itemGrid.X) == playerTileX && int(itemGrid.Y) == playerTileY {
			hasItems = true
		}
	}))

	if !hasItems {
		return fmt.Errorf("拾えるアイテムがありません")
	}

	return nil
}

// Start はアイテム拾得開始時の処理を実行する
// Start はActivityInterfaceの実装
func (pa *PickupActivity) Start(act *Activity, _ w.World) error {
	act.Logger.Debug("アイテム拾得開始", "actor", act.Actor)
	return nil
}

// DoTurn はアイテム拾得アクティビティの1ターン分の処理を実行する
// DoTurn はActivityInterfaceの実装
func (pa *PickupActivity) DoTurn(act *Activity, world w.World) error {
	// アイテム拾得処理を実行
	if err := pa.performPickupActivity(act, world); err != nil {
		act.Cancel(fmt.Sprintf("アイテム拾得エラー: %s", err.Error()))
		return err
	}

	// 拾得処理完了
	act.Complete()
	return nil
}

// Finish はアイテム拾得完了時の処理を実行する
// Finish はActivityInterfaceの実装
func (pa *PickupActivity) Finish(act *Activity, _ w.World) error {
	act.Logger.Debug("アイテム拾得アクティビティ完了", "actor", act.Actor)
	return nil
}

// Canceled はアイテム拾得キャンセル時の処理を実行する
// Canceled はActivityInterfaceの実装
func (pa *PickupActivity) Canceled(act *Activity, _ w.World) error {
	act.Logger.Debug("アイテム拾得キャンセル", "actor", act.Actor, "reason", act.CancelReason)
	return nil
}

// performPickupActivity は実際のアイテム拾得処理を実行する
func (pa *PickupActivity) performPickupActivity(act *Activity, world w.World) error {
	// プレイヤー位置を取得
	gridElement := world.Components.GridElement.Get(act.Actor)
	if gridElement == nil {
		return fmt.Errorf("位置情報が見つかりません")
	}

	playerGrid := gridElement.(*gc.GridElement)
	playerTileX := int(playerGrid.X)
	playerTileY := int(playerGrid.Y)

	// 同じタイルのフィールドアイテムを検索
	var itemsToCollect []ecs.Entity
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationOnField,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(itemEntity ecs.Entity) {
		itemGrid := world.Components.GridElement.Get(itemEntity).(*gc.GridElement)
		// タイル単位の位置判定
		if int(itemGrid.X) == playerTileX && int(itemGrid.Y) == playerTileY {
			itemsToCollect = append(itemsToCollect, itemEntity)
		}
	}))

	if len(itemsToCollect) == 0 {
		return fmt.Errorf("拾えるアイテムがありません")
	}

	// 収集されたアイテムを処理
	collectedCount := 0
	for _, itemEntity := range itemsToCollect {
		if err := pa.collectFieldItem(act, world, itemEntity); err != nil {
			act.Logger.Warn("アイテム拾得エラー", "item", itemEntity, "error", err.Error())
			continue
		}
		collectedCount++
	}

	if collectedCount == 0 {
		return fmt.Errorf("アイテムの拾得に失敗しました")
	}

	act.Logger.Debug("アイテム拾得完了", "count", collectedCount)

	// プレイヤーの場合のみ複数アイテム収集時の総括メッセージを表示
	if collectedCount > 1 && isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append(fmt.Sprintf("%d個のアイテムを入手した", collectedCount)).
			Log()
	}

	return nil
}

// collectFieldItem はフィールドアイテムを収集してバックパックに移動する
func (pa *PickupActivity) collectFieldItem(_ *Activity, world w.World, itemEntity ecs.Entity) error {
	itemName := "Unknown Item"
	if nameComp := world.Components.Name.Get(itemEntity); nameComp != nil {
		name := nameComp.(*gc.Name)
		itemName = name.Name
	}

	// フィールドからバックパックに移動
	// ItemLocationOnFieldコンポーネントを削除
	itemEntity.RemoveComponent(world.Components.ItemLocationOnField)
	// ItemLocationInBackpackコンポーネントを追加
	itemEntity.AddComponent(world.Components.ItemLocationInBackpack, gc.LocationInBackpack{})

	// グリッド表示コンポーネントを削除（フィールドから消す）
	if itemEntity.HasComponent(world.Components.GridElement) {
		itemEntity.RemoveComponent(world.Components.GridElement)
	}

	// 既存のバックパック内の同じアイテムと統合する処理
	if err := worldhelper.MergeStackableIntoInventory(world, itemEntity, itemName); err != nil {
		return fmt.Errorf("インベントリ統合エラー: %w", err)
	}

	gamelog.New(gamelog.FieldLog).
		ItemName(itemName).
		Append(" を入手した。").
		Log()

	return nil
}
