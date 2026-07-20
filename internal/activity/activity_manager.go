package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ActionResult はアクション実行結果を表す
type ActionResult struct {
	Success      bool             // 実行成功/失敗
	State        gc.ActivityState // アクティビティの終了状態
	ActivityName gc.BehaviorName  // 実行されたアクティビティ名
	Message      string           // 結果メッセージ
}

// Execute は指定されたアクティビティを実行する
// 即座実行アクション（移動、攻撃等）も継続アクション（休息等）も統一的に処理する
func Execute(behavior Behavior, actor ecs.Entity, world w.World) (*ActionResult, error) {
	behaviorName := behavior.Name()
	log.Debug("アクション実行開始",
		"type", behaviorName,
		"actor", actor)

	// アクティビティを作成
	comp, err := behavior.BuildActivity(actor, world)
	if err != nil {
		result := &ActionResult{
			Success:      false,
			State:        gc.ActivityStateCanceled,
			ActivityName: behaviorName,
			Message:      err.Error(),
		}
		setLastResult(actor, result, world)
		return result, err
	}

	// アクティビティを開始
	if err := StartActivity(comp, actor, world); err != nil {
		result := &ActionResult{
			Success:      false,
			State:        gc.ActivityStateCanceled,
			ActivityName: behaviorName,
			Message:      err.Error(),
		}
		setLastResult(actor, result, world)
		return result, err
	}

	// 即座実行アクション（1ターン）は、登録済みアクティビティを1ターン進めてその場で完結させる。
	// アクター1体だけを対象にするため、入れ子処理（攻撃→被弾側の処理など）で他エンティティが
	// 消えても影響を受けない。全エンティティを回すと処理中コンポーネントの再利用で panic しうる。
	if comp.TurnsTotal == 1 {
		stepActivity(behavior, actor, world)

		// ターン管理システムに移動コストを通知
		consumePassCost(world, behavior, actor, comp.Destination)

		// 結果を確認
		currentActivity := query.GetActivity(world, actor)
		if currentActivity == nil || IsCompleted(currentActivity) {
			result := &ActionResult{
				Success:      true,
				State:        gc.ActivityStateCompleted,
				ActivityName: behaviorName,
				Message:      "アクション完了",
			}
			setLastResult(actor, result, world)
			return result, nil
		} else if IsCanceled(currentActivity) {
			result := &ActionResult{
				Success:      false,
				State:        gc.ActivityStateCanceled,
				ActivityName: behaviorName,
				Message:      currentActivity.CancelReason,
			}
			setLastResult(actor, result, world)
			return result, nil
		}
	}

	// 継続アクションの場合は開始成功を返す
	result := &ActionResult{
		Success:      true,
		State:        gc.ActivityStateRunning,
		ActivityName: behaviorName,
		Message:      "アクション開始",
	}
	setLastResult(actor, result, world)
	return result, nil
}

// stepActivity は登録済みアクティビティを1ターン進める共通処理。
// 即時アクションの初回実行（Execute）と継続アクションの毎ターン処理
// （ProcessContinuousActivities）の両方から呼ばれ、両者で「1ターン進める」
// ロジックを一本化する。即時アクションは1ステップで完結する継続アクションの特殊ケースとして扱う。
//
// DoTurn が失敗すればキャンセルし、完了していれば Finish して直近結果を記録し除去する。
// アクター1体のみを直接処理するため、DoTurn 内の入れ子処理で他エンティティが
// 消えても走査中コンポーネントの破壊による panic を招かない。
func stepActivity(behavior Behavior, entity ecs.Entity, world w.World) {
	stored := query.GetActivity(world, entity)
	if stored == nil {
		return
	}

	behaviorName := behavior.Name()
	if err := behavior.DoTurn(stored, entity, world); err != nil {
		log.Error("アクティビティターン処理エラー", "entity", entity, "type", behaviorName, "error", err.Error())
		CancelActivity(entity, fmt.Sprintf("エラー: %s", err.Error()), world)
		return
	}

	if !IsCompleted(stored) {
		return
	}

	if err := behavior.Finish(stored, entity, world); err != nil {
		log.Error("アクティビティ完了処理エラー", "entity", entity, "type", behaviorName, "error", err.Error())
	}
	setLastResult(entity, &ActionResult{
		Success:      true,
		State:        gc.ActivityStateCompleted,
		ActivityName: behaviorName,
		Message:      "完了",
	}, world)
	query.RemoveActivity(world, entity)
}

// setLastResult はエンティティの直近アクティビティ結果を設定する
func setLastResult(actor ecs.Entity, result *ActionResult, world w.World) {
	lastResult := &gc.LastActivity{
		BehaviorName: result.ActivityName,
		State:        result.State,
		Success:      result.Success,
		Message:      result.Message,
	}

	if err := gc.Upsert(world.ECS, world.Components.LastActivity, actor, lastResult); err != nil {
		log.Warn("直近アクティビティ結果の記録に失敗", "actor", actor, "error", err.Error())
	}
}

