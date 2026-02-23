package systems

import (
	"errors"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// TemperatureSystem は体温の更新を行うシステム
// 環境気温から健康状態のタイマーを更新する
type TemperatureSystem struct{}

// String はシステム名を返す
func (sys *TemperatureSystem) String() string {
	return "TemperatureSystem"
}

// 快適温度（摂氏）
const comfortableTemp = 20

// Update は健康状態のタイマーを更新する
func (sys *TemperatureSystem) Update(world w.World) error {
	dungeonRes := world.Resources.Dungeon
	if dungeonRes == nil {
		return errors.New("ダンジョンリソースが設定されていない")
	}

	// ダンジョン定義を取得
	def, ok := dungeon.GetDungeon(dungeonRes.DefinitionName)
	if !ok {
		return nil
	}

	// 基本環境気温
	baseTemp := def.BaseTemperature

	// 時間帯による修正
	timeModifier := 0
	if world.Resources.GameTime != nil {
		timeModifier = world.Resources.GameTime.GetTemperatureModifier()
	}

	// HealthStatusとGridElementを持つエンティティを処理
	world.Manager.Join(
		world.Components.HealthStatus,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		hs := world.Components.HealthStatus.Get(entity).(*gc.HealthStatus)
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// エンティティの位置のタイル気温修正を取得
		tileModifier := getTileTemperatureAt(world, gridElement.X, gridElement.Y)

		// 環境気温 = 基本気温 + タイル修正 + 時間帯修正
		envTemp := baseTemp + tileModifier + timeModifier

		// 装備から保温値を計算する
		warmth := calculateEquippedWarmth(world, entity)

		isPlayer := entity.HasComponent(world.Components.Player)

		// 各部位の健康状態を更新
		hasChange := updateTemperatureConditions(hs, envTemp, warmth, isPlayer)

		// プレイヤーで状態変化があれば属性を再計算
		if isPlayer && hasChange {
			entity.AddComponent(world.Components.EquipmentChanged, &gc.EquipmentChanged{})
		}
	}))

	return nil
}

// calculateEquippedWarmth はエンティティの装備から保温値を計算する
func calculateEquippedWarmth(world w.World, owner ecs.Entity) [gc.BodyPartCount]int {
	var warmth [gc.BodyPartCount]int

	world.Manager.Join(
		world.Components.ItemLocationEquipped,
		world.Components.Wearable,
	).Visit(ecs.Visit(func(item ecs.Entity) {
		equipped := world.Components.ItemLocationEquipped.Get(item).(*gc.LocationEquipped)
		if equipped.Owner != owner {
			return
		}

		wearable := world.Components.Wearable.Get(item).(*gc.Wearable)
		for _, part := range wearable.EquipmentCategory.CoveredBodyParts() {
			warmth[part] += wearable.Warmth
		}
	}))

	return warmth
}

// getTileTemperatureAt は指定座標のタイル気温修正値を取得する
func getTileTemperatureAt(world w.World, x, y gc.Tile) int {
	var modifier int
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.TileTemperature,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		if grid.X == x && grid.Y == y {
			tileTemp := world.Components.TileTemperature.Get(entity).(*gc.TileTemperature)
			modifier = tileTemp.Total()
		}
	}))
	return modifier
}

// updateTemperatureConditions は環境気温から各部位の健康状態タイマーを更新する
// isPlayerがtrueの場合、状態悪化時にログを出力する
// 戻り値: いずれかの状態のSeverityが変化した場合trueを返す
func updateTemperatureConditions(hs *gc.HealthStatus, envTemp int, warmth [gc.BodyPartCount]int, isPlayer bool) bool {
	hasChange := false

	for i := 0; i < int(gc.BodyPartCount); i++ {
		part := gc.BodyPart(i)
		partHealth := &hs.Parts[i]

		// 有効温度 = 環境気温 + 装備保温値
		effectiveTemp := envTemp + warmth[i]

		// 有効温度からタイマー変化量を計算
		delta := calcTimerDelta(effectiveTemp)

		// 低体温/高体温のタイマーを更新
		var changes []gc.SeverityChange
		if delta < 0 {
			// 寒い: 低体温タイマー増加、高体温タイマー減少
			changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHypothermia, -delta))
			changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHyperthermia, delta))
		} else if delta > 0 {
			// 暑い: 高体温タイマー増加、低体温タイマー減少
			changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHyperthermia, delta))
			changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHypothermia, -delta))
		} else {
			// 快適: 両方のタイマーが回復
			changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHypothermia, -0.25))
			changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHyperthermia, -0.25))
		}

		// 凍傷タイマー更新（末端部位のみ）
		if gc.IsExtremity(part) {
			changes = append(changes, updateFrostbiteTimer(partHealth, effectiveTemp))
		}

		// 状態変化をチェックし、プレイヤーならログ出力
		for _, change := range changes {
			if change.Prev != change.Current {
				hasChange = true
				if isPlayer {
					logTemperatureChange(part, change.CondType, change.Current, change.Prev)
				}
			}
		}

		// 効果を更新
		updateConditionEffects(partHealth, part)
	}

	return hasChange
}

