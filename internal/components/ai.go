package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// AIMoveFSM はAI移動の有限状態マシン
type AIMoveFSM struct {
	// AIシステムによる制御を示すマーカーコンポーネント
}

// AIStateSubState はAI行動のサブ状態を表す
type AIStateSubState string

const (
	// AIStateWaiting は待機状態
	AIStateWaiting = AIStateSubState("WAIT")
	// AIStateDriving は移動状態
	AIStateDriving = AIStateSubState("DRIVING")
	// AIStateChasing は追跡状態
	AIStateChasing = AIStateSubState("CHASING")
	// AIStateFleeing は逃亡状態
	AIStateFleeing = AIStateSubState("FLEEING")
)

// AIVision はAIの視界システム
type AIVision struct {
	// ViewDistance は視界距離（タイル単位）
	ViewDistance consts.Tile
	// TargetEntity は追跡対象のエンティティ（プレイヤーなど）
	TargetEntity *ecs.Entity
}

// AIState はAIのランタイム状態を保持する
type AIState struct {
	SubState AIStateSubState
	// サブステートの開始ターン
	StartSubStateTurn int
	// サブステートの持続ターン数
	DurationSubStateTurns int
	// スポーン地点。Territorial移動の範囲基準に使う
	SpawnX, SpawnY int
	// 巡回方向。Patrol移動で現在の進行方向を保持する
	PatrolDirX, PatrolDirY int
}
