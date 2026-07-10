package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// log はactivityパッケージ用のロガー
var log = logger.New(logger.CategoryAction)

// behaviors はProcessTurn経由の継続アクション処理で使うシングルトンBehaviorのマップ。
// Execute経路で使うBuildActivityはこのマップのインスタンスでは呼ばれない
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
	gc.BehaviorRead:      &ReadActivity{},
	gc.BehaviorShoot:     &ShootActivity{},
	gc.BehaviorReload:    &ReloadActivity{},
	gc.BehaviorTransfer:  &TransferActivity{},
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
	BuildActivity(actor ecs.Entity, world w.World) (*gc.Activity, error)
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

// requireDestination はActivityのDestinationからタイル座標を取得する。
// Destinationが未設定の場合はエラーを返す
func requireDestination(comp *gc.Activity) (consts.Coord[consts.Tile], error) {
	if comp.Destination == nil {
		return consts.Coord[consts.Tile]{}, fmt.Errorf("目的地が指定されていません")
	}
	return consts.Coord[consts.Tile]{X: comp.Destination.X, Y: comp.Destination.Y}, nil
}

// progressHunger はターン経過による空腹進行を処理する
func progressHunger(actor ecs.Entity, world w.World) {
	if !world.Components.Player.Has(actor) {
		return
	}
	hungerComp := world.Components.Hunger.Get(actor)
	if hungerComp == nil {
		return
	}
	hunger := hungerComp.(*gc.Hunger)

	hungerPct := 100
	if modsComp := world.Components.CharModifiers.Get(actor); modsComp != nil {
		hungerPct = modsComp.(*gc.CharModifiers).HungerProgress
	}
	if world.Config.RNG.IntN(100) < hungerPct {
		hunger.Decrease(1)
	}
}

// isAreaSafe はアクターの周囲に敵対エンティティがいないかチェックする
func isAreaSafe(actor ecs.Entity, world w.World) bool {
	gridElement := world.Components.GridElement.Get(actor)
	if gridElement == nil {
		return false
	}

	actorGrid := gridElement.(*gc.GridElement)
	actorX, actorY := int(actorGrid.X), int(actorGrid.Y)

	safeRadius := 1
	hasHostile := false

	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if hasHostile {
			return
		}
		if query.FactionRelation(world, actor, entity) != query.RelationHostile {
			return
		}
		grid := world.Components.GridElement.Get(entity)
		dx, dy := int(grid.X)-actorX, int(grid.Y)-actorY
		if dx >= -safeRadius && dx <= safeRadius && dy >= -safeRadius && dy <= safeRadius {
			hasHostile = true
		}
	}))

	return !hasHostile
}
