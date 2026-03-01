package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// Manager はアクティビティの管理を行う
type Manager struct {
	// 現在実行中の全アクティビティ(全エンティティごと)
	// 1エンティティで最大1アクティビティ
	currentActivities map[ecs.Entity]*Activity
	logger            *logger.Logger
	// History はテスト用の履歴記録先。nilでなければ実行結果を追記する
	History *[]HistoryEntry
}

// NewManager は新しいActivityManagerを作成する
func NewManager(l *logger.Logger) *Manager {
	if l == nil {
		l = logger.New(logger.CategoryAction)
	}
	return &Manager{
		currentActivities: make(map[ecs.Entity]*Activity),
		logger:            l,
	}
}

// HistoryEntry は実行されたアクティビティの記録
// テスト用に公開している
type HistoryEntry struct {
	Activity Behavior    // 実行されたアクティビティ
	Actor    ecs.Entity  // 実行者
	Target   *ecs.Entity // 対象（あれば）
	Success  bool        // 成功/失敗
	Message  string      // 結果メッセージ
}

// Execute は指定されたアクション（アクティビティ）を実行する
// 即座実行アクション（移動、攻撃等）も継続アクション（休息等）も統一的に処理
func (m *Manager) Execute(actorImpl Behavior, params ActionParams, world w.World) (*ActionResult, error) {
	activityName := actorImpl.String()
	m.logger.Debug("アクション実行開始",
		"type", activityName,
		"actor", params.Actor)

	// アクティビティを作成
	activity := m.createActivity(actorImpl, params, world)

	// アクティビティを開始
	if err := m.StartActivity(activity, world); err != nil {
		result := &ActionResult{
			Success:      false,
			ActivityName: activityName,
			Message:      err.Error(),
		}
		m.addHistory(actorImpl, params, result)
		return result, err
	}

	// 即座実行アクション（1ターン）の場合は即座に処理
	if activity.TurnsTotal == 1 {
		// ターン処理実行
		m.ProcessTurn(world)

		// ターン管理システムに移動コストを通知
		m.consumeMoveCost(world, actorImpl, params.Actor)

		// 結果を確認
		currentActivity := m.GetCurrentActivity(params.Actor)
		if currentActivity == nil || currentActivity.IsCompleted() {
			result := &ActionResult{
				Success:      true,
				ActivityName: activityName,
				Message:      "アクション完了",
			}
			m.addHistory(actorImpl, params, result)
			return result, nil
		} else if currentActivity.IsCanceled() {
			result := &ActionResult{
				Success:      false,
				ActivityName: activityName,
				Message:      currentActivity.CancelReason,
			}
			m.addHistory(actorImpl, params, result)
			return result, fmt.Errorf("アクション失敗: %s", currentActivity.CancelReason)
		}
	}

	// 継続アクションの場合は開始成功を返す
	result := &ActionResult{
		Success:      true,
		ActivityName: activityName,
		Message:      "アクション開始",
	}
	m.addHistory(actorImpl, params, result)
	return result, nil
}

// addHistory は履歴エントリを追加する
func (m *Manager) addHistory(actorImpl Behavior, params ActionParams, result *ActionResult) {
	if m.History == nil {
		return
	}
	*m.History = append(*m.History, HistoryEntry{
		Activity: actorImpl,
		Actor:    params.Actor,
		Target:   params.Target,
		Success:  result.Success,
		Message:  result.Message,
	})
}

// StartActivity は新しいアクティビティを開始する
func (m *Manager) StartActivity(activity *Activity, world w.World) error {
	if activity == nil {
		return ErrActivityNil
	}

	// 既存のアクティビティがある場合は中断
	if currentActivity := m.GetCurrentActivity(activity.Actor); currentActivity != nil {
		if err := m.InterruptActivity(activity.Actor, "新しいアクティビティを開始"); err != nil {
			m.logger.Warn("既存アクティビティの中断に失敗", "entity", activity.Actor, "error", err.Error())
		}
	}

	// アクティビティアクターでの検証
	if err := activity.ActorImpl.Validate(activity, world); err != nil {
		return fmt.Errorf("アクティビティ検証失敗: %w", err)
	}

	// 基本的な必須項目チェック
	if err := m.validateBasicRequirements(activity); err != nil {
		return fmt.Errorf("基本要件検証失敗: %w", err)
	}

	// アクティビティを登録
	m.currentActivities[activity.Actor] = activity
	activity.State = ActivityStateRunning

	// アクティビティアクターのStart処理を実行
	if err := activity.ActorImpl.Start(activity, world); err != nil {
		// 開始に失敗した場合はクリーンアップ
		delete(m.currentActivities, activity.Actor)
		return fmt.Errorf("アクティビティ開始失敗: %w", err)
	}

	m.logger.Debug("アクティビティ開始",
		"entity", activity.Actor,
		"type", activity.ActorImpl.String(),
		"duration", activity.TurnsTotal)

	return nil
}

