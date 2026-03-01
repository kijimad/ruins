package components

// TurnPhase はターンの段階を表す
type TurnPhase int

const (
	// TurnPhasePlayer はプレイヤーのターン
	TurnPhasePlayer TurnPhase = iota
	// TurnPhaseAI はAI/NPCのターン
	TurnPhaseAI
	// TurnPhaseEnd はターン終了処理
	TurnPhaseEnd
)

// String はTurnPhaseの文字列表現を返す
func (tp TurnPhase) String() string {
	switch tp {
	case TurnPhasePlayer:
		return "PlayerTurn"
	case TurnPhaseAI:
		return "AITurn"
	case TurnPhaseEnd:
		return "TurnEnd"
	default:
		panic("不正なTurnPhase値")
	}
}

// TurnState はターン状態を保持する
// resources.Dungeon に格納して使用する
type TurnState struct {
	Phase      TurnPhase // 現在のターンフェーズ
	TurnNumber int       // ターン番号（1から開始）
}
