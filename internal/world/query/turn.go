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
	log.Debug("アクションポイント消費",
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

		log.Debug("アクションポイント回復",
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
		return 0, fmt.Errorf("能力値が設定されていない")
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

	// 状態異常ペナルティ（空腹・過積載）
	speed += calculateStatusSpeedPenalty(world, entity)
	speed += calculateOverweightPenalty(world, entity)

	// MoveCost倍率を適用する。
	// 100% = 変化なし、90% = 速い（走破スキル）、130% = 遅い（低体温）
	if world.Components.CharModifiers.Has(entity) {
		effects := world.Components.CharModifiers.Get(entity)
		// MoveCost はコスト倍率なので速度へは逆適用する（高いほど遅い）。ApplyInt は使わない
		moveCost := max(int(effects.MoveCost), 10)
		speed = speed * 100 / moveCost
	}

	// 最小値制限
	if speed < speedMinimum {
		speed = speedMinimum
	}

	return speed
}

// calculateStatusSpeedPenalty は状態異常によるSpeedペナルティを計算する。
// 体温ペナルティはCharModifiers.MoveCost経由で適用されるためここには含まない。
func calculateStatusSpeedPenalty(world w.World, entity ecs.Entity) int {
	penalty := 0

	// 空腹ペナルティ
	if hunger := world.Components.Hunger.Get(entity); hunger != nil {
		penalty += hungerSpeedPenalty(hunger.Current)
	}

	return penalty
}

// hungerSpeedPenalty は空腹度によるペナルティを返す
func hungerSpeedPenalty(current int) int {
	switch {
	case current >= 75:
		return 0 // 満腹
	case current >= 50:
		return -10 // やや空腹
	case current >= 25:
		return -25 // 空腹
	case current >= 10:
		return -50 // 飢餓
	default:
		return -75 // 餓死寸前
	}
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
