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

// behaviors は gc.Activity に永続化された BehaviorName から実装を復元するレジストリ。
// 値はフィールドがゼロ値の共有シングルトンで、着手後のライフサイクル Validate/Start/DoTurn/Finish/Canceled は
// すべてこのシングルトンで回る。呼び出し側インスタンスを使うのは着手時の BuildActivity だけ。
// そのため per-アクティビティの状態はインスタンスのフィールドに持たず gc.Activity 側に持たせる。
var behaviors = map[gc.BehaviorName]Behavior{
	gc.BehaviorMove:      &MoveBehavior{},
	gc.BehaviorAttack:    &AttackBehavior{},
	gc.BehaviorRest:      &RestBehavior{},
	gc.BehaviorWait:      &WaitBehavior{},
	gc.BehaviorPickup:    &PickupBehavior{},
	gc.BehaviorDrop:      &DropBehavior{},
	gc.BehaviorUseItem:   &UseItemBehavior{},
	gc.BehaviorTalk:      &TalkBehavior{},
	gc.BehaviorOpenDoor:  &OpenDoorBehavior{},
	gc.BehaviorCloseDoor: &CloseDoorBehavior{},
	gc.BehaviorRead:      &ReadBehavior{},
	gc.BehaviorShoot:     &ShootBehavior{},
	gc.BehaviorReload:    &ReloadBehavior{},
	gc.BehaviorTransfer:  &TransferBehavior{},

	gc.BehaviorDisassemble: &DisassembleBehavior{},
	gc.BehaviorPush:        &PushBehavior{},
	gc.BehaviorPull:        &PullBehavior{},
}

// GetBehavior は名前からBehavior実装を取得する
func GetBehavior(name gc.BehaviorName) (Behavior, error) {
	b, ok := behaviors[name]
	if !ok {
		return nil, fmt.Errorf("未登録のBehavior: %s", name)
	}
	return b, nil
}

// Behavior はアクティビティの実行を担当するインターフェース。
//
// メソッドは2種類に分かれる。BuildActivity だけは着手時に呼び出し側が生成した
// インスタンスで呼ばれ、そのフィールドを着手パラメータとして読んでよい。残りの
// Validate/Start/DoTurn/Finish/Canceled は behaviors レジストリの共有シングルトンで
// 回るため、インスタンスのフィールドはゼロ値であり読んではいけない。継続する状態は
// すべて gc.Activity に持たせる。
//
// この非対称ゆえ BuildActivity をシングルトンで呼んではならない。着手パラメータが
// ゼロ値になり、Duration 依存の Behavior などがエラーになる。
type Behavior interface {
	Info() Info
	Name() gc.BehaviorName
	// BuildActivity は着手時の呼び出し側インスタンスでのみ呼ぶ。シングルトンで呼ばない
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

// NewActivity は新しいActivityコンポーネントを作成する。
// required は完了に必要な総量。即時アクションは 0 を渡す。
func NewActivity(behavior Behavior, required int) (*gc.Activity, error) {
	if required < 0 {
		return nil, ErrInvalidRequired
	}

	return &gc.Activity{
		BehaviorName: behavior.Name(),
		State:        gc.ActivityStateRunning,
		Required:     required,
	}, nil
}

// perTurnAP はアクターが継続アクションへ今ターン注げるAPを返す。
// 毎ターン再計算するためAPの変動へ追従する。取得できない場合も進行が止まらないよう最低 1 を返す。
func perTurnAP(actor ecs.Entity, world w.World) int {
	ap, err := getEntityMaxAP(actor, world)
	if err != nil || ap < 1 {
		return 1
	}
	return ap
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
	return comp.State == gc.ActivityStateCompleted
}

// IsCanceled はアクティビティがキャンセルされているかを返す
func IsCanceled(comp *gc.Activity) bool {
	return comp.State == gc.ActivityStateCanceled
}

// GetProgressPercent は進捗率を0-100の値で返す
func GetProgressPercent(comp *gc.Activity) float64 {
	if comp.Required <= 0 {
		return 100.0
	}
	return (float64(comp.Accumulated) / float64(comp.Required)) * 100.0
}

// Complete はアクティビティを完了状態にする
func Complete(comp *gc.Activity) {
	comp.State = gc.ActivityStateCompleted
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

// isAreaSafe はアクターの周囲に敵対エンティティがいないかチェックする
func isAreaSafe(actor ecs.Entity, world w.World) bool {
	if !world.Components.GridElement.Has(actor) {
		return false
	}
	gridElement := world.Components.GridElement.Get(actor)
	actorX, actorY := int(gridElement.X), int(gridElement.Y)

	safeRadius := 1
	hasHostile := false

	areaQuery := query.ActiveFilter1[gc.GridElement](world).Query()
	for areaQuery.Next() {
		entity := areaQuery.Entity()
		if hasHostile {
			continue
		}
		if query.FactionRelation(world, actor, entity) != query.RelationHostile {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		dx, dy := int(grid.X)-actorX, int(grid.Y)-actorY
		if dx >= -safeRadius && dx <= safeRadius && dy >= -safeRadius && dy <= safeRadius {
			hasHostile = true
		}
	}

	return !hasHostile
}
