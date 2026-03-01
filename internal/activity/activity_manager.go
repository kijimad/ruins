package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ActionParams はアクション実行時のパラメータを表す
type ActionParams struct {
	Actor       ecs.Entity      // アクションを実行するエンティティ
	Target      *ecs.Entity     // 対象エンティティ（攻撃等で使用）
	Destination *gc.GridElement // 移動先のグリッド座標（移動等で使用）
	Duration    int             // 継続時間（休息、待機等で使用）
	Reason      string          // 理由（待機等で使用）
}

// ActionResult はアクション実行結果を表す
type ActionResult struct {
	Success      bool             // 実行成功/失敗
	State        gc.ActivityState // アクティビティの終了状態
	ActivityName gc.BehaviorName  // 実行されたアクティビティ名
	Message      string           // 結果メッセージ
}

// Execute は指定されたアクティビティを実行する
// 即座実行アクション（移動、攻撃等）も継続アクション（休息等）も統一的に処理する
func Execute(behavior Behavior, params ActionParams, world w.World) (*ActionResult, error) {
	behaviorName := behavior.Name()
	log.Debug("アクション実行開始",
		"type", behaviorName,
		"actor", params.Actor)

	// アクティビティを作成
	comp, err := buildActivity(behavior, params, world)
	if err != nil {
		result := &ActionResult{
			Success:      false,
			State:        gc.ActivityStateCanceled,
			ActivityName: behaviorName,
			Message:      err.Error(),
		}
		setLastResult(params.Actor, result, world)
		return result, err
	}

	// アクティビティを開始
	if err := StartActivity(comp, params.Actor, world); err != nil {
		result := &ActionResult{
			Success:      false,
			State:        gc.ActivityStateCanceled,
			ActivityName: behaviorName,
			Message:      err.Error(),
		}
		setLastResult(params.Actor, result, world)
		return result, err
	}

	// 即座実行アクション（1ターン）の場合は即座に処理
	if comp.TurnsTotal == 1 {
		// ターン処理実行
		ProcessTurn(world)

		// ターン管理システムに移動コストを通知
		consumeMoveCost(world, behavior, params.Actor)

		// 結果を確認
		currentActivity := worldhelper.GetActivity(world, params.Actor)
		if currentActivity == nil || IsCompleted(currentActivity) {
			result := &ActionResult{
				Success:      true,
				State:        gc.ActivityStateCompleted,
				ActivityName: behaviorName,
				Message:      "アクション完了",
			}
			setLastResult(params.Actor, result, world)
			return result, nil
		} else if IsCanceled(currentActivity) {
			result := &ActionResult{
				Success:      false,
				State:        gc.ActivityStateCanceled,
				ActivityName: behaviorName,
				Message:      currentActivity.CancelReason,
			}
			setLastResult(params.Actor, result, world)
			return result, fmt.Errorf("アクション失敗: %s", currentActivity.CancelReason)
		}
	}

	// 継続アクションの場合は開始成功を返す
	result := &ActionResult{
		Success:      true,
		State:        gc.ActivityStateRunning,
		ActivityName: behaviorName,
		Message:      "アクション開始",
	}
	setLastResult(params.Actor, result, world)
	return result, nil
}

// setLastResult はエンティティの直近アクティビティ結果を設定する
func setLastResult(actor ecs.Entity, result *ActionResult, world w.World) {
	lastResult := &gc.LastActivityResult{
		BehaviorName: result.ActivityName,
		State:        result.State,
		Success:      result.Success,
		Message:      result.Message,
	}

	if actor.HasComponent(world.Components.LastActivityResult) {
		actor.RemoveComponent(world.Components.LastActivityResult)
	}
	actor.AddComponent(world.Components.LastActivityResult, lastResult)
}

// GetLastResult はエンティティの直近アクティビティ結果を取得する
func GetLastResult(actor ecs.Entity, world w.World) *gc.LastActivityResult {
	comp := world.Components.LastActivityResult.Get(actor)
	if comp == nil {
		return nil
	}
	return comp.(*gc.LastActivityResult)
}

// StartActivity は新しいアクティビティを開始する
func StartActivity(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp == nil {
		return ErrActivityNil
	}

	behavior, err := GetBehavior(comp.BehaviorName)
	if err != nil {
		return err
	}

	// 既存のアクティビティがある場合は中断
	if currentActivity := worldhelper.GetActivity(world, actor); currentActivity != nil {
		if err := InterruptActivity(actor, "新しいアクティビティを開始", world); err != nil {
			log.Warn("既存アクティビティの中断に失敗", "entity", actor, "error", err.Error())
		}
	}

	// Behaviorでの検証
	if err := behavior.Validate(comp, actor, world); err != nil {
		return fmt.Errorf("アクティビティ検証失敗: %w", err)
	}

	// アクティビティをコンポーネントとして登録
	worldhelper.SetActivity(world, actor, comp)
	comp.State = gc.ActivityStateRunning

	// BehaviorのStart処理を実行
	if err := behavior.Start(comp, actor, world); err != nil {
		// 開始に失敗した場合はクリーンアップ
		worldhelper.RemoveActivity(world, actor)
		return fmt.Errorf("アクティビティ開始失敗: %w", err)
	}

	log.Debug("アクティビティ開始",
		"entity", actor,
		"type", behavior.Name(),
		"duration", comp.TurnsTotal)

	return nil
}

// InterruptActivity は指定されたエンティティのアクティビティを中断する
func InterruptActivity(entity ecs.Entity, reason string, world w.World) error {
	comp := worldhelper.GetActivity(world, entity)
	if comp == nil {
		return ErrActivityNotFound
	}

	return Interrupt(comp, reason)
}

