package systems

import (
	"github.com/kijimaD/ruins/internal/activity"
	"github.com/kijimaD/ruins/internal/aiinput"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
)

// TurnSystem はターン管理を行うシステム。
//
// 1ゲームターンは Player → AI → End の3フェーズで進む。Update は毎フレーム1回呼ばれ、
// 通常は1回につき1フェーズだけ進めるので、1ターン = 3フレームで消化される。
//
//	Player: プレイヤーの行動。入力を待つか、継続アクティビティを1ステップ進める
//	AI:     敵・NPC を一括処理し、視界の再計算を要求する
//	End:    AP回復・空腹・気温・GameTime を1ターン進める
//
// 継続アクティビティ中(押し・休息・分解など)は例外で、fastForwardActivity が完了・中断・
// 上限まで1フレーム内で複数ターンをまとめて回す。各ターンは通常と同じ3ステップを通すので、
// 敵・時間の進行も毎ターンの中断判定も保たれ、ゲーム上の結果は1フレーム1ターンと変わらない。
// 縮むのは実時間で、省かれるのは途中ターンの描画だけ。継続中の入力は DungeonState が
// HasActivity で塞ぐため、フェーズが Player のままでも操作は受け付けない。
type TurnSystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys TurnSystem) String() string {
	return "TurnSystem"
}

// fastForwardTurnsPerFrame は継続アクティビティ中に1フレームで進める最大ターン数。
// 大量ターンの押し・休息・分解を実時間で待たせないよう複数ターンをまとめて進めつつ、
// 1フレームの処理が跳ね上がらないよう上限を設ける。上限を超えても次フレームで続きを進める。
const fastForwardTurnsPerFrame = 100

// Update はターン管理を行う
// w.Updater interfaceを実装
func (sys *TurnSystem) Update(world w.World) error {
	turnState := query.GetTurnState(world)

	switch turnState.Phase {
	case gc.TurnPhasePlayer:
		// プレイヤーが継続アクション中なら、実時間を食わないよう複数ターンを一気に進める。
		// 各ターンは通常と同じ 継続ステップ→AI→終了 を回すので、敵・NPC・時間は同じ速さで進み、
		// 敵接近などによる毎ターンの中断判定も保たれる。縮むのは実時間だけ。
		if playerHasActivity(world) {
			return sys.fastForwardActivity(world, turnState)
		}
		// APが最小行動コストを満たさない場合は自動でターンを終了
		if shouldAutoEndTurn(world) {
			turnState.Phase = gc.TurnPhaseAI
			return nil
		}
		// プレイヤー入力処理はDungeonStateで実行される
	case gc.TurnPhaseAI:
		if err := runAIPhase(world); err != nil {
			return err
		}
		turnState.Phase = gc.TurnPhaseEnd
	case gc.TurnPhaseEnd:
		if err := runEndPhase(world, turnState); err != nil {
			return err
		}
		turnState.Phase = gc.TurnPhasePlayer
	}
	return nil
}

// fastForwardActivity はプレイヤーの継続アクティビティを、完了・中断・上限まで1フレーム内で
// 複数ターン進める。1ターンは通常フローと同じ 継続ステップ→AI→ターン終了 を回すので、
// ゲーム上の結果は1フレーム1ターンで進めた場合と変わらず、縮むのは実時間だけ。
func (sys *TurnSystem) fastForwardActivity(world w.World, turnState *gc.TurnState) error {
	for range fastForwardTurnsPerFrame {
		if !playerHasActivity(world) {
			break // 完了・中断したら通常進行へ戻す
		}
		processPlayerContinuousActivity(world)
		if err := runAIPhase(world); err != nil {
			return err
		}
		if err := runEndPhase(world, turnState); err != nil {
			return err
		}
	}
	// フェーズは Player のまま。継続中なら次フレームで続きを、完了なら通常の入力待ちへ。
	// 継続中の入力は DungeonState が HasActivity で塞ぐのでフェーズが Player でも受け付けない。
	turnState.Phase = gc.TurnPhasePlayer
	return nil
}

// runAIPhase は全AI・NPCを一括処理し、視界の再計算を要求する。
func runAIPhase(world w.World) error {
	// AIターン: 全AI・NPCを一括処理
	if err := processAITurn(world); err != nil {
		return err
	}
	// AIターン完了後に視界を再計算させる
	query.GetVisionState(world).RequestUpdate()
	return nil
}

// runEndPhase はturn end processingをして1ゲームターンを確定させる。
func runEndPhase(world w.World, turnState *gc.TurnState) error {
	if err := processTurnEnd(world); err != nil {
		return err
	}
	// 空間インデックスを無効化する。次ターンで再構築される
	query.InvalidateSpatialIndex(world)
	turnState.TurnNumber++
	// ゲーム内時間を1ターン進める。昼夜・気温の時間修正がこれに依存する。
	// GameTime は Dungeon 内で永続なのでセーブ/ロードでも一貫する
	query.GetGameTime(world).Advance()
	// 季節や日の出入りが変わったらログへ出す
	logEnvironmentChange(world)
	return nil
}

