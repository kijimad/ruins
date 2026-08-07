package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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

// Execute は構築済みのアクティビティを実行する。
// 即座実行アクション（移動、攻撃等）も継続アクション（休息等）も統一的に処理する。
// comp は NewXxxActivity 構築関数で作る。
func Execute(comp *gc.Activity, actor ecs.Entity, world w.World) (*ActionResult, error) {
	if comp == nil {
		return nil, ErrActivityNil
	}
	behaviorName := comp.BehaviorName
	log.Debug("action execution started",
		"type", behaviorName,
		"actor", actor)

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

	// 全アクションを継続アクションとして扱い、常にここで1ターン進める。
	// たまたま初回ステップで完了したものが即時アクションになる。特別な即時判定は持たない。
	// 呼び出し側が同じ呼び出しで結果を必要とするため、初回で解決すれば同期的に結果を返す。
	// アクター1体だけを対象にするため、入れ子処理（攻撃→被弾側の処理など）で他エンティティが
	// 消えても影響を受けない。全エンティティを回すと処理中コンポーネントの再利用で panic しうる。
	stepActivity(actor, world)

	currentActivity := query.GetActivity(world, actor)

	// 出口は「継続」か「初回で解決」かの2分岐。結果の中身だけが違い、
	// setLastResult と return は共通なので末尾へ1回だけ置く。
	var result *ActionResult
	if currentActivity != nil && !IsCompleted(currentActivity) && !IsCanceled(currentActivity) {
		// 継続アクション。この初回ターン分のコストを消費し、残りは ProcessContinuousActivities が進める
		query.ConsumeActionPoints(world, actor, consts.StandardActionCost)
		result = &ActionResult{Success: true, State: gc.ActivityStateRunning, ActivityName: behaviorName, Message: "action started"}
	} else {
		// 初回で解決した即時アクション。移動コストなど behavior 固有のコストを消費する
		consumePassCost(world, actor, comp)
		if currentActivity != nil && IsCanceled(currentActivity) {
			result = &ActionResult{Success: false, State: gc.ActivityStateCanceled, ActivityName: behaviorName, Message: query.T(world, currentActivity.CancelReason)}
		} else {
			result = &ActionResult{Success: true, State: gc.ActivityStateCompleted, ActivityName: behaviorName, Message: "action completed"}
		}
	}
	setLastResult(actor, result, world)
	return result, nil
}

// stepActivity は登録済みアクティビティを1ターン進める共通処理。
// 即時アクションの初回実行（Execute）と継続アクションの毎ターン処理
// （ProcessContinuousActivities）の両方から呼ばれ、両者で「1ターン進める」
// ロジックを一本化する。即時アクションは1ステップで完結する継続アクションの特殊ケースとして扱う。
//
// 実行する Behavior は永続化された BehaviorName から GetBehavior で引く。毎回ゼロ値の新しい
// インスタンスなので、着手時の呼び出し側インスタンスはここでは使わない。ライフサイクルを常に
// ゼロ値インスタンスで回すことで、per-アクティビティの状態は gc.Activity に置くという規律が経路によらず一貫する。
//
// DoTurn が失敗すればキャンセルし、完了していれば Finish して直近結果を記録し除去する。
// アクター1体のみを直接処理するため、DoTurn 内の入れ子処理で他エンティティが
// 消えても走査中コンポーネントの破壊による panic を招かない。
func stepActivity(entity ecs.Entity, world w.World) {
	stored := query.GetActivity(world, entity)
	if stored == nil {
		return
	}

	behaviorName := stored.BehaviorName
	behavior, err := GetBehavior(behaviorName)
	if err != nil {
		log.Error("failed to get behavior", "entity", entity, "error", err.Error())
		query.RemoveActivity(world, entity)
		return
	}

	if err := behavior.DoTurn(stored, entity, world); err != nil {
		log.Error("activity turn processing error", "entity", entity, "type", behaviorName, "error", err.Error())
		CancelActivity(entity, fmt.Sprintf("error: %s", err.Error()), world)
		return
	}

	if !IsCompleted(stored) {
		return
	}

	if err := behavior.Finish(stored, entity, world); err != nil {
		log.Error("activity finish processing error", "entity", entity, "type", behaviorName, "error", err.Error())
	}
	setLastResult(entity, &ActionResult{
		Success:      true,
		State:        gc.ActivityStateCompleted,
		ActivityName: behaviorName,
		Message:      "completed",
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
		log.Warn("failed to record last activity result", "actor", actor, "error", err.Error())
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
		if err := InterruptActivity(actor, "starting a new activity", world); err != nil {
			log.Warn("failed to interrupt existing activity", "entity", actor, "error", err.Error())
		}
	}

	// Behaviorでの検証
	if err := behavior.Validate(comp, actor, world); err != nil {
		return fmt.Errorf("activity validation failed: %w", err)
	}

	// アクティビティをコンポーネントとして登録する
	if err := query.SetActivity(world, actor, comp); err != nil {
		return fmt.Errorf("failed to register activity: %w", err)
	}
	stored := query.GetActivity(world, actor)
	stored.State = gc.ActivityStateRunning

	// BehaviorのStart処理を実行
	if err := behavior.Start(stored, actor, world); err != nil {
		// 開始に失敗した場合はクリーンアップ
		query.RemoveActivity(world, actor)
		return fmt.Errorf("failed to start activity: %w", err)
	}

	log.Debug("activity started",
		"entity", actor,
		"type", behavior.Name(),
		"required", stored.Progress.Max)

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
		return fmt.Errorf("activity '%s' cannot be resumed", GetDisplayName(comp))
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
		log.Warn("failed to get behavior", "entity", entity, "error", err.Error())
		query.RemoveActivity(world, entity)
		return
	}

	// BehaviorのCanceled処理を実行
	if err := behavior.Canceled(comp, entity, world); err != nil {
		log.Warn("activity cancel processing error",
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

	log.Debug("activity canceled",
		"entity", entity,
		"type", comp.BehaviorName,
		"reason", reason)
}

// ProcessContinuousActivities は継続中の全アクティビティを1ターン分進める。
// 即時アクション（Required==0）は Execute がその場で完結させるため、ここに残るのは継続実行アクションのみ。
// 走査中に他エンティティのアクティビティが削除されても、各要素で生存確認するため安全。
func ProcessContinuousActivities(world w.World) {
	var entities []ecs.Entity
	// 退避中ステージのエンティティのアクティビティは進めない
	activityQuery := query.ActiveFilter1[gc.Activity](world).Query()
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

		stepActivity(entity, world)
	}
}

// consumePassCost はアクションのAPコストを消費する。
// Behavior は comp.BehaviorName から GetBehavior で引く。
func consumePassCost(world w.World, actor ecs.Entity, comp *gc.Activity) {
	behavior, err := GetBehavior(comp.BehaviorName)
	if err != nil {
		log.Error("failed to get behavior", "actor", actor, "error", err.Error())
		return
	}
	cost := behavior.Info().ActionPointCost

	// 移動行動の場合、移動先タイルのPassCostを加算する。MoveParams を持つのは移動だけ
	if mp, ok := comp.Params.(*gc.MoveParams); ok {
		cost += getPassCostAt(world, int(mp.Destination.X), int(mp.Destination.Y))
	}

	if !query.ConsumeActionPoints(world, actor, cost) {
		log.Debug("no TurnBased component", "actor", actor)
	}
}

// getPassCostAt は指定座標にある固定物のPassCostを合算して返す
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
		return 0, fmt.Errorf("TurnBased component not found: entity=%v", entity)
	}
	return world.Components.TurnBased.Get(entity).AP.Max, nil
}
