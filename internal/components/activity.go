package components

import (
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
	BehaviorMove        BehaviorName = "Move"
	BehaviorAttack      BehaviorName = "Attack"
	BehaviorRest        BehaviorName = "Rest"
	BehaviorWait        BehaviorName = "Wait"
	BehaviorPickup      BehaviorName = "Pickup"
	BehaviorDrop        BehaviorName = "Drop"
	BehaviorUseItem     BehaviorName = "UseItem"
	BehaviorTalk        BehaviorName = "Talk"
	BehaviorOpenDoor    BehaviorName = "OpenDoor"
	BehaviorCloseDoor   BehaviorName = "CloseDoor"
	BehaviorPortal      BehaviorName = "Portal"
	BehaviorDungeonGate BehaviorName = "DungeonGate"
	BehaviorRead        BehaviorName = "Read"
)

// Activity は実行中のアクティビティを保持するコンポーネント
// 1エンティティにつき最大1つのアクティビティを持つ
type Activity struct {
	BehaviorName BehaviorName  // アクティビティの種類
	State        ActivityState // 実行状態
	TurnsTotal   int           // 総必要ターン数
	TurnsLeft    int           // 残りターン数
	Target       *ecs.Entity   // 対象エンティティ
	Destination  *GridElement  // 移動先のグリッド座標
	CancelReason string        // キャンセル理由
}

// LastActivity は直近のアクティビティ実行結果を保持するコンポーネント
type LastActivity struct {
	BehaviorName BehaviorName  // 実行されたアクティビティ名
	State        ActivityState // アクティビティの終了状態
	Success      bool          // 成功/失敗
	Message      string        // 結果メッセージ
}
