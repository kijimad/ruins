package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ActivityState はアクティビティの実行状態を表す
type ActivityState int

const (
	// ActivityStateRunning はアクティビティが実行中であることを表す
	ActivityStateRunning ActivityState = iota
	// ActivityStatePaused はアクティビティが一時停止中であることを表す
	ActivityStatePaused
	// ActivityStateCompleted はアクティビティが完了したことを表す
	ActivityStateCompleted
	// ActivityStateCanceled はアクティビティがキャンセルされたことを表す
	ActivityStateCanceled
)

// ActivityInfo はアクティビティのメタデータを保持する
type ActivityInfo struct {
	Name            string // 表示名
	Description     string // 説明文
	Interruptible   bool   // 中断可能か
	Resumable       bool   // 中断後の再開可能か
	ActionPointCost int    // 1ターン毎のアクションポイントコスト
	TotalRequiredAP int    // アクティビティ完了に必要な総AP量
}

// ActivityInterface はアクティビティの実行を担当するインターフェース
// CDDAのactivity_actorを参考にした設計
type ActivityInterface interface {
	Info() ActivityInfo
	String() string
	Validate(act *Activity, world w.World) error
	Start(act *Activity, world w.World) error
	DoTurn(act *Activity, world w.World) error
	Finish(act *Activity, world w.World) error
	Canceled(act *Activity, world w.World) error
}

// Activity は継続的なアクション（アクティビティ）のデータを表す
type Activity struct {
	ActorImpl    ActivityInterface // アクティビティの実装
	State        ActivityState     // 実行状態
	TurnsTotal   int               // 総必要ターン数
	TurnsLeft    int               // 残りターン数
	Actor        ecs.Entity        // 実行者
	Target       *ecs.Entity       // 対象エンティティ（nilの場合もある）
	Position     *gc.Position      // 対象位置（nilの場合もある）
	CancelReason string            // キャンセル理由
	Logger       *logger.Logger
}

// NewActivity は新しいアクティビティを作成する
func NewActivity(actorImpl ActivityInterface, actor ecs.Entity, duration int) *Activity {
	// durationは0以下の値は許可しない（呼び出し側で適切な値を指定する必要がある）
	if duration <= 0 {
		duration = 1 // 最低1ターンは必要
	}

	return &Activity{
		ActorImpl:  actorImpl,
		State:      ActivityStateRunning,
		TurnsTotal: duration,
		TurnsLeft:  duration,
		Actor:      actor,
		Logger:     logger.New(logger.CategoryAction),
	}
}

// CalculateRequiredTurns はキャラクターのAP量に基づいて必要ターン数を計算する
func CalculateRequiredTurns(actorImpl ActivityInterface, characterAP int) int {
	info := actorImpl.Info()

	// AP積み上げ方式の場合
	if info.TotalRequiredAP > 0 && characterAP > 0 {
		// 必要総AP ÷ キャラクターのAP = 必要ターン数（切り上げ）
		return (info.TotalRequiredAP + characterAP - 1) / characterAP
	}

	// 即座実行アクションや特殊なアクションの場合は1ターン固定
	return 1
}

// CanInterrupt はアクティビティが中断可能かを返す
func (a *Activity) CanInterrupt() bool {
	info := a.ActorImpl.Info()
	return info.Interruptible && a.State == ActivityStateRunning
}

// CanResume はアクティビティが再開可能かを返す
func (a *Activity) CanResume() bool {
	info := a.ActorImpl.Info()
	return info.Resumable && a.State == ActivityStatePaused
}

// Interrupt はアクティビティを中断する
func (a *Activity) Interrupt(reason string) error {
	if !a.CanInterrupt() {
		return fmt.Errorf("アクティビティ '%s' は中断できません", a.GetDisplayName())
	}

	a.State = ActivityStatePaused
	a.CancelReason = reason

	a.Logger.Debug("アクティビティ中断",
		"type", a.ActorImpl.String(),
		"actor", a.Actor,
		"reason", reason,
		"turns_left", a.TurnsLeft)

	return nil
}

// Resume はアクティビティを再開する
func (a *Activity) Resume() error {
	if !a.CanResume() {
		return fmt.Errorf("アクティビティ '%s' は再開できません", a.GetDisplayName())
	}

	a.State = ActivityStateRunning
	a.CancelReason = ""

	a.Logger.Debug("アクティビティ再開",
		"type", a.ActorImpl.String(),
		"actor", a.Actor,
		"turns_left", a.TurnsLeft)

	return nil
}

// Cancel はアクティビティをキャンセルする
func (a *Activity) Cancel(reason string) {
	a.State = ActivityStateCanceled
	a.CancelReason = reason

	a.Logger.Debug("アクティビティキャンセル",
		"type", a.ActorImpl.String(),
		"actor", a.Actor,
		"reason", reason)
}

// Complete はアクティビティを完了状態にする
func (a *Activity) Complete() {
	a.State = ActivityStateCompleted
	a.TurnsLeft = 0

	a.Logger.Debug("アクティビティ完了",
		"type", a.ActorImpl.String(),
		"actor", a.Actor,
		"duration", a.TurnsTotal)
}

// IsActive はアクティビティがアクティブかを返す
func (a *Activity) IsActive() bool {
	return a.State == ActivityStateRunning
}

// IsCompleted はアクティビティが完了しているかを返す
func (a *Activity) IsCompleted() bool {
	return a.State == ActivityStateCompleted || a.TurnsLeft <= 0
}

// IsCanceled はアクティビティがキャンセルされているかを返す
func (a *Activity) IsCanceled() bool {
	return a.State == ActivityStateCanceled
}

// GetProgressPercent は進捗率を0-100の値で返す
func (a *Activity) GetProgressPercent() float64 {
	if a.TurnsTotal <= 0 {
		return 100.0
	}
	completed := float64(a.TurnsTotal - a.TurnsLeft)
	return (completed / float64(a.TurnsTotal)) * 100.0
}

// GetDisplayName は表示用の名前を返す
func (a *Activity) GetDisplayName() string {
	info := a.ActorImpl.Info()
	return info.Name
}

// String はActivityStateの文字列表現を返す
func (s ActivityState) String() string {
	switch s {
	case ActivityStateRunning:
		return "Running"
	case ActivityStatePaused:
		return "Paused"
	case ActivityStateCompleted:
		return "Completed"
	case ActivityStateCanceled:
		return "Canceled"
	default:
		return fmt.Sprintf("ActivityState(%d)", int(s))
	}
}

// isPlayerActivity はアクティビティの実行者がプレイヤーかを判定する
func isPlayerActivity(act *Activity, world w.World) bool {
	return act.Actor.HasComponent(world.Components.Player)
}
