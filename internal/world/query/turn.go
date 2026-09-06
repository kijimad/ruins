package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// Speed計算係数
const (
	speedBaseValue         = 100 // Speed計算の基本値
	speedAgilityMultiply   = 2   // Speed計算の敏捷係数
	speedDexterityMultiply = 1   // Speed計算の器用係数
	speedMinimum           = 25  // Speedの最小値（基本値の1/4）
)

// CanPlayerAct はプレイヤーが行動可能かを判定する
// プレイヤーターンかつAP >= 0 の場合にtrueを返す
func CanPlayerAct(world w.World) bool {
	// プレイヤーターンでなければ行動不可
	turnState := GetTurnState(world)
	if turnState == nil || turnState.Phase != gc.TurnPhasePlayer {
		return false
	}

	// プレイヤーのAPをチェック
	playerEntity, err := GetPlayerEntity(world)
	if err != nil {
		return false
	}

	tb := world.Components.TurnBased.Get(playerEntity)
	if tb == nil {
		return false
	}
	return tb.AP.Current >= 0
}

// ConsumeActionPoints はエンティティのアクションポイントを消費する。
// TurnBasedを持たない（生存）エンティティには false を返す。
// entity の生存は呼び出し側が保証すること。Ark の Has/Get は死亡エンティティで panic する
func ConsumeActionPoints(world w.World, entity ecs.Entity, cost int) bool {
	if !world.Components.TurnBased.Has(entity) {
		return false
	}
	tb := world.Components.TurnBased.Get(entity)
	tb.AP.Current -= cost

	log := logger.New(logger.CategoryTurn)
	log.Debug("action points consumed",
		"entity", entity,
		"cost", cost,
		"remaining", tb.AP.Current)

	return true
}

// RestoreAllActionPoints は全エンティティのAPを回復する
func RestoreAllActionPoints(world w.World) error {
	log := logger.New(logger.CategoryTurn)
	var err error

	// 退避中ステージの敵はAP回復しない。現ステージのみ対象にする
	turnBasedQuery := ActiveFilter1[gc.TurnBased](world).Query()
	for turnBasedQuery.Next() {
		entity := turnBasedQuery.Entity()
		tb := world.Components.TurnBased.Get(entity)

		// MaxAPとSpeedを計算
		maxAP, calcErr := CalculateMaxActionPoints(world, entity)
		if calcErr != nil {
			err = calcErr
			continue
		}

		speed := CalculateSpeed(world, entity)
		tb.Speed = speed
		tb.AP.Max = maxAP

		// 現在AP + Speed で上限まで回復
		newAP := min(tb.AP.Current+speed, maxAP)
		tb.AP.Current = newAP

		log.Debug("action points restored",
			"entity", entity,
			"speed", speed,
			"current", tb.AP.Current,
			"max", maxAP)
	}

	return err
}

// CalculateMaxActionPoints はエンティティの最大アクションポイントを計算する
// 敏捷性を重視したAP計算式
func CalculateMaxActionPoints(world w.World, entity ecs.Entity) (int, error) {
	abils := world.Components.Abilities.Get(entity)
	if abils == nil {
		return 0, fmt.Errorf("abilities are not set")
	}

	baseAP := 100
	agilityMultiplier := 3
	dexterityMultiplier := 1

	calculatedAP := max(baseAP+abils.Agility.Total*agilityMultiplier+abils.Dexterity.Total*dexterityMultiplier, 20)

	return calculatedAP, nil
}

// CalculateSpeed はエンティティのSpeedを計算する
// 能力値ボーナス・状態異常ペナルティ・過積載ペナルティ・Effect倍率を考慮する
func CalculateSpeed(world w.World, entity ecs.Entity) int {
	speed := speedBaseValue

	// 能力値ボーナス
	if abils := world.Components.Abilities.Get(entity); abils != nil {
		speed += abils.Agility.Total*speedAgilityMultiply + abils.Dexterity.Total*speedDexterityMultiply
	}

	// 過積載は加算ペナルティ
	speed += calculateOverweightPenalty(world, entity)

	// 疲労・空腹は行動速度倍率として乗算で効く。moving/moveCost と同じ乗算系。
	// Effects タブの表示と同じ ModActionSpeed の導出を経由する
	speed = ModifierValue(world, entity, gc.ModActionSpeed).ApplyInt(speed)

	// MoveCost倍率を適用する。全エンティティへ毎ターン走る最頻経路なので、
	// 内訳を作らない単キー導出で読む。
	// 100% = 変化なし、90% = 速い（走破スキル）、130% = 遅い（低体温）
	// MoveCost はコスト倍率なので速度へは逆適用する（高いほど遅い）。ApplyInt は使わない
	moveCost := max(int(ModifierValue(world, entity, gc.ModMoveCost)), 10)
	speed = speed * 100 / moveCost
	// 身体機能の歩行を掛ける。脚・足の怪我や意識低下で歩行が落ちると遅くなる。
	// Effects タブの表示と同じ導出を経由する
	moving := gc.HealthyCapacities().Moving
	if world.Components.HealthStatus.Has(entity) {
		moving = world.Components.HealthStatus.Get(entity).Capacities().Moving
	}
	speed = max(
		// 最小値制限
		moving.ApplyInt(speed), speedMinimum)

	return speed
}

// actionSpeedSources は ModActionSpeed への状態由来の寄与を内訳として返す。空腹・疲労が
// 段階ごとの加算%として効く。適用と Effects タブの内訳がこの1箇所を読むので、値と内訳がずれない。
// 体温は CharModifiers.MoveCost 経由で別に効くのでここには含めない。
func actionSpeedSources(world w.World, entity ecs.Entity) []gc.ModifierSource {
	var srcs []gc.ModifierSource

	if world.Components.Hunger.Has(entity) {
		level := world.Components.Hunger.Get(entity).GetLevel()
		if v := HungerSpeedPenalty(level); v != 0 {
			srcs = append(srcs, gc.ModifierSource{Kind: gc.SourceHunger, Hunger: level, Value: v})
		}
	}
	if world.Components.Fatigue.Has(entity) {
		f := world.Components.Fatigue.Get(entity)
		if v := f.Penalty().SpeedAdd; v != 0 {
			srcs = append(srcs, gc.ModifierSource{Kind: gc.SourceFatigue, Fatigue: f.GetLevel(), Value: v})
		}
	}
	return srcs
}

// HungerSpeedPenalty は空腹段階が行動速度倍率へ与える加算%を返す。命中・回復と同じく段階基準で、
// 飢えるほど負に大きい。基準は100で、適用と Effects タブの内訳表示が同じこの導出を読む
func HungerSpeedPenalty(level gc.HungerLevel) int {
	switch level {
	case gc.HungerSatiated, gc.HungerNormal:
		return 0
	case gc.HungerHungry:
		return -10
	case gc.HungerStarving:
		return -20
	}
	return 0
}

// calculateOverweightPenalty は過積載によるSpeedペナルティを計算する
func calculateOverweightPenalty(world w.World, entity ecs.Entity) int {
	cw := world.Components.WeightCapacity.Get(entity)
	if cw == nil {
		return 0
	}
	if cw.Max == 0 {
		return 0
	}

	if cw.Current > cw.Max {
		overweight := cw.Current - cw.Max
		penalty := min(int((overweight*25)/cw.Max), 75)
		return -penalty
	}

	return 0
}