// ResumeActivity は指定されたエンティティのアクティビティを再開する
func ResumeActivity(entity ecs.Entity, world w.World) error {
	comp := worldhelper.GetActivity(world, entity)
	if comp == nil {
		return ErrActivityNotFound
	}

	if !CanResume(comp) {
		return fmt.Errorf("アクティビティ '%s' は再開できません", GetDisplayName(comp))
	}

	return Resume(comp)
}

// CancelActivity は指定されたエンティティのアクティビティをキャンセルする
func CancelActivity(entity ecs.Entity, reason string, world w.World) {
	comp := worldhelper.GetActivity(world, entity)
	if comp == nil {
		return
	}

	behavior, err := GetBehavior(comp.BehaviorName)
	if err != nil {
		log.Warn("Behaviorの取得に失敗", "entity", entity, "error", err.Error())
		worldhelper.RemoveActivity(world, entity)
		return
	}

	// BehaviorのCanceled処理を実行
	if err := behavior.Canceled(comp, entity, world); err != nil {
		log.Warn("アクティビティキャンセル処理エラー",
			"entity", entity,
			"error", err.Error())
	}

	// アクティビティ自体をキャンセル状態に
	Cancel(comp, reason)

	// 結果を記録
	result := &ActionResult{
		Success:      false,
		State:        gc.ActivityStateCanceled,
		ActivityName: comp.BehaviorName,
		Message:      reason,
	}
	setLastResult(entity, result, world)

	worldhelper.RemoveActivity(world, entity)

	log.Debug("アクティビティキャンセル",
		"entity", entity,
		"type", comp.BehaviorName,
		"reason", reason)
}

// ProcessTurn は全てのアクティブなアクティビティの1ターン分の処理を実行する
func ProcessTurn(world w.World) {
	log.Debug("アクティビティターン処理開始")

	// 完了・キャンセルされたアクティビティを削除するためのリスト
	var toRemove []ecs.Entity

	world.Manager.Join(world.Components.Activity).Visit(ecs.Visit(func(entity ecs.Entity) {
		comp := world.Components.Activity.Get(entity).(*gc.Activity)

		// アクティブなアクティビティのみ処理
		if !IsActive(comp) {
			if IsCompleted(comp) || IsCanceled(comp) {
				toRemove = append(toRemove, entity)
			}
			return
		}

		behavior, err := GetBehavior(comp.BehaviorName)
		if err != nil {
			log.Error("Behaviorの取得に失敗", "entity", entity, "error", err.Error())
			toRemove = append(toRemove, entity)
			return
		}

		// ターン処理を実行
		if err := behavior.DoTurn(comp, entity, world); err != nil {
			log.Error("アクティビティターン処理エラー",
				"entity", entity,
				"type", comp.BehaviorName,
				"error", err.Error())

			// エラーが発生した場合はキャンセル
			CancelActivity(entity, fmt.Sprintf("エラー: %s", err.Error()), world)
			toRemove = append(toRemove, entity)
			return
		}

		// 完了したアクティビティの処理
		if IsCompleted(comp) {
			// Finish処理を実行
			if err := behavior.Finish(comp, entity, world); err != nil {
				log.Error("アクティビティ完了処理エラー",
					"entity", entity,
					"type", comp.BehaviorName,
					"error", err.Error())
			}

			// 結果を記録
			result := &ActionResult{
				Success:      true,
				State:        gc.ActivityStateCompleted,
				ActivityName: comp.BehaviorName,
				Message:      "完了",
			}
			setLastResult(entity, result, world)

			log.Debug("アクティビティ完了",
				"entity", entity,
				"type", comp.BehaviorName)
			toRemove = append(toRemove, entity)
		}
	}))

	// 完了・キャンセルされたアクティビティを削除
	for _, entity := range toRemove {
		worldhelper.RemoveActivity(world, entity)
	}

	log.Debug("アクティビティターン処理完了", "removed", len(toRemove))
}

// buildActivity はアクティビティ実装とパラメータからアクティビティを作成する
func buildActivity(behavior Behavior, params ActionParams, world w.World) (*gc.Activity, error) {
	// 基本のdurationを計算
	duration := params.Duration
	if duration <= 0 {
		characterAP := getEntityMaxAP(params.Actor, world)
		duration = CalculateRequiredTurns(behavior, characterAP)
	}

	// アクティビティを作成
	comp, err := NewActivity(behavior, duration)
	if err != nil {
		return nil, err
	}

	// パラメータを設定
	if params.Destination != nil {
		comp.Destination = params.Destination
	}
	if params.Target != nil {
		comp.Target = params.Target
	}

	return comp, nil
}

// consumeMoveCost はアクションのAPコストを消費する
func consumeMoveCost(world w.World, behavior Behavior, actor ecs.Entity) {
	info := behavior.Info()
	cost := info.ActionPointCost

	// TurnBasedコンポーネントから直接APを消費
	tbComp := world.Components.TurnBased.Get(actor)
	if tbComp == nil {
		log.Debug("TurnBasedコンポーネントがない", "actor", actor)
		return
	}

	tb := tbComp.(*gc.TurnBased)
	tb.AP.Current -= cost

	log.Debug("移動コスト消費",
		"activity", behavior.Name(),
		"cost", cost,
		"remaining", tb.AP.Current,
		"actor", actor,
		"isPlayer", actor.HasComponent(world.Components.Player))
}

// getEntityMaxAP はエンティティの最大AP値を取得する
func getEntityMaxAP(entity ecs.Entity, world w.World) int {
	if turnBasedComponent := world.Components.TurnBased.Get(entity); turnBasedComponent != nil {
		turnBased := turnBasedComponent.(*gc.TurnBased)
		return turnBased.AP.Max
	}
	log.Debug("TurnBasedコンポーネントが見つからない", "entity", entity)
	return 100 // デフォルトAP値
}
