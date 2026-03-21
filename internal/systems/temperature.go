package systems

import (
	"errors"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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

// 温度閾値の定数
const (
	// ComfortableTempLower は快適温度の下限（これより低いと寒さダメージ）
	ComfortableTempLower = 11
	// ComfortableTempUpper は快適温度の上限（これより高いと暑さダメージ）
	ComfortableTempUpper = 30
)

// Insulation は部位ごとの断熱値
type Insulation struct {
	Cold int // 耐寒（快適温度の下限を下げる）
	Heat int // 耐暑（快適温度の上限を上げる）
}

// ComfortableRange は断熱値から快適温度範囲を計算する
func ComfortableRange(insulation Insulation) (lower, upper int) {
	return ComfortableTempLower - insulation.Cold, ComfortableTempUpper + insulation.Heat
}

// CalculateEnvTemperature は指定位置の環境気温を計算する
// 基本気温 + タイル修正 + 時間帯修正
func CalculateEnvTemperature(world w.World, x, y consts.Tile) (int, error) {
	dungeonRes := world.Resources.Dungeon
	if dungeonRes == nil {
		return 0, errors.New("ダンジョンリソースが設定されていない")
	}

	def, ok := dungeon.GetDungeon(dungeonRes.DefinitionName)
	if !ok {
		return 0, nil
	}

	baseTemp := def.BaseTemperature

	timeModifier := world.Resources.Dungeon.GameTime.GetTemperatureModifier()

	tileModifier := getTileTemperatureAt(world, x, y)

	return baseTemp + timeModifier + tileModifier, nil
}

// Update は健康状態のタイマーを更新する
func (sys *TemperatureSystem) Update(world w.World) error {
	if world.Resources.Dungeon == nil {
		return errors.New("ダンジョンリソースが設定されていない")
	}

	// HealthStatusとGridElementを持つエンティティを処理
	world.Manager.Join(
		world.Components.HealthStatus,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		hs := world.Components.HealthStatus.Get(entity).(*gc.HealthStatus)
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// 環境気温を計算
		envTemp, err := CalculateEnvTemperature(world, gridElement.X, gridElement.Y)
		if err != nil {
			return
		}

		// 装備から断熱値を計算する
		insulation := CalculateEquippedInsulation(world, entity)

		isPlayer := entity.HasComponent(world.Components.Player)

		// 各部位の健康状態を更新
		hasChange := updateTemperatureConditions(hs, envTemp, insulation, isPlayer)

		// プレイヤーで状態変化があれば属性を再計算
		if isPlayer && hasChange {
			entity.AddComponent(world.Components.EquipmentChanged, &gc.EquipmentChanged{})
		}
	}))

	return nil
}

// CalculateEquippedInsulation はエンティティの装備から全身の断熱値を計算する。
// 各装備部位の断熱値を合算して返す。
func CalculateEquippedInsulation(world w.World, owner ecs.Entity) Insulation {
	var total Insulation

	world.Manager.Join(
		world.Components.ItemLocationEquipped,
		world.Components.Wearable,
	).Visit(ecs.Visit(func(item ecs.Entity) {
		equipped := world.Components.ItemLocationEquipped.Get(item).(*gc.LocationEquipped)
		if equipped.Owner != owner {
			return
		}

		wearable := world.Components.Wearable.Get(item).(*gc.Wearable)
		total.Cold += wearable.InsulationCold
		total.Heat += wearable.InsulationHeat
	}))

	return total
}

// getTileTemperatureAt は指定座標のタイル気温修正値を取得する
func getTileTemperatureAt(world w.World, x, y consts.Tile) int {
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

// updateTemperatureConditions は環境気温から全身の体温状態タイマーを更新する。
// - 断熱値は装備全体の合算値を使う。
// - isPlayerがtrueの場合、状態変化時にログを出力する。
// - 戻り値: 状態のSeverityが変化した場合trueを返す
func updateTemperatureConditions(hs *gc.HealthStatus, envTemp int, insulation Insulation, isPlayer bool) bool {
	hasChange := false
	partHealth := &hs.Parts[gc.BodyPartWholeBody]

	// 耐寒を適用した有効温度（寒さ判定用）: 耐寒が高いほど暖かく感じる
	effectiveTempCold := envTemp + insulation.Cold
	// 耐暑を適用した有効温度（暑さ判定用）: 耐暑が高いほど涼しく感じる
	effectiveTempHeat := envTemp - insulation.Heat

	coldDelta := calcTimerDelta(effectiveTempCold)
	heatDelta := calcTimerDelta(effectiveTempHeat)

	var changes []gc.SeverityChange

	// 低体温の処理（寒さ判定）
	if coldDelta < 0 {
		changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHypothermia, -coldDelta))
	} else {
		changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHypothermia, -0.25))
	}

	// 高体温の処理（暑さ判定）
	if heatDelta > 0 {
		changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHyperthermia, heatDelta))
	} else {
		changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHyperthermia, -0.25))
	}

	for _, change := range changes {
		if change.Prev != change.Current {
			hasChange = true
			if isPlayer {
				logTemperatureChange(change.CondType, change.Current, change.Prev)
			}
		}
	}

	// 効果を更新
	updateConditionEffects(partHealth)

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

