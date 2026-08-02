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

// GetBehavior は gc.Activity に永続化された BehaviorName から実装を復元する。
// 単なる名前と実装の対応付けで、毎回ゼロ値の新しいインスタンスを返す。
// Behavior は状態を持たない振る舞いの束なので、これで問題ない。着手後のライフサイクル
// Validate/Start/DoTurn/Finish/Canceled はこのゼロ値インスタンスで回るため、per-アクティビティの
// 状態はインスタンスのフィールドに持たず gc.Activity 側に持たせる。着手パラメータを持つのは
// BuildActivity へ渡す呼び出し側インスタンスだけ。
//
// default 句は置かない。新しい BehaviorName を足したとき case 追加漏れを exhaustive linter に
// 検知させるため。未知の名前は switch を抜けた後のエラーで扱う。
func GetBehavior(name gc.BehaviorName) (Behavior, error) {
	switch name {
	case gc.BehaviorMove:
		return &MoveBehavior{}, nil
	case gc.BehaviorAttack:
		return &AttackBehavior{}, nil
	case gc.BehaviorRest:
		return &RestBehavior{}, nil
	case gc.BehaviorWait:
		return &WaitBehavior{}, nil
	case gc.BehaviorPickup:
		return &PickupBehavior{}, nil
	case gc.BehaviorDrop:
		return &DropBehavior{}, nil
	case gc.BehaviorUseItem:
		return &UseItemBehavior{}, nil
	case gc.BehaviorTalk:
		return &TalkBehavior{}, nil
	case gc.BehaviorOpenDoor:
		return &OpenDoorBehavior{}, nil
	case gc.BehaviorCloseDoor:
		return &CloseDoorBehavior{}, nil
	case gc.BehaviorRead:
		return &ReadBehavior{}, nil
	case gc.BehaviorShoot:
		return &ShootBehavior{}, nil
	case gc.BehaviorReload:
		return &ReloadBehavior{}, nil
	case gc.BehaviorTransfer:
		return &TransferBehavior{}, nil
	case gc.BehaviorDisassemble:
		return &DisassembleBehavior{}, nil
	case gc.BehaviorPush:
		return &PushBehavior{}, nil
	case gc.BehaviorPull:
		return &PullBehavior{}, nil
	case gc.BehaviorPortal, gc.BehaviorDoorLock, gc.BehaviorStorage:
		// ExecuteInteraction が直接処理する結果ラベルで、対応する Behavior 実装は持たない
	}
	return nil, fmt.Errorf("未登録のBehavior: %s", name)
}

// Behavior はアクティビティの実行を担当するインターフェース。
//
// メソッドは2種類に分かれる。BuildActivity だけは着手時に呼び出し側が生成した
// インスタンスで呼ばれ、そのフィールドを着手パラメータとして読んでよい。残りの
// Validate/Start/DoTurn/Finish/Canceled は GetBehavior が毎回作るゼロ値インスタンスで
// 回るため、インスタンスのフィールドはゼロ値であり読んではいけない。継続する状態は
// すべて gc.Activity に持たせる。
//
// この非対称ゆえ GetBehavior が返すインスタンスで BuildActivity を呼んではならない。
// 着手パラメータがゼロ値になり、Duration 依存の Behavior などがエラーになる。
type Behavior interface {
	Info() Info
	Name() gc.BehaviorName
	// BuildActivity は着手時の呼び出し側インスタンスでのみ呼ぶ。GetBehavior が返すインスタンスで呼ばない
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
// required は完了に必要な総量。初回ステップで満ちれば即時アクションになる。
func NewActivity(behavior Behavior, required int) (*gc.Activity, error) {
	if required < 0 {
		return nil, ErrInvalidRequired
	}

	return &gc.Activity{
		BehaviorName: behavior.Name(),
		State:        gc.ActivityStateRunning,
		Progress:     gc.IntPool{Max: required},
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
	if comp.Progress.Max <= 0 {
		return 100.0
	}
	return (float64(comp.Progress.Current) / float64(comp.Progress.Max)) * 100.0
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