// calcTimerDelta は有効温度からタイマー変化量を計算する
// 負の値は低体温方向、正の値は高体温方向
func calcTimerDelta(effectiveTemp int) float64 {
	switch {
	case effectiveTemp <= 0:
		return -0.5 // 非常に寒い
	case effectiveTemp <= 10:
		return -0.25 // 寒い
	case effectiveTemp <= 15:
		return 0 // やや寒い（現状維持）
	case effectiveTemp <= 25:
		return 0 // 快適
	case effectiveTemp <= 30:
		return 0 // やや暑い（現状維持）
	case effectiveTemp <= 35:
		return 0.25 // 暑い
	default:
		return 0.5 // 非常に暑い
	}
}

// updateFrostbiteTimer は凍傷タイマーを更新する
func updateFrostbiteTimer(partHealth *gc.BodyPartHealth, effectiveTemp int) gc.SeverityChange {
	var delta float64
	switch {
	case effectiveTemp <= 0:
		delta = 0.5 // 非常に危険
	case effectiveTemp <= 5:
		delta = 0.25 // 危険
	case effectiveTemp <= 10:
		delta = 0 // 現状維持
	default:
		delta = -0.25 // 回復
	}

	return partHealth.UpdateConditionTimer(gc.ConditionFrostbite, delta)
}

// updateConditionEffects は状態の効果を更新する
func updateConditionEffects(partHealth *gc.BodyPartHealth, part gc.BodyPart) {
	// 低体温の効果
	if cond := partHealth.GetCondition(gc.ConditionHypothermia); cond != nil {
		cond.Effects = calculateHypothermiaEffects(part, cond.Severity)
	}

	// 高体温の効果
	if cond := partHealth.GetCondition(gc.ConditionHyperthermia); cond != nil {
		cond.Effects = calculateHyperthermiaEffects(part, cond.Severity)
	}

	// 凍傷の効果
	if cond := partHealth.GetCondition(gc.ConditionFrostbite); cond != nil {
		cond.Effects = calculateFrostbiteEffects(part, cond.Severity)
	}
}

// getWorstSeverity は全部位で最も重い状態のSeverityを返す
func getWorstSeverity(hs *gc.HealthStatus) gc.Severity {
	worst := gc.SeverityNone
	for i := 0; i < int(gc.BodyPartCount); i++ {
		for _, cond := range hs.Parts[i].Conditions {
			if cond.Severity > worst {
				worst = cond.Severity
			}
		}
	}
	return worst
}

// logTemperatureChange は状態変化をログ出力する
func logTemperatureChange(part gc.BodyPart, condType gc.ConditionType, current, prev gc.Severity) {
	var msg string
	if current > prev {
		// 悪化
		msg = getWorseningMessage(part, condType, current)
	} else {
		// 回復
		msg = getRecoveryMessage(part, condType, current)
	}

	if msg != "" {
		gamelog.New(gamelog.FieldLog).
			Warning(msg).
			Log()
	}
}

// getWorseningMessage は悪化時のメッセージを返す
func getWorseningMessage(part gc.BodyPart, condType gc.ConditionType, severity gc.Severity) string {
	partTag := "[" + part.String() + "]"
	switch condType {
	case gc.ConditionHypothermia:
		switch severity {
		case gc.SeverityNone:
			return ""
		case gc.SeverityMinor:
			return "寒さで冷えてきた" + partTag
		case gc.SeverityMedium:
			return "寒さでかなり冷えている" + partTag
		case gc.SeveritySevere:
			return "寒さで危険な状態だ" + partTag
		}
	case gc.ConditionHyperthermia:
		switch severity {
		case gc.SeverityNone:
			return ""
		case gc.SeverityMinor:
			return "暑さで火照ってきた" + partTag
		case gc.SeverityMedium:
			return "暑さでかなり消耗している" + partTag
		case gc.SeveritySevere:
			return "暑さで危険な状態だ" + partTag
		}
	case gc.ConditionFrostbite:
		switch severity {
		case gc.SeverityNone:
			return ""
		case gc.SeverityMinor:
			return "凍傷になりかけている" + partTag
		case gc.SeverityMedium:
			return "凍傷が進行している" + partTag
		case gc.SeveritySevere:
			return "凍傷が危険な状態だ" + partTag
		}
	}
	return ""
}