// GetLastResult はエンティティの直近アクティビティ結果を取得する
func GetLastResult(actor ecs.Entity, world w.World) *gc.LastActivity {
	if !world.Components.LastActivity.Has(actor) {
		return nil
	}
	return world.Components.LastActivity.Get(actor)
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
	if currentActivity := query.GetActivity(world, actor); currentActivity != nil {
		if err := InterruptActivity(actor, "新しいアクティビティを開始", world); err != nil {
			log.Warn("既存アクティビティの中断に失敗", "entity", actor, "error", err.Error())
		}
	}

	// Behaviorでの検証
	if err := behavior.Validate(comp, actor, world); err != nil {
		return fmt.Errorf("アクティビティ検証失敗: %w", err)
	}

	// アクティビティをコンポーネントとして登録する
	if err := query.SetActivity(world, actor, comp); err != nil {
		return fmt.Errorf("アクティビティ登録失敗: %w", err)
	}
	stored := query.GetActivity(world, actor)
	stored.State = gc.ActivityStateRunning

	// BehaviorのStart処理を実行
	if err := behavior.Start(stored, actor, world); err != nil {
		// 開始に失敗した場合はクリーンアップ
		query.RemoveActivity(world, actor)
		return fmt.Errorf("アクティビティ開始失敗: %w", err)
	}

	log.Debug("アクティビティ開始",
		"entity", actor,
		"type", behavior.Name(),
		"duration", stored.TurnsTotal)

	return nil
}

// InterruptActivity は指定されたエンティティのアクティビティを中断する
func InterruptActivity(entity ecs.Entity, reason string, world w.World) error {
	comp := query.GetActivity(world, entity)
	if comp == nil {
		return ErrActivityNotFound
	}

	return Interrupt(comp, reason)
}

// ResumeActivity は指定されたエンティティのアクティビティを再開する
func ResumeActivity(entity ecs.Entity, world w.World) error {
	comp := query.GetActivity(world, entity)
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
	comp := query.GetActivity(world, entity)
	if comp == nil {
		return
	}

	behavior, err := GetBehavior(comp.BehaviorName)
	if err != nil {
		log.Warn("Behaviorの取得に失敗", "entity", entity, "error", err.Error())
		query.RemoveActivity(world, entity)
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

	query.RemoveActivity(world, entity)

	log.Debug("アクティビティキャンセル",
		"entity", entity,
		"type", comp.BehaviorName,
		"reason", reason)
}

// ProcessContinuousActivities は継続中の全アクティビティを1ターン分進める。
// 即時アクション（TurnsTotal==1）は Execute がその場で完結させるため、ここに残るのは継続実行アクションのみ。
// 走査中に他エンティティのアクティビティが削除されても、各要素で生存確認するため安全。
func ProcessContinuousActivities(world w.World) {
	var entities []ecs.Entity
	activityQuery := ecs.NewFilter1[gc.Activity](world.ECS).Query()
	for activityQuery.Next() {
		entities = append(entities, activityQuery.Entity())
	}

	for _, entity := range entities {
		if !world.ECS.Alive(entity) || !world.Components.Activity.Has(entity) {
			continue
		}
		comp := world.Components.Activity.Get(entity)

		if !IsActive(comp) {
			if IsCompleted(comp) || IsCanceled(comp) {
				query.RemoveActivity(world, entity)
			}
			continue
		}

		behavior, err := GetBehavior(comp.BehaviorName)
		if err != nil {
			log.Error("Behaviorの取得に失敗", "entity", entity, "error", err.Error())
			query.RemoveActivity(world, entity)
			continue
		}

		stepActivity(behavior, entity, world)
	}
}

// consumePassCost はアクションのAPコストを消費する
func consumePassCost(world w.World, behavior Behavior, actor ecs.Entity, destination *gc.GridElement) {
	info := behavior.Info()
	cost := info.ActionPointCost

	// 移動行動の場合、移動先タイルのPassCostを加算する
	if behavior.Name() == gc.BehaviorMove && destination != nil {
		cost += getPassCostAt(world, int(destination.X), int(destination.Y))
	}

	if !query.ConsumeActionPoints(world, actor, cost) {
		log.Debug("TurnBasedコンポーネントがない", "actor", actor)
	}
}

// getPassCostAt は指定座標にあるPropのPassCostを合算して返す
func getPassCostAt(world w.World, x, y int) int {
	total := 0
	passCostQuery := query.ActiveFilter2[gc.GridElement, gc.PassCost](world).Query()
	for passCostQuery.Next() {
		entity := passCostQuery.Entity()
		grid := world.Components.GridElement.Get(entity)
		if int(grid.X) == x && int(grid.Y) == y {
			mc := world.Components.PassCost.Get(entity)
			total += mc.Value
		}
	}
	return total
}

// getEntityMaxAP はエンティティの最大AP値を取得する
func getEntityMaxAP(entity ecs.Entity, world w.World) (int, error) {
	if !world.Components.TurnBased.Has(entity) {
		return 0, fmt.Errorf("TurnBasedコンポーネントが見つからない: entity=%v", entity)
	}
	return world.Components.TurnBased.Get(entity).AP.Max, nil
}
