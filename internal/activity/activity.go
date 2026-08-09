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
// 状態はインスタンスのフィールドに持たず gc.Activity 側に持たせる。着手パラメータは
// NewXxxActivity 構築関数が引数から gc.Activity の Params へ書き込む。
//
// default 句は置かない。新しい BehaviorName を足したとき case 追加漏れを exhaustive linter に
// 検知させるため。未知の名前は switch を抜けた後のエラーで扱う。
func GetBehavior(name gc.BehaviorName) (Behavior, error) {
	switch name {
	case gc.BehaviorMove:
		return &MoveBehavior{}, nil
	case gc.BehaviorMelee:
		return &MeleeBehavior{}, nil
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
	return nil, fmt.Errorf("unregistered behavior: %s", name)
}

// Behavior はアクティビティの実行を担当するインターフェース。
//
// 全メソッドは GetBehavior が毎回作るゼロ値インスタンスで回る。インスタンスは状態を持たず、
// per-アクティビティの状態はすべて gc.Activity に持たせる。着手時のパラメータは Behavior の
// フィールドではなく、NewXxxActivity 構築関数が引数から gc.Activity へ書き込む。
// そのため Behavior は状態を持たない振る舞いの束であり、構築とライフサイクルが分離している。
type Behavior interface {
	Info() Info
	Name() gc.BehaviorName
	// Validate は実行前提を検査する副作用のない純粋関数。失敗の種別を error の型で表す。
	// 返す error のうち *UserError だけがユーザ起因で、呼び出し側が Msg を gamelog へ出して
	// 操作を取り消す。弾切れ・敵接近・本が無い等がこれにあたる。
	// それ以外の error は致命的で、呼び出し側が伝播させる。構築ミスや不変条件違反、
	// 手番を得た actor の死亡などがこれにあたる。
	// 検証通過なら nil を返す。副作用はなく、gamelog も呼ばず panic もしない。
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

// NewActivity は名前と必要総量から新しいActivityコンポーネントを作成する。
// required は完了に必要な総量で、各構築関数が常に0以上を渡す。初回ステップで満ちれば即時アクションになる。
func NewActivity(name gc.BehaviorName, required int) *gc.Activity {
	return &gc.Activity{
		BehaviorName: name,
		State:        gc.ActivityStateRunning,
		Progress:     gc.IntPool{Max: required},
	}
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
		return fmt.Errorf("activity '%s' cannot be paused", GetDisplayName(comp))
	}
	comp.State = gc.ActivityStatePaused
	comp.CancelReason = reason
	return nil
}

// Resume はアクティビティを再開する
func Resume(comp *gc.Activity) error {
	if !CanResume(comp) {
		return fmt.Errorf("activity '%s' cannot be resumed", GetDisplayName(comp))
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

// requireDestination はActivityのPlaceParamsからタイル座標を取得する。
// PlaceParamsが未設定の場合はエラーを返す
func requireDestination(comp *gc.Activity) (consts.Coord[consts.Tile], error) {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok {
		return consts.Coord[consts.Tile]{}, ErrParamsTypeMismatch
	}
	return consts.Coord[consts.Tile]{X: p.Destination.X, Y: p.Destination.Y}, nil
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