// getRecoveryMessage は回復時のメッセージを返す
func getRecoveryMessage(part gc.BodyPart, condType gc.ConditionType, severity gc.Severity) string {
	partTag := "[" + part.String() + "]"
	switch condType {
	case gc.ConditionHypothermia:
		switch severity {
		case gc.SeverityNone:
			return "温まった" + partTag
		case gc.SeverityMinor:
			return "少し温まってきた" + partTag
		case gc.SeverityMedium:
			return "まだ寒いが、少しマシになった" + partTag
		case gc.SeveritySevere:
			return ""
		}
	case gc.ConditionHyperthermia:
		switch severity {
		case gc.SeverityNone:
			return "涼しくなった" + partTag
		case gc.SeverityMinor:
			return "少し涼しくなってきた" + partTag
		case gc.SeverityMedium:
			return "まだ暑いが、少しマシになった" + partTag
		case gc.SeveritySevere:
			return ""
		}
	case gc.ConditionFrostbite:
		switch severity {
		case gc.SeverityNone:
			return "凍傷が治った" + partTag
		case gc.SeverityMinor:
			return "凍傷が少し回復した" + partTag
		case gc.SeverityMedium:
			return "凍傷がまだ残っているが、少しマシになった" + partTag
		case gc.SeveritySevere:
			return ""
		}
	}
	return ""
}

// calculateHypothermiaEffects は低体温による効果を計算する
func calculateHypothermiaEffects(part gc.BodyPart, severity gc.Severity) []gc.StatEffect {
	multiplier := severityToMultiplier(severity)
	if multiplier == 0 {
		return nil
	}

	var effects []gc.StatEffect

	switch part {
	case gc.BodyPartTorso:
		effects = append(effects, gc.StatEffect{Stat: gc.StatStrength, Value: -1 * multiplier})
		effects = append(effects, gc.StatEffect{Stat: gc.StatVitality, Value: -1 * multiplier})
	case gc.BodyPartHead:
		effects = append(effects, gc.StatEffect{Stat: gc.StatSensation, Value: -1 * multiplier})
	case gc.BodyPartArms:
		effects = append(effects, gc.StatEffect{Stat: gc.StatStrength, Value: -1 * multiplier})
	case gc.BodyPartHands:
		effects = append(effects, gc.StatEffect{Stat: gc.StatDexterity, Value: -1 * multiplier})
	case gc.BodyPartLegs:
		effects = append(effects, gc.StatEffect{Stat: gc.StatAgility, Value: -1 * multiplier})
	case gc.BodyPartFeet:
		effects = append(effects, gc.StatEffect{Stat: gc.StatAgility, Value: -1 * multiplier})
	case gc.BodyPartCount:
		// BodyPartCount は列挙の終端を示す定数なので何もしない
	}

	return effects
}

// calculateHyperthermiaEffects は高体温による効果を計算する
func calculateHyperthermiaEffects(part gc.BodyPart, severity gc.Severity) []gc.StatEffect {
	multiplier := severityToMultiplier(severity)
	if multiplier == 0 {
		return nil
	}

	var effects []gc.StatEffect

	switch part {
	case gc.BodyPartTorso:
		effects = append(effects, gc.StatEffect{Stat: gc.StatStrength, Value: -1 * multiplier})
	case gc.BodyPartHead:
		effects = append(effects, gc.StatEffect{Stat: gc.StatSensation, Value: -1 * multiplier})
	case gc.BodyPartArms, gc.BodyPartHands, gc.BodyPartLegs, gc.BodyPartFeet, gc.BodyPartCount:
		// これらの部位は高体温の影響を受けない
	}

	return effects
}

// calculateFrostbiteEffects は凍傷による効果を計算する
func calculateFrostbiteEffects(part gc.BodyPart, severity gc.Severity) []gc.StatEffect {
	multiplier := severityToMultiplier(severity)
	if multiplier == 0 {
		return nil
	}

	var effects []gc.StatEffect

	switch part {
	case gc.BodyPartHands:
		effects = append(effects, gc.StatEffect{Stat: gc.StatDexterity, Value: -1 * multiplier})
	case gc.BodyPartFeet:
		effects = append(effects, gc.StatEffect{Stat: gc.StatAgility, Value: -1 * multiplier})
	case gc.BodyPartTorso, gc.BodyPartHead, gc.BodyPartArms, gc.BodyPartLegs, gc.BodyPartCount:
		// 凍傷は手と足のみに発生する
	}

	return effects
}

// severityToMultiplier はSeverityから効果倍率を返す
func severityToMultiplier(severity gc.Severity) int {
	switch severity {
	case gc.SeveritySevere:
		return 3
	case gc.SeverityMedium:
		return 2
	case gc.SeverityMinor:
		return 1
	default:
		return 0
	}
}
