package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// log はactivityパッケージ用のロガー
var log = logger.New(logger.CategoryAction)

// behaviors は登録されたBehavior実装のマップ
var behaviors = map[gc.BehaviorName]Behavior{
	gc.BehaviorMove:      &MoveActivity{},
	gc.BehaviorAttack:    &AttackActivity{},
	gc.BehaviorRest:      &RestActivity{},
	gc.BehaviorWait:      &WaitActivity{},
	gc.BehaviorPickup:    &PickupActivity{},
	gc.BehaviorDrop:      &DropActivity{},
	gc.BehaviorUseItem:   &UseItemActivity{},
	gc.BehaviorTalk:      &TalkActivity{},
	gc.BehaviorOpenDoor:  &OpenDoorActivity{},
	gc.BehaviorCloseDoor: &CloseDoorActivity{},
}

// GetBehavior は名前からBehavior実装を取得する
func GetBehavior(name gc.BehaviorName) (Behavior, error) {
	b, ok := behaviors[name]
	if !ok {
		return nil, fmt.Errorf("未登録のBehavior: %s", name)
	}
	return b, nil
}

// Behavior はアクティビティの実行を担当するインターフェース
type Behavior interface {
	Info() Info
	Name() gc.BehaviorName
	Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error
	Start(comp *gc.Activity, actor ecs.Entity, world w.World) error
	DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error
	Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error
	Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error
}

// Info はアクティビティのメタデータを保持する
type Info struct {
	Name            string // 表示名
	Description     string // 説明文
	Interruptible   bool   // 中断可能か
	Resumable       bool   // 中断後の再開可能か
	ActionPointCost int    // 1ターン毎のアクションポイントコスト
	TotalRequiredAP int    // アクティビティ完了に必要な総AP量
}

// NewActivity は新しいActivityコンポーネントを作成する
func NewActivity(behavior Behavior, duration int) (*gc.Activity, error) {
	if duration <= 0 {
		return nil, ErrInvalidDuration
	}

	return &gc.Activity{
		BehaviorName: behavior.Name(),
		State:        gc.ActivityStateRunning,
		TurnsTotal:   duration,
		TurnsLeft:    duration,
	}, nil
}

// CalculateRequiredTurns はキャラクターのAP量に基づいて必要ターン数を計算する
func CalculateRequiredTurns(behavior Behavior, characterAP int) int {
	if behavior.Info().TotalRequiredAP > 0 && characterAP > 0 {
		return (behavior.Info().TotalRequiredAP + characterAP - 1) / characterAP
	}
	return 1
}

// CanInterrupt はアクティビティが中断可能かを返す
func CanInterrupt(comp *gc.Activity) bool {
	behavior, err := GetBehavior(comp.BehaviorName)
	if err != nil {
		return false
	}
	return behavior.Info().Interruptible && comp.State == gc.ActivityStateRunning
}

// CanResume はアクティビティが再開可能かを返す
func CanResume(comp *gc.Activity) bool {
	behavior, err := GetBehavior(comp.BehaviorName)
	if err != nil {
		return false
	}
	return behavior.Info().Resumable && comp.State == gc.ActivityStatePaused
}

// Interrupt はアクティビティを中断する
func Interrupt(comp *gc.Activity, reason string) error {
	if !CanInterrupt(comp) {
		return fmt.Errorf("アクティビティ '%s' は中断できません", GetDisplayName(comp))
	}
	comp.State = gc.ActivityStatePaused
	comp.CancelReason = reason
	return nil
}

// Resume はアクティビティを再開する
func Resume(comp *gc.Activity) error {
	if !CanResume(comp) {
		return fmt.Errorf("アクティビティ '%s' は再開できません", GetDisplayName(comp))
	}
	comp.State = gc.ActivityStateRunning
	comp.CancelReason = ""
	return nil
}

// GetDisplayName は表示用の名前を返す
func GetDisplayName(comp *gc.Activity) string {
	behavior, err := GetBehavior(comp.BehaviorName)
	if err != nil {
		return string(comp.BehaviorName)
	}
	return behavior.Info().Name
}

// IsActive はアクティビティがアクティブかを返す
func IsActive(comp *gc.Activity) bool {
	return comp.State == gc.ActivityStateRunning
}

// IsCompleted はアクティビティが完了しているかを返す
func IsCompleted(comp *gc.Activity) bool {
	return comp.State == gc.ActivityStateCompleted || comp.TurnsLeft <= 0
}

// IsCanceled はアクティビティがキャンセルされているかを返す
func IsCanceled(comp *gc.Activity) bool {
	return comp.State == gc.ActivityStateCanceled
}

// GetProgressPercent は進捗率を0-100の値で返す
func GetProgressPercent(comp *gc.Activity) float64 {
	if comp.TurnsTotal <= 0 {
		return 100.0
	}
	completed := float64(comp.TurnsTotal - comp.TurnsLeft)
	return (completed / float64(comp.TurnsTotal)) * 100.0
}

// Complete はアクティビティを完了状態にする
func Complete(comp *gc.Activity) {
	comp.State = gc.ActivityStateCompleted
	comp.TurnsLeft = 0
}

// Cancel はアクティビティをキャンセルする
func Cancel(comp *gc.Activity, reason string) {
	comp.State = gc.ActivityStateCanceled
	comp.CancelReason = reason
}
