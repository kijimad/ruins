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
type TemperatureSystem struct {
	// ログ出力用の状態。プレイヤー専用
	// 前の状態から変わったかを判定するのに使う
	prevWorstLevel gc.TempLevel
	// 収束したか
	hasConverged bool
}

// String はシステム名を返す
func (sys *TemperatureSystem) String() string {
	return "TemperatureSystem"
}

// 収束レート
// 半減期約30ターンで収束する
const convergenceRate = 0.03

// 快適温度（摂氏）
const comfortableTemp = 20

// Update は体温を更新する
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

	// BodyTemperatureとGridElementを持つエンティティを処理
	world.Manager.Join(
		world.Components.BodyTemperature,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		bt := world.Components.BodyTemperature.Get(entity).(*gc.BodyTemperature)
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// エンティティの位置のタイル気温修正を取得
		tileModifier := getTileTemperatureAt(world, gridElement.X, gridElement.Y)

		// 環境気温 = 基本気温 + タイル修正 + 時間帯修正
		envTemp := baseTemp + tileModifier + timeModifier

		// 装備から保温値を計算する
		warmth := calculateEquippedWarmth(world, entity)
		updateBodyTemperature(bt, envTemp, warmth)

		// HealthStatus を更新
		if entity.HasComponent(world.Components.HealthStatus) {
			hs := world.Components.HealthStatus.Get(entity).(*gc.HealthStatus)
			updateHealthConditionsFromTemperature(bt, hs)
		}

		// プレイヤーの場合、温度レベルが変化したらログ出力と属性再計算する
		if entity.HasComponent(world.Components.Player) {
			currentLevel := bt.GetWorstConvergentLevel()
			if currentLevel != sys.prevWorstLevel {
				if msg := getTempLogMessage(currentLevel); msg != "" {
					gamelog.New(gamelog.FieldLog).
						Warning(msg).
						Log()
				}
				sys.prevWorstLevel = currentLevel
				sys.hasConverged = false

				// 温度レベル変化時に属性を再計算
				entity.AddComponent(world.Components.EquipmentChanged, &gc.EquipmentChanged{})
			}

			// 収束温度に達したらメッセージを出力
			if !sys.hasConverged && isFullyConverged(bt) {
				if msg := getConvergedLogMessage(currentLevel); msg != "" {
					gamelog.New(gamelog.FieldLog).
						Warning(msg).
						Log()
				}
				sys.hasConverged = true
			}
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

// updateBodyTemperature は体温を更新する
// warmth は部位ごとの装備保温値
func updateBodyTemperature(bt *gc.BodyTemperature, envTemp int, warmth [gc.BodyPartCount]int) {
	for i := 0; i < int(gc.BodyPartCount); i++ {
		part := gc.BodyPart(i)
		state := &bt.Parts[i]

		// 収束温度を計算
		// 収束温度 = 50 + (環境気温 - 快適温度) * 2 + 装備保温値
		convergent := gc.TempNormal + (envTemp-comfortableTemp)*2 + warmth[i]
		state.Convergent = gc.ClampTemp(convergent)

		// 現在温度を収束温度に近づける
		delta := state.Convergent - state.Temp
		change := int(float64(delta) * convergenceRate)
		if change == 0 && delta != 0 {
			// 最低でも1は変化させる
			if delta > 0 {
				change = 1
			} else {
				change = -1
			}
		}
		state.Temp = gc.ClampTemp(state.Temp + change)

		// 凍傷タイマーを更新（手・足のみ）
		if gc.IsExtremity(part) {
			updateFrostbiteTimer(bt, part)
		}
	}
}

// updateFrostbiteTimer は凍傷タイマーを更新する
func updateFrostbiteTimer(bt *gc.BodyTemperature, part gc.BodyPart) {
	level := bt.GetLevel(part)
	state := &bt.Parts[part]

	switch level {
	case gc.TempLevelFreezing:
		state.FrostbiteTimer += 5 // 非常に危険
	case gc.TempLevelVeryCold:
		state.FrostbiteTimer += 3 // 危険
	case gc.TempLevelCold:
		state.FrostbiteTimer++ // リスク
	default:
		state.FrostbiteTimer -= 2 // 回復
	}

	// 範囲を制限
	if state.FrostbiteTimer < 0 {
		state.FrostbiteTimer = 0
	}
	if state.FrostbiteTimer > 100 {
		state.FrostbiteTimer = 100
	}

	// 凍傷発症判定
	if state.FrostbiteTimer >= 100 {
		state.HasFrostbite = true
	}
}

// updateHealthConditionsFromTemperature は体温から健康状態を更新する
func updateHealthConditionsFromTemperature(bt *gc.BodyTemperature, hs *gc.HealthStatus) {
	for i := 0; i < int(gc.BodyPartCount); i++ {
		part := gc.BodyPart(i)
		level := bt.GetLevel(part)
		partHealth := &hs.Parts[i]

		// 低体温状態を更新
		updateHypothermiaCondition(partHealth, part, level)

		// 高体温状態を更新
		updateHyperthermiaCondition(partHealth, part, level)

		// 凍傷状態を更新（手・足のみ）
		if gc.IsExtremity(part) {
			updateFrostbiteCondition(partHealth, part, bt.Parts[i].HasFrostbite)
		}
	}
}

// updateHypothermiaCondition は低体温状態を更新する
func updateHypothermiaCondition(partHealth *gc.BodyPartHealth, part gc.BodyPart, level gc.TempLevel) {
	// 正常以上の温度なら低体温を解除
	if level >= gc.TempLevelNormal {
		partHealth.RemoveCondition(gc.ConditionHypothermia)
		return
	}

	// 重症度を決定
	var severity gc.Severity
	switch level {
	case gc.TempLevelFreezing:
		severity = gc.SeveritySevere
	case gc.TempLevelVeryCold:
		severity = gc.SeverityMedium
	case gc.TempLevelCold:
		severity = gc.SeverityMinor
	default:
		return
	}

	// 状態による効果を計算
	effects := calculateHypothermiaEffects(part, severity)

	partHealth.SetCondition(gc.HealthCondition{
		Type:     gc.ConditionHypothermia,
		Severity: severity,
		Effects:  effects,
	})
}

// updateHyperthermiaCondition は高体温状態を更新する
func updateHyperthermiaCondition(partHealth *gc.BodyPartHealth, part gc.BodyPart, level gc.TempLevel) {
	// 正常以下の温度なら高体温を解除
	if level <= gc.TempLevelNormal {
		partHealth.RemoveCondition(gc.ConditionHyperthermia)
		return
	}

	// 重症度を決定
	var severity gc.Severity
	switch level {
	case gc.TempLevelScorching:
		severity = gc.SeveritySevere
	case gc.TempLevelVeryHot:
		severity = gc.SeverityMedium
	case gc.TempLevelHot:
		severity = gc.SeverityMinor
	default:
		return
	}

	// 状態による効果を計算
	effects := calculateHyperthermiaEffects(part, severity)

	partHealth.SetCondition(gc.HealthCondition{
		Type:     gc.ConditionHyperthermia,
		Severity: severity,
		Effects:  effects,
	})
}

// updateFrostbiteCondition は凍傷状態を更新する
func updateFrostbiteCondition(partHealth *gc.BodyPartHealth, part gc.BodyPart, hasFrostbite bool) {
	if !hasFrostbite {
		partHealth.RemoveCondition(gc.ConditionFrostbite)
		return
	}

	effects := calculateFrostbiteEffects(part)

	partHealth.SetCondition(gc.HealthCondition{
		Type:     gc.ConditionFrostbite,
		Severity: gc.SeveritySevere,
		Effects:  effects,
	})
}

// calculateHypothermiaEffects は低体温による効果を計算する
func calculateHypothermiaEffects(part gc.BodyPart, severity gc.Severity) []gc.StatEffect {
	// 重症度による倍率
	multiplier := 1
	switch severity {
	case gc.SeveritySevere:
		multiplier = 3
	case gc.SeverityMedium:
		multiplier = 2
	case gc.SeverityMinor:
		multiplier = 1
	case gc.SeverityNone:
		return nil
	}

	var effects []gc.StatEffect

	// 部位ごとの効果
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
	multiplier := 1
	switch severity {
	case gc.SeveritySevere:
		multiplier = 3
	case gc.SeverityMedium:
		multiplier = 2
	case gc.SeverityMinor:
		multiplier = 1
	case gc.SeverityNone:
		return nil
	}

	var effects []gc.StatEffect

	// 高体温は主に胴体と頭に影響
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
func calculateFrostbiteEffects(part gc.BodyPart) []gc.StatEffect {
	var effects []gc.StatEffect

	switch part {
	case gc.BodyPartHands:
		effects = append(effects, gc.StatEffect{Stat: gc.StatDexterity, Value: -2})
	case gc.BodyPartFeet:
		effects = append(effects, gc.StatEffect{Stat: gc.StatAgility, Value: -2})
	case gc.BodyPartTorso, gc.BodyPartHead, gc.BodyPartArms, gc.BodyPartLegs, gc.BodyPartCount:
		// 凍傷は手と足のみに発生する
	}

	return effects
}

// getTempLogMessage は温度レベルに応じたログメッセージを返す
func getTempLogMessage(level gc.TempLevel) string {
	switch level {
	case gc.TempLevelFreezing:
		return "凍えそうだ"
	case gc.TempLevelVeryCold:
		return "とても寒い"
	case gc.TempLevelCold:
		return "寒い"
	case gc.TempLevelHot:
		return "暑い"
	case gc.TempLevelVeryHot:
		return "とても暑い"
	case gc.TempLevelScorching:
		return "焼けつくように暑い"
	case gc.TempLevelNormal:
		return ""
	default:
		panic("不正なTempLevel値")
	}
}

// isFullyConverged は全部位が収束温度に達しているか確認する
func isFullyConverged(bt *gc.BodyTemperature) bool {
	for i := 0; i < int(gc.BodyPartCount); i++ {
		if bt.Parts[i].Temp != bt.Parts[i].Convergent {
			return false
		}
	}
	return true
}

// getConvergedLogMessage は収束時のログメッセージを返す
func getConvergedLogMessage(level gc.TempLevel) string {
	switch level {
	case gc.TempLevelFreezing:
		return "完全に凍えている"
	case gc.TempLevelVeryCold:
		return "冷え切っている"
	case gc.TempLevelCold:
		return "体が冷えた"
	case gc.TempLevelHot:
		return "体が温まった"
	case gc.TempLevelVeryHot:
		return "体が火照っている"
	case gc.TempLevelScorching:
		return "体が焼けるように熱い"
	case gc.TempLevelNormal:
		return ""
	default:
		panic("不正なTempLevel値")
	}
}
