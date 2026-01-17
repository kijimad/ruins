package turns

import (
	gc "github.com/kijimaD/ruins/lib/components"
	"github.com/kijimaD/ruins/lib/logger"
	w "github.com/kijimaD/ruins/lib/world"
	"github.com/kijimaD/ruins/lib/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// TurnPhase はターンの段階を表す
type TurnPhase int

const (
	// PlayerTurn プレイヤーのターン（入力待ち・アクション実行）
	PlayerTurn TurnPhase = iota
	// AITurn AI・NPCのターン（一括処理）
	AITurn
	// TurnEnd ターン終了処理
	TurnEnd
)

func (tp TurnPhase) String() string {
	switch tp {
	case PlayerTurn:
		return "PlayerTurn"
	case AITurn:
		return "AITurn"
	case TurnEnd:
		return "TurnEnd"
	default:
		return "Unknown"
	}
}

// TurnManager はターンの管理を行う
type TurnManager struct {
	PlayerMoves int       // プレイヤーの残り移動ポイント
	TurnPhase   TurnPhase // 現在のターンフェーズ
	TurnNumber  int       // ターン番号
	logger      *logger.Logger
}

// NewTurnManager は新しいTurnManagerを作成する
func NewTurnManager() *TurnManager {
	return &TurnManager{
		PlayerMoves: 100,
		TurnPhase:   PlayerTurn,
		TurnNumber:  1,
		logger:      logger.New(logger.CategoryTurn),
	}
}

// CanPlayerAct はプレイヤーがアクションを実行可能かチェックする
func (tm *TurnManager) CanPlayerAct() bool {
	return tm.TurnPhase == PlayerTurn && tm.PlayerMoves > 0
}

// ConsumePlayerMoves はプレイヤーの移動ポイントを消費する
func (tm *TurnManager) ConsumePlayerMoves(actionName string, cost int) {
	tm.PlayerMoves -= cost

	tm.logger.Debug("プレイヤー移動ポイント消費",
		"action", actionName,
		"cost", cost,
		"remaining", tm.PlayerMoves)

	// 移動ポイントが尽きた場合はAIターンに移行
	if tm.PlayerMoves <= 0 {
		tm.TurnPhase = AITurn
		tm.logger.Debug("AIターンに移行", "turn", tm.TurnNumber)
	}
}

// AdvanceToAITurn はAIターンに強制移行する（待機など）
func (tm *TurnManager) AdvanceToAITurn() {
	tm.TurnPhase = AITurn
	tm.logger.Debug("AIターンに強制移行", "turn", tm.TurnNumber)
}

// AdvanceToTurnEnd はターン終了に移行する
func (tm *TurnManager) AdvanceToTurnEnd() {
	tm.TurnPhase = TurnEnd
	tm.logger.Debug("ターン終了処理", "turn", tm.TurnNumber)
}

// StartNewTurn は新しいターンを開始する
func (tm *TurnManager) StartNewTurn() {
	tm.TurnNumber++
	tm.PlayerMoves = 100
	tm.TurnPhase = PlayerTurn

	tm.logger.Debug("新ターン開始",
		"turn", tm.TurnNumber,
		"player_moves", tm.PlayerMoves)
}

// IsPlayerTurn はプレイヤーのターンかチェックする
func (tm *TurnManager) IsPlayerTurn() bool {
	return tm.TurnPhase == PlayerTurn
}

// IsAITurn はAIのターンかチェックする
func (tm *TurnManager) IsAITurn() bool {
	return tm.TurnPhase == AITurn
}

// ConsumeActionPoints はエンティティのアクションポイントを消費する
// CDDAスタイルの共通AP管理システム
func (tm *TurnManager) ConsumeActionPoints(world w.World, entity ecs.Entity, actionName string, cost int) bool {
	// ActionPointsコンポーネントを取得
	apComponent := world.Components.TurnBased.Get(entity)
	if apComponent == nil {
		// TODO: 直す
		// ActionPointsコンポーネントがない場合は従来のプレイヤー専用処理
		if entity.HasComponent(world.Components.Player) {
			tm.ConsumePlayerMoves(actionName, cost)
			return true
		}
		return false
	}

	actionPoints := apComponent.(*gc.TurnBased)

	// AP不足チェック
	if actionPoints.AP.Current < cost {
		return false
	}

	// AP消費
	actionPoints.AP.Current -= cost

	tm.logger.Debug("アクションポイント消費",
		"entity", entity,
		"action", actionName,
		"cost", cost,
		"remaining", actionPoints.AP.Current)

	return true
}

// CanEntityAct はエンティティがアクション可能かチェックする
func (tm *TurnManager) CanEntityAct(world w.World, entity ecs.Entity, cost int) bool {
	// ActionPointsコンポーネントを取得
	apComponent := world.Components.TurnBased.Get(entity)
	if apComponent == nil {
		// ActionPointsコンポーネントがない場合は従来のプレイヤー専用処理
		if entity.HasComponent(world.Components.Player) {
			return tm.CanPlayerAct()
		}
		// 敵もAP制限を受ける
		return false
	}

	actionPoints := apComponent.(*gc.TurnBased)

	return actionPoints.AP.Current >= cost
}

// RestoreAllActionPoints は全エンティティのAPを回復する（ターン終了時）
func (tm *TurnManager) RestoreAllActionPoints(world w.World) error {
	// 複数ある場合は最後のエラー
	var err error
	// ActionPointsコンポーネントを持つ全エンティティのAP回復
	world.Manager.Join(world.Components.TurnBased).Visit(ecs.Visit(func(entity ecs.Entity) {
		actionPoints := world.Components.TurnBased.Get(entity).(*gc.TurnBased)
		maxAP, calcErr := worldhelper.CalculateMaxActionPoints(world, entity)
		err = calcErr

		actionPoints.AP.Current = maxAP
		actionPoints.AP.Max = maxAP
		tm.logger.Debug("アクションポイント回復",
			"entity", entity,
			"restored", maxAP)
	}))
	return err
}
