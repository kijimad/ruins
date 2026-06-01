package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// DispositionType はエンティティの他者に対する態度を表す
type DispositionType string

const (
	// DispositionHostile は敵対態度を示す。視界内のプレイヤーを攻撃する
	DispositionHostile DispositionType = "hostile"
	// DispositionNeutral は中立態度を示す。攻撃されると反撃する
	DispositionNeutral DispositionType = "neutral"
	// DispositionCowardly は臆病な態度を示す。攻撃されると逃亡する
	DispositionCowardly DispositionType = "cowardly"
	// DispositionFleeing は逃亡中を示す。プレイヤーから距離を取る
	DispositionFleeing DispositionType = "fleeing"
)

// ValidAsDefault はデータ入力で指定可能なDispositionTypeかを検証する。
// DispositionFleeingはランタイム専用の値なので含めない
func (d DispositionType) ValidAsDefault() error {
	switch d {
	case DispositionHostile, DispositionNeutral, DispositionCowardly:
		return nil
	default:
		return fmt.Errorf("get %s: %w", d, ErrInvalidEnumType)
	}
}

// Disposition はエンティティの動的な態度を管理するコンポーネント
type Disposition struct {
	// Default は初期態度。逃亡後にこの値に復帰する
	Default DispositionType
	// Current は現在の態度。被ダメージなどで変化する
	Current DispositionType
}

// MovementPattern は非戦闘時の移動パターンを表す
type MovementPattern string

const (
	// MovementRandom はランダム移動。既存の動作と同じ
	MovementRandom MovementPattern = "random"
	// MovementPatrol は定点巡回。指定経路を往復する
	MovementPatrol MovementPattern = "patrol"
	// MovementWallHug は壁沿い移動。壁に沿って移動する
	MovementWallHug MovementPattern = "wallHug"
	// MovementStationary は固定。移動しない番兵タイプ
	MovementStationary MovementPattern = "stationary"
	// MovementWander は徘徊。低頻度でスポーン地点周辺をランダム移動する
	MovementWander MovementPattern = "wander"
	// MovementTerritorial は縄張り。スポーン地点から一定範囲内で移動する
	MovementTerritorial MovementPattern = "territorial"
	// MovementSwarm は群れ。同種族の仲間に寄る
	MovementSwarm MovementPattern = "swarm"
)

// Valid はMovementPatternの値が有効かを検証する
func (bs MovementPattern) Valid() error {
	switch bs {
	case MovementRandom, MovementPatrol, MovementWallHug, MovementStationary,
		MovementWander, MovementTerritorial, MovementSwarm:
		return nil
	default:
		return fmt.Errorf("get %s: %w", bs, ErrInvalidEnumType)
	}
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