// GetCurrentActivity は指定されたエンティティの現在のアクティビティを取得する
func (m *Manager) GetCurrentActivity(entity ecs.Entity) *Activity {
	return m.currentActivities[entity]
}

// HasActivity は指定されたエンティティがアクティビティを実行中かを返す
func (m *Manager) HasActivity(entity ecs.Entity) bool {
	activity := m.GetCurrentActivity(entity)
	return activity != nil && activity.IsActive()
}

// InterruptActivity は指定されたエンティティのアクティビティを中断する
func (m *Manager) InterruptActivity(entity ecs.Entity, reason string) error {
	activity := m.GetCurrentActivity(entity)
	if activity == nil {
		return ErrActivityNotFound
	}

	return activity.Interrupt(reason)
}

// ResumeActivity は指定されたエンティティのアクティビティを再開する
func (m *Manager) ResumeActivity(entity ecs.Entity, world w.World) error {
	activity := m.GetCurrentActivity(entity)
	if activity == nil {
		return ErrActivityNotFound
	}

	// 再開条件をチェック
	if err := m.validateResume(activity, world); err != nil {
		return fmt.Errorf("アクティビティ再開検証失敗: %w", err)
	}

	return activity.Resume()
}

// CancelActivity は指定されたエンティティのアクティビティをキャンセルする
func (m *Manager) CancelActivity(entity ecs.Entity, reason string, world w.World) {
	activity := m.GetCurrentActivity(entity)
	if activity == nil {
		return
	}

	// アクティビティアクターを取得してCanceled処理を実行
	if err := activity.ActorImpl.Canceled(activity, world); err != nil {
		m.logger.Warn("アクティビティキャンセル処理エラー",
			"entity", entity,
			"error", err.Error())
	}

	// アクティビティ自体をキャンセル状態に
	activity.Cancel(reason)
	delete(m.currentActivities, entity)

	m.logger.Debug("アクティビティキャンセル",
		"entity", entity,
		"type", activity.ActorImpl.String(),
		"reason", reason)
}

// ProcessTurn は全てのアクティブなアクティビティの1ターン分の処理を実行する
func (m *Manager) ProcessTurn(world w.World) {
	m.logger.Debug("アクティビティターン処理開始", "count", len(m.currentActivities))

	// 完了・キャンセルされたアクティビティを削除するためのリスト
	var toRemove []ecs.Entity

	for entity, activity := range m.currentActivities {
		// アクティブなアクティビティのみ処理
		if !activity.IsActive() {
			if activity.IsCompleted() || activity.IsCanceled() {
				toRemove = append(toRemove, entity)
			}
			continue
		}

		// ターン処理を実行
		if err := activity.ActorImpl.DoTurn(activity, world); err != nil {
			m.logger.Error("アクティビティターン処理エラー",
				"entity", entity,
				"type", activity.ActorImpl.String(),
				"error", err.Error())

			// エラーが発生した場合はキャンセル
			m.CancelActivity(entity, fmt.Sprintf("エラー: %s", err.Error()), world)
			toRemove = append(toRemove, entity)
			continue
		}

		// 完了したアクティビティの処理
		if activity.IsCompleted() {
			// Finish処理を実行
			if err := activity.ActorImpl.Finish(activity, world); err != nil {
				m.logger.Error("アクティビティ完了処理エラー",
					"entity", entity,
					"type", activity.ActorImpl.String(),
					"error", err.Error())
			}

			m.logger.Debug("アクティビティ完了",
				"entity", entity,
				"type", activity.ActorImpl.String())
			toRemove = append(toRemove, entity)
		}
	}

	// 完了・キャンセルされたアクティビティを削除
	for _, entity := range toRemove {
		delete(m.currentActivities, entity)
	}

	m.logger.Debug("アクティビティターン処理完了", "removed", len(toRemove))
}

