package systems

import (
	"errors"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// TemperatureSystem は体温の更新を行うシステム
// 環境気温から健康状態のタイマーを更新する
type TemperatureSystem struct{}

// String はシステム名を返す
func (sys *TemperatureSystem) String() string {
	return "TemperatureSystem"
}

// naturalRecoveryPerTurn は悪化方向でないときに体温状態タイマーが1ターンで下がる量
const naturalRecoveryPerTurn = 0.25

// 体温の定数。値は実プレイで調整する
const (
	// bodyTempColdBand は平熱からこれだけ下がるまでを正常とみなす帯。これより冷えると低体温タイマーが進む
	bodyTempColdBand = 2.0
	// bodyTempMin はオフセットの下限クランプ
	bodyTempMin = -5.0
	// bodyTempMax はオフセットの上限クランプ。平熱が上限
	bodyTempMax = 0.0
	// bodyTempHomeostasisPerTurn は外因が無いときに平熱へ戻る1ターンの量
	bodyTempHomeostasisPerTurn = 0.1
)

// Update は環境から体温を動かし、正常帯を外れた体温で健康状態のタイマーを進める
func (sys *TemperatureSystem) Update(world w.World) error {
	if query.GetDungeon(world) == nil {
		return errors.New("dungeon resource is not set")
	}

	// HealthStatusとGridElementを持つエンティティを処理。
	var toMark []ecs.Entity
	healthQuery := query.ActiveFilter2[gc.HealthStatus, gc.GridElement](world).Query()
	for healthQuery.Next() {
		entity := healthQuery.Entity()
		hs := world.Components.HealthStatus.Get(entity)

		// 体温を環境レートぶん動かす
		rate := bodyTempRate(world, entity)
		hs.BodyTempOffset = math.Max(bodyTempMin, math.Min(bodyTempMax, hs.BodyTempOffset+rate))

		isPlayer := world.Components.Player.Has(entity)

		// 低体温進行倍率を取得する。体温の物理には掛けず、タイマー進行にだけ掛ける
		coldProgressPct := query.ModifierValue(world, entity, gc.ModColdProgress)

		// 体温の帯判定から健康状態を更新
		hasChange := updateTemperatureConditions(world, hs, isPlayer, coldProgressPct)

		// 状態変化があれば属性を再計算
		if isPlayer && hasChange {
			toMark = append(toMark, entity)
		}
	}

	for _, entity := range toMark {
		if !world.Components.StatsChanged.Has(entity) {
			world.Components.StatsChanged.Add(entity, &gc.StatsChanged{})
		}
	}

	return nil
}

// bodyTempRate は現在地の環境が1ターンに動かす体温の変化量を返す。温まる向きが正。
// Update の適用と HUD のトレンド矢印が同じこの関数を読む
func bodyTempRate(world w.World, entity ecs.Entity) float64 {
	if !world.Components.HealthStatus.Has(entity) || !world.Components.GridElement.Has(entity) {
		return 0
	}
	grid := world.Components.GridElement.Get(entity)
	ambientTemp, err := query.AmbientTemperatureAt(world, grid.X, grid.Y)
	if err != nil {
		return 0
	}
	insulation := query.CalculateEquippedInsulation(world, entity)
	offset := world.Components.HealthStatus.Get(entity).BodyTempOffset

	var rate float64
	if cold := calcBodyTempRate(ambientTemp + insulation.Cold); cold < 0 {
		rate += cold
	}
	// 熱源は冷えた体だけを温める。平熱以上では効かせない
	if offset < 0 {
		rate += query.HeatSourceWarmthAt(world, grid.X, grid.Y)
	}
	// 外因が無ければ恒常性で平熱へ戻る
	if rate == 0 && offset < 0 {
		return math.Min(bodyTempHomeostasisPerTurn, -offset)
	}
	return rate
}

// updateTemperatureConditions は体温の正常帯判定から全身の体温状態タイマーを更新する。
// - 帯の外では超過に応じてタイマーが進み、帯の中では自然回復する。
// - isPlayerがtrueの場合、状態変化時にログを出力する。
// - coldProgressPctは低体温進行倍率%。100が基準で、低いほど進行が遅くなる。
// - 戻り値: 状態のSeverityが変化した場合trueを返す
func updateTemperatureConditions(world w.World, hs *gc.HealthStatus, isPlayer bool, coldProgressPct consts.Percent) bool {
	hasChange := false
	partHealth := &hs.Parts[gc.BodyPartWholeBody]
	offset := hs.BodyTempOffset

	var changes []gc.SeverityChange

	// 低体温の処理
	switch {
	case offset < -bodyTempColdBand:
		delta := coldProgressPct.ApplyFloat(timerProgress(-offset - bodyTempColdBand))
		changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHypothermia, delta))
	case partHealth.GetCondition(gc.ConditionHypothermia) != nil:
		changes = append(changes, partHealth.UpdateConditionTimer(gc.ConditionHypothermia, -naturalRecoveryPerTurn))
	}

	for _, change := range changes {
		if change.Prev != change.Current {
			hasChange = true
			if isPlayer {
				logTemperatureChange(world, change.CondType, change.Current, change.Prev)
			}
		}
	}

	return hasChange
}

// timerProgress は正常帯からの超過 excess °C に応じたタイマー進行量を返す。
// 帯の縁で 0.25、超過1°Cごとに +0.25、上限 1.0
func timerProgress(excess float64) float64 {
	return math.Min(naturalRecoveryPerTurn+0.25*excess, 1.0)
}

// calcBodyTempRate は有効温度から体温の変化量を計算する。寒いほど負へ大きく、適温以上は0で正の値は返さない
func calcBodyTempRate(effectiveTemp int) float64 {
	switch {
	case effectiveTemp <= -50:
		return -0.5 // 極寒。最も厳しい区分で、居座れば急速に凍える
	case effectiveTemp <= 0:
		return -0.2 // 非常に寒い
	case effectiveTemp <= 10:
		return -0.1 // 寒い
	default:
		return 0 // 適温以上
	}
}

// logTemperatureChange は状態変化をログ出力する
func logTemperatureChange(world w.World, condType gc.ConditionType, current, prev gc.Severity) {
	var msg string
	if current > prev {
		msg = getWorseningMessage(condType, current)
	} else {
		msg = getRecoveryMessage(condType, current)
	}

	if msg != "" {
		gamelog.New(query.GetGameLog(world)).
			Markup(gamelog.Tag("warning", query.T(world, msg))).
			Log()
	}
}

// getWorseningMessage は悪化時のメッセージを返す
func getWorseningMessage(condType gc.ConditionType, severity gc.Severity) string {
	if condType == gc.ConditionHypothermia {
		switch severity {
		case gc.SeverityNone:
			return ""
		case gc.SeverityMinor:
			return "The cold is setting in"
		case gc.SeverityMedium:
			return "You are quite cold"
		case gc.SeveritySevere:
			return "The cold is dangerous"
		}
	}
	return ""
}

// getRecoveryMessage は回復時のメッセージを返す
func getRecoveryMessage(condType gc.ConditionType, severity gc.Severity) string {
	if condType == gc.ConditionHypothermia {
		switch severity {
		case gc.SeverityNone:
			return "You have warmed up"
		case gc.SeverityMinor:
			return "You are warming up a little"
		case gc.SeverityMedium:
			return "Still cold, but a little better"
		case gc.SeveritySevere:
			return ""
		}
	}
	return ""
}