// updateConditionEffects は全身の状態の効果を更新する
func updateConditionEffects(partHealth *gc.BodyPartHealth) {
	if cond := partHealth.GetCondition(gc.ConditionHypothermia); cond != nil {
		cond.Effects = calculateHypothermiaEffects(cond.Severity)
	}
	if cond := partHealth.GetCondition(gc.ConditionHyperthermia); cond != nil {
		cond.Effects = calculateHyperthermiaEffects(cond.Severity)
	}
}

// logTemperatureChange は状態変化をログ出力する
func logTemperatureChange(condType gc.ConditionType, current, prev gc.Severity) {
	var msg string
	if current > prev {
		msg = getWorseningMessage(condType, current)
	} else {
		msg = getRecoveryMessage(condType, current)
	}

	if msg != "" {
		gamelog.New(gamelog.FieldLog).
			Warning(msg).
			Log()
	}
}

// getWorseningMessage は悪化時のメッセージを返す
func getWorseningMessage(condType gc.ConditionType, severity gc.Severity) string {
	switch condType {
	case gc.ConditionHypothermia:
		switch severity {
		case gc.SeverityNone:
			return ""
		case gc.SeverityMinor:
			return "寒さで冷えてきた"
		case gc.SeverityMedium:
			return "寒さでかなり冷えている"
		case gc.SeveritySevere:
			return "寒さで危険な状態だ"
		}
	case gc.ConditionHyperthermia:
		switch severity {
		case gc.SeverityNone:
			return ""
		case gc.SeverityMinor:
			return "暑さで火照ってきた"
		case gc.SeverityMedium:
			return "暑さでかなり消耗している"
		case gc.SeveritySevere:
			return "暑さで危険な状態だ"
		}
	}
	return ""
}

// getRecoveryMessage は回復時のメッセージを返す
func getRecoveryMessage(condType gc.ConditionType, severity gc.Severity) string {
	switch condType {
	case gc.ConditionHypothermia:
		switch severity {
		case gc.SeverityNone:
			return "温まった"
		case gc.SeverityMinor:
			return "少し温まってきた"
		case gc.SeverityMedium:
			return "まだ寒いが、少しマシになった"
		case gc.SeveritySevere:
			return ""
		}
	case gc.ConditionHyperthermia:
		switch severity {
		case gc.SeverityNone:
			return "涼しくなった"
		case gc.SeverityMinor:
			return "少し涼しくなってきた"
		case gc.SeverityMedium:
			return "まだ暑いが、少しマシになった"
		case gc.SeveritySevere:
			return ""
		}
	}
	return ""
}

// calculateHypothermiaEffects は低体温による全身への効果を計算する
func calculateHypothermiaEffects(severity gc.Severity) []gc.StatEffect {
	m := severityToMultiplier(severity)
	if m == 0 {
		return nil
	}

	return []gc.StatEffect{
		{Stat: gc.StatStrength, Value: -1 * m},
		{Stat: gc.StatVitality, Value: -1 * m},
		{Stat: gc.StatDexterity, Value: -1 * m},
		{Stat: gc.StatAgility, Value: -1 * m},
	}
}

// calculateHyperthermiaEffects は高体温による全身への効果を計算する
func calculateHyperthermiaEffects(severity gc.Severity) []gc.StatEffect {
	m := severityToMultiplier(severity)
	if m == 0 {
		return nil
	}

	return []gc.StatEffect{
		{Stat: gc.StatStrength, Value: -1 * m},
		{Stat: gc.StatSensation, Value: -1 * m},
		{Stat: gc.StatVitality, Value: -1 * m},
	}
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