// GetActivitySummary はアクティビティの要約情報を取得する
func (m *Manager) GetActivitySummary() map[string]interface{} {
	summary := make(map[string]interface{})

	var activeCount, pausedCount, totalCount int
	for _, activity := range m.currentActivities {
		totalCount++
		switch activity.State {
		case ActivityStateRunning:
			activeCount++
		case ActivityStatePaused:
			pausedCount++
		case ActivityStateCompleted, ActivityStateCanceled:
			// 完了/キャンセル状態はカウントしない
		}
	}

	summary["total"] = totalCount
	summary["active"] = activeCount
	summary["paused"] = pausedCount

	return summary
}

// validateBasicRequirements はアクティビティの基本要件を検証する
// 詳細な検証は各アクティビティのValidateメソッドで行う
func (m *Manager) validateBasicRequirements(activity *Activity) error {
	// 基本的なnilチェックのみ実行
	if activity == nil {
		return ErrActivityNil
	}

	return nil
}

// validateResume はアクティビティの再開可能性を検証する
func (m *Manager) validateResume(activity *Activity, world w.World) error {
	if !activity.CanResume() {
		return fmt.Errorf("アクティビティ '%s' は再開できません", activity.GetDisplayName())
	}

	// アクティビティアクターでの検証を再実行
	if err := activity.ActorImpl.Validate(activity, world); err != nil {
		return fmt.Errorf("再開時検証失敗: %w", err)
	}

	// 基本要件を再チェック
	return m.validateBasicRequirements(activity)
}

// createActivity はアクティビティ実装とパラメータからアクティビティを作成する
func (m *Manager) createActivity(actorImpl Behavior, params ActionParams, world w.World) *Activity {
	// 基本のdurationを計算
	duration := params.Duration
	if duration <= 0 {
		characterAP := m.getEntityMaxAP(params.Actor, world)
		duration = CalculateRequiredTurns(actorImpl, characterAP)
	}

	// アクティビティを作成
	activity := NewActivity(actorImpl, params.Actor, duration)

	// パラメータを設定
	if params.Destination != nil {
		activity.Position = params.Destination
	}
	if params.Target != nil {
		activity.Target = params.Target
	}

	return activity
}

// consumeMoveCost はアクションのAPコストを消費する
func (m *Manager) consumeMoveCost(world w.World, actorImpl Behavior, actor ecs.Entity) {
	info := actorImpl.Info()
	cost := info.ActionPointCost

	// TurnBasedコンポーネントから直接APを消費
	tbComp := world.Components.TurnBased.Get(actor)
	if tbComp == nil {
		m.logger.Debug("TurnBasedコンポーネントがない", "actor", actor)
		return
	}

	tb := tbComp.(*gc.TurnBased)
	tb.AP.Current -= cost

	m.logger.Debug("移動コスト消費",
		"activity", actorImpl.String(),
		"cost", cost,
		"remaining", tb.AP.Current,
		"actor", actor,
		"isPlayer", actor.HasComponent(world.Components.Player))
}

// getEntityMaxAP はエンティティの最大AP値を取得する
func (m *Manager) getEntityMaxAP(entity ecs.Entity, world w.World) int {
	if turnBasedComponent := world.Components.TurnBased.Get(entity); turnBasedComponent != nil {
		turnBased := turnBasedComponent.(*gc.TurnBased)
		return turnBased.AP.Max
	}
	m.logger.Debug("TurnBasedコンポーネントが見つからない", "entity", entity)
	return 100 // デフォルトAP値
}

// ActionParams はアクション実行時のパラメータを表す
type ActionParams struct {
	Actor       ecs.Entity   // アクションを実行するエンティティ
	Target      *ecs.Entity  // 対象エンティティ（攻撃等で使用）
	Destination *gc.Position // 対象位置（移動等で使用）
	Duration    int          // 継続時間（休息、待機等で使用）
	Reason      string       // 理由（待機等で使用）
}

// ActionResult はアクション実行結果を表す
type ActionResult struct {
	Success      bool   // 実行成功/失敗
	ActivityName string // 実行されたアクティビティ名
	Message      string // 結果メッセージ
}