// logEnvironmentChange は季節や日の出入りが変わったらゲームログへ出す。
// 変化は現在ターンと直前ターンの導出値の差で判定するので、GameTime.Advance の直後に呼ぶ。
func logEnvironmentChange(world w.World) {
	gt := query.GetGameTime(world)

	if gt.SeasonJustChanged() {
		name := query.T(world, gt.GetSeason().String())
		gamelog.New(query.GetGameLog(world)).
			Markup(gamelog.Tag("system", query.T(world, "The season changed to %s.", name))).
			Log()
	}

	// 知らせるのは日の出と日の入りだけ。夜明け入りが日の出、夜入りが日の入り。
	// 夕は太陽がまだ空にある薄暮なので、沈み切って夜になった瞬間を日の入りとする。
	if tod, changed := gt.TimeOfDayJustChanged(); changed {
		switch tod {
		case gc.TimeDawn:
			gamelog.New(query.GetGameLog(world)).
				Markup(gamelog.Tag("system", query.T(world, "The sun rises."))).
				Log()
		case gc.TimeNight:
			gamelog.New(query.GetGameLog(world)).
				Markup(gamelog.Tag("system", query.T(world, "The sun sets."))).
				Log()
		default:
			// ほかの区分は日の出入りではないので出さない
		}
	}
}

// playerHasActivity はプレイヤーが継続アクティビティ中かを返す。
func playerHasActivity(world w.World) bool {
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return false
	}
	return query.HasActivity(world, playerEntity)
}

// shouldAutoEndTurn はプレイヤーのAPがマイナスの場合にtrueを返す
// APがマイナスの間は自動でターンを経過させる
func shouldAutoEndTurn(world w.World) bool {
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return false
	}

	turnBased := world.Components.TurnBased.Get(playerEntity)
	if turnBased == nil {
		return false
	}

	// APが最小閾値未満の場合は自動でターンを終了
	return turnBased.AP.Current < consts.MinActionThreshold
}

// processAITurn はAIターンの処理を行う
func processAITurn(world w.World) error {
	log := logger.New(logger.CategoryTurn)
	log.Debug("AI turn processing start")

	processor := aiinput.NewProcessor(world.Resources.Config.RNG)
	if err := processor.ProcessAll(world); err != nil {
		return err
	}

	log.Debug("AI turn processing done")
	return nil
}

// processTurnEnd はturn end processingを行う
func processTurnEnd(world w.World) error {
	log := logger.New(logger.CategoryTurn)
	turnState := query.GetTurnState(world)

	log.Debug("turn end processing", "turn", turnState.TurnNumber)

	// 全エンティティのアクションポイントを回復
	if err := query.RestoreAllActionPoints(world); err != nil {
		return err
	}

	// 空腹を1ターンにつき1回進める。行動種別に依らず全員が等しく空腹になる
	progressTurnHunger(world)

	return runTurnEndSystems(world)
}

// runTurnEndSystems はターン終了時に実行するシステム群を呼び出す。
// DeadCleanupSystem を末尾に置き、そのターンに死んだものを同じターンで回収する。
// 死は必ずターン処理で起きるので回収もターンに閉じる。fast-forward は毎ターンこれを回すため、
// 燃え尽きた火が数ターン残って暖め・照らし続けることがない。
func runTurnEndSystems(world w.World) error {
	for _, updater := range []w.Updater{
		&AutoInteractionSystem{},
		&TemperatureSystem{},
		&HealthRegenSystem{},
		&ConditionSystem{},
		&FireSystem{},
		&DeadCleanupSystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return err
			}
		}
	}

	return nil
}

// processPlayerContinuousActivity はプレイヤーの継続アクションを1ステップ処理する。
// 進行中なら true を返し、呼び出し側がAIフェーズへ遷移させて世界を1ターン進める
func processPlayerContinuousActivity(world w.World) bool {
	// プレイヤーエンティティを取得
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return false
	}

	// プレイヤーの継続アクションをチェック
	if !query.HasActivity(world, playerEntity) {
		return false
	}

	log := logger.New(logger.CategoryTurn)
	log.Debug("player continuous action processing")

	// 継続アクションの1ターン分を処理
	activity.ProcessContinuousActivities(world)

	if !query.HasActivity(world, playerEntity) {
		log.Debug("player continuous action done")
	}

	// 継続中でも完了でもターンコストを消費する
	query.ConsumeActionPoints(world, playerEntity, consts.StandardActionCost)
	return true
}
