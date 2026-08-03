package components

import (
	"github.com/mlange-42/ark/ecs"
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
		return "Unknown"
	}
}

// BehaviorName はアクティビティの種類を表す列挙型
type BehaviorName string

// BehaviorName の定義
const (
	BehaviorMove      BehaviorName = "Move"
	BehaviorAttack    BehaviorName = "Attack"
	BehaviorRest      BehaviorName = "Rest"
	BehaviorWait      BehaviorName = "Wait"
	BehaviorPickup    BehaviorName = "Pickup"
	BehaviorDrop      BehaviorName = "Drop"
	BehaviorUseItem   BehaviorName = "UseItem"
	BehaviorTalk      BehaviorName = "Talk"
	BehaviorOpenDoor  BehaviorName = "OpenDoor"
	BehaviorCloseDoor BehaviorName = "CloseDoor"
	BehaviorPortal    BehaviorName = "Portal"
	BehaviorDoorLock  BehaviorName = "DoorLock"
	BehaviorStorage   BehaviorName = "Storage"
	BehaviorRead      BehaviorName = "Read"
	BehaviorShoot     BehaviorName = "Shoot"
	BehaviorReload    BehaviorName = "Reload"
	BehaviorTransfer  BehaviorName = "Transfer"
	// BehaviorDisassemble は工具でpropやアイテムを分解して素材を得る
	BehaviorDisassemble BehaviorName = "Disassemble"
	// BehaviorPush は隣接する移動拠点キューブを押して動かす
	BehaviorPush BehaviorName = "Push"
	// BehaviorPull は隣接する移動拠点キューブを自分の側へ引いて動かす
	BehaviorPull BehaviorName = "Pull"
)

// Activity は実行中のアクティビティを保持するコンポーネント
// 1エンティティにつき最大1つのアクティビティを持つ
type Activity struct {
	BehaviorName BehaviorName   // アクティビティの種類
	State        ActivityState  // 実行状態
	Progress     IntPool        // Max=完了に必要な総量、Current=注ぎ込んだ総量。Current>=Max で完了。初回ステップで満ちれば即時アクションになる
	Params       ActivityParams // 各アクティビティ固有のパラメータ。無いアクションは nil
	CancelReason string         // キャンセル理由
}

// ActivityParams は各アクティビティ固有のパラメータを表すマーカーインターフェース。
// Activity は serde の skip 対象なので、interface を保持しても保存互換に影響しない。
// VisualEffect と同じく、依存の無いデータだけを components 層の interface として持つ。
// 実行ロジックを持つ Behavior は w.World に依存するため activity 層に置く。
type ActivityParams interface {
	isActivityParams()
}

// MoveParams は移動アクションのパラメータ。
type MoveParams struct {
	Destination GridElement // 移動先タイル
}

func (*MoveParams) isActivityParams() {}

// 単一の対象エンティティを取るアクションは behavior ごとに専用の Params 型を持つ。
// 対象の意味が behavior ごとに違うため、共有せず型で区別する。共有ヘルパを介さないので
// 分割してもコード共有を壊さない。

// AttackParams は近接攻撃のパラメータ。
type AttackParams struct {
	Target ecs.Entity // 攻撃対象のエンティティ
}

func (*AttackParams) isActivityParams() {}

// TalkParams は会話のパラメータ。
type TalkParams struct {
	Target ecs.Entity // 会話相手のエンティティ
}

func (*TalkParams) isActivityParams() {}

// OpenDoorParams は扉を開くアクションのパラメータ。
type OpenDoorParams struct {
	Target ecs.Entity // 開く扉のエンティティ
}

func (*OpenDoorParams) isActivityParams() {}

// CloseDoorParams は扉を閉じるアクションのパラメータ。
type CloseDoorParams struct {
	Target ecs.Entity // 閉じる扉のエンティティ
}

func (*CloseDoorParams) isActivityParams() {}

// ShootParams は射撃のパラメータ。
type ShootParams struct {
	Target ecs.Entity // 射撃対象のエンティティ
}

func (*ShootParams) isActivityParams() {}

// UseItemParams はアイテム使用のパラメータ。
type UseItemParams struct {
	Target ecs.Entity // 使用するアイテムのエンティティ
}

func (*UseItemParams) isActivityParams() {}

// ReadParams は読書のパラメータ。
type ReadParams struct {
	Target ecs.Entity // 読む本のエンティティ
}

func (*ReadParams) isActivityParams() {}

// DisassembleParams は分解のパラメータ。
type DisassembleParams struct {
	Target ecs.Entity // 分解するアイテムのエンティティ
}

func (*DisassembleParams) isActivityParams() {}

// PlaceParams は対象と操作先を取るアクションのパラメータ。ドロップ・押し引きが使う。
type PlaceParams struct {
	Target      ecs.Entity  // 操作対象のエンティティ
	Destination GridElement // 操作先タイル
}

func (*PlaceParams) isActivityParams() {}

// PickupParams は拾得アクションのパラメータ。拾う対象は呼び出し側が確定して渡す。
// 単一のアイテムでも足元一掃でも、拾うエンティティの並びを Targets として同じ形で持つ。
// タイル上の何を拾うかという決定は behavior でなく構築側にあり、標識に頼らない。
type PickupParams struct {
	Targets []ecs.Entity // 拾うエンティティ。空なら検証で弾く
}

func (*PickupParams) isActivityParams() {}

// TransferParams はアイテム転送のパラメータ。Count が0以下なら在庫全量を渡す。
type TransferParams struct {
	Target    ecs.Entity // 渡すアイテム
	Recipient ecs.Entity // 受取人エンティティ
	Count     int        // 渡す個数。0以下は全量
}

func (*TransferParams) isActivityParams() {}

// LastActivity は直近のアクティビティ実行結果を保持するコンポーネント
type LastActivity struct {
	BehaviorName BehaviorName  // 実行されたアクティビティ名
	State        ActivityState // アクティビティの終了状態
	Success      bool          // 成功/失敗
	Message      string        // 結果メッセージ
}
