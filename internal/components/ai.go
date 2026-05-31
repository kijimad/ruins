package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// DispositionType はエンティティの他者に対する態度を表す
type DispositionType int

const (
	// DispositionHostile は敵対態度を示す。視界内のプレイヤーを攻撃する
	DispositionHostile DispositionType = iota
	// DispositionNeutral は中立態度を示す。攻撃されると反撃する
	DispositionNeutral
	// DispositionCowardly は臆病な態度を示す。攻撃されると逃亡する
	DispositionCowardly
	// DispositionFleeing は逃亡中を示す。プレイヤーから距離を取る
	DispositionFleeing
)

// Disposition はエンティティの動的な態度を管理するコンポーネント
type Disposition struct {
	// Default は初期態度。逃亡後にこの値に復帰する
	Default DispositionType
	// Current は現在の態度。被ダメージなどで変化する
	Current DispositionType
}

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
	// AIRoamingFleeing はプレイヤーから逃亡する状態
	AIRoamingFleeing = AIRoamingSubState("FLEEING")
)

// AIVision はAIの視界システム
type AIVision struct {
	// ViewDistance は視界距離（ピクセル単位）
	ViewDistance consts.Pixel
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
