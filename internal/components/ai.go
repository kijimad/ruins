package components

import (
	ecs "github.com/x-hgg-x/goecs/v2"
)

// AIMoveFSM はAI移動の有限状態マシン
type AIMoveFSM struct {
	// AIシステムによる制御を示すマーカーコンポーネント
}

// AIRoamingSubState はAI徘徊行動のサブ状態を表す
type AIRoamingSubState string

const (
	// AIRoamingWaiting はAI徘徊における待機状態
	AIRoamingWaiting = AIRoamingSubState("WAIT")
	// AIRoamingDriving はAI徘徊における移動状態
	AIRoamingDriving = AIRoamingSubState("DRIVING")
	// AIRoamingChasing はプレイヤーを追跡する状態
	AIRoamingChasing = AIRoamingSubState("CHASING")
)

// AIVision はAIの視界システム
type AIVision struct {
	// ViewDistance は視界距離（ピクセル単位）
	ViewDistance Pixel
	// TargetEntity は追跡対象のエンティティ（プレイヤーなど）
	TargetEntity *ecs.Entity
}

// AIRoaming はAI移動で歩き回り状態
type AIRoaming struct {
	SubState AIRoamingSubState
	// サブステートの開始ターン
	StartSubStateTurn int
	// サブステートの持続ターン数
	DurationSubStateTurns int
}

// AIChasing は追跡状態のコンポーネント
type AIChasing struct {
	// TargetX は追跡対象のX座標
	TargetX float64
	// TargetY は追跡対象のY座標
	TargetY float64
	// LastSeenTurn は最後に視認したターン
	LastSeenTurn int
}
