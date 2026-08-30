package systems

import (
	"errors"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
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

// 温度閾値の定数
const (
	// ComfortableTempLower は快適温度の下限
	ComfortableTempLower = 11
	// ComfortableTempUpper は快適温度の上限
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

// dungeonWorldInfluenceDivisor はダンジョンの周囲気温が世界温度から受ける影響の減衰。
// 屋内なので屋外の世界温度をそのままでなく割って受け、揺れを和らげる。値は実プレイで調整する。
const dungeonWorldInfluenceDivisor = 2

// naturalRecoveryPerTurn は悪化方向でないときに体温状態タイマーが1ターンで下がる量
const naturalRecoveryPerTurn = 0.25

// AmbientTemperatureAt はタイルの周囲気温を返す。オーバーワールドは屋外なので季節の世界温度
// そのもの、ダンジョンは屋内なのでステージの基本気温に世界温度の影響を緩和して足した値。
// どちらもタイルの熱源補正を足す。屋内外はステージ種別で決まり、タイルごとの判定はしない。
func AmbientTemperatureAt(world w.World, x, y consts.Tile) (int, error) {
	dungeonRes := query.GetDungeon(world)
	if dungeonRes == nil {
		return 0, errors.New("dungeon resource is not set")
	}

	gt := query.GetGameTime(world)
	// 屋外の世界温度。季節ベースに時間帯の揺れを重ねる
	worldTemp := gt.GetSeasonalTemperature() + gt.GetTemperatureModifier()
	tileModifier := getTileTemperatureAt(world, x, y)

	// 屋外はステージの基本気温を使わないので、定義を引く前に返す
	if query.IsOnOverworld(world) {
		return worldTemp + tileModifier, nil
	}

	// 屋内はステージの基本気温が要る。定義が無ければ判定できないので0を返す
	def, ok := dungeon.GetStageDefinition(dungeonRes.CurrentStage.Name)
	if !ok {
		return 0, nil
	}

	// 世界温度の影響を緩和して受ける。世界が寒いほどダンジョンも寒くなるが、
	// 屋外ほど厳しくならず寒さの逆転も起きない
	return def.BaseTemperature() + worldTemp/dungeonWorldInfluenceDivisor + tileModifier, nil
}

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
	// hypothermiaSevereHPDamagePerTurn は重症の低体温が毎ターン与える HP ダメージ。値は実プレイで調整する
	hypothermiaSevereHPDamagePerTurn = 3
)

// Update は環境から体温を動かし、正常帯を外れた体温で健康状態のタイマーを進める
func (sys *TemperatureSystem) Update(world w.World) error {
	if query.GetDungeon(world) == nil {
		return errors.New("dungeon resource is not set")
	}

	// HealthStatusとGridElementを持つエンティティを処理。
	var toMark []ecs.Entity
	var toFreeze []ecs.Entity
	healthQuery := query.ActiveFilter2[gc.HealthStatus, gc.GridElement](world).Query()
	for healthQuery.Next() {
		entity := healthQuery.Entity()
		hs := world.Components.HealthStatus.Get(entity)

		// 体温を環境レートぶん動かす
		rate := bodyTempRate(world, entity)
		hs.BodyTempOffset = math.Max(bodyTempMin, math.Min(bodyTempMax, hs.BodyTempOffset+rate))

		isPlayer := world.Components.Player.Has(entity)

		// 低体温進行倍率を取得する。体温の物理には掛けず、タイマー進行にだけ掛ける
		coldProgressPct := consts.PercentBase
		if world.Components.CharModifiers.Has(entity) {
			coldProgressPct = world.Components.CharModifiers.Get(entity).ColdProgress
		}

		// 体温の帯判定から健康状態を更新
		hasChange := updateTemperatureConditions(world, hs, isPlayer, coldProgressPct)

		// 状態変化があれば属性を再計算
		if isPlayer && hasChange {
			toMark = append(toMark, entity)
		}

		// 重症の低体温は毎ターン HP を削る。反復中に Dead を付けないようループ後へ回す
		if world.Components.HP.Has(entity) && isSevereHypothermia(hs) {
			toFreeze = append(toFreeze, entity)
		}
	}

	for _, entity := range toMark {
		if !world.Components.StatsChanged.Has(entity) {
			world.Components.StatsChanged.Add(entity, &gc.StatsChanged{})
		}
	}

	for _, entity := range toFreeze {
		gameaction.ApplyConditionDamage(world, entity, hypothermiaSevereHPDamagePerTurn, gc.CauseFrozen)
	}

	return nil
}

// isSevereHypothermia は全身の低体温が重症かを返す
func isSevereHypothermia(hs *gc.HealthStatus) bool {
	cond := hs.Parts[gc.BodyPartWholeBody].GetCondition(gc.ConditionHypothermia)
	return cond != nil && cond.Severity == gc.SeveritySevere
}

// bodyTempRate は現在地の環境が1ターンに動かす体温の変化量を返す。温まる向きが正。
// Update の適用と HUD のトレンド矢印が同じこの関数を読む
func bodyTempRate(world w.World, entity ecs.Entity) float64 {
	if !world.Components.HealthStatus.Has(entity) || !world.Components.GridElement.Has(entity) {
		return 0
	}
	grid := world.Components.GridElement.Get(entity)
	ambientTemp, err := AmbientTemperatureAt(world, grid.X, grid.Y)
	if err != nil {
		return 0
	}
	insulation := CalculateEquippedInsulation(world, entity)
	offset := world.Components.HealthStatus.Get(entity).BodyTempOffset

	var rate float64
	if cold := calcBodyTempRate(ambientTemp + insulation.Cold); cold < 0 {
		rate += cold
	}
	// 熱源は冷えた体だけを温める。平熱以上では効かせない
	if offset < 0 {
		rate += heatSourceWarmthAt(world, grid.X, grid.Y)
	}
	// 外因が無ければ恒常性で平熱へ戻る
	if rate == 0 && offset < 0 {
		return math.Min(bodyTempHomeostasisPerTurn, -offset)
	}
	return rate
}

// CalculateEquippedInsulation はエンティティの装備から全身の断熱値を計算する。
// 各装備部位の断熱値を合算して返す。
func CalculateEquippedInsulation(world w.World, owner ecs.Entity) Insulation {
	var total Insulation

	equipQuery := ecs.NewFilter2[gc.LocationEquipped, gc.Wearable](world.ECS).Query()
	for equipQuery.Next() {
		item := equipQuery.Entity()
		equipped := world.Components.LocationEquipped.Get(item)
		if equipped.Owner != owner {
			continue
		}

		wearable := world.Components.Wearable.Get(item)
		total.Cold += wearable.InsulationCold
		total.Heat += wearable.InsulationHeat
	}

	return total
}

// getTileTemperatureAt は指定座標のタイル気温修正値を取得する
func getTileTemperatureAt(world w.World, x, y consts.Tile) int {
	var modifier int
	tileTempQuery := query.ActiveFilter2[gc.GridElement, gc.TileTemperature](world).Query()
	for tileTempQuery.Next() {
		entity := tileTempQuery.Entity()
		grid := world.Components.GridElement.Get(entity)
		if grid.X == x && grid.Y == y {
			tileTemp := world.Components.TileTemperature.Get(entity)
			modifier = tileTemp.Total()
		}
	}
	return modifier
}

// heatSourceWarmthAt はタイル座標に届く全熱源の暖かさ合計を返す。
// 各熱源はチェビシェフ距離に応じて線形に減衰し、半径外は効かない。複数の熱源は加算する。
// HeatSource を持つものを数える。暖房かどうかは HeatSource だけで決まり Burning とは独立で、
// 電熱のように燃えない熱源も暖房になる。火は燃え尽きると自分の HeatSource を外すので数から外れる
func heatSourceWarmthAt(world w.World, x, y consts.Tile) float64 {
	at := consts.Coord[consts.Tile]{X: x, Y: y}
	var warmth float64
	heatQuery := query.ActiveFilter2[gc.HeatSource, gc.GridElement](world).Query()
	for heatQuery.Next() {
		entity := heatQuery.Entity()
		src := world.Components.HeatSource.Get(entity)
		grid := world.Components.GridElement.Get(entity)
		if d := geometry.ChebyshevDistance(at, grid.Coord); d <= int(src.Radius) {
			reach := int(src.Radius) + 1
			warmth += src.Warmth * float64(reach-d) / float64(reach)
		}
	}
	return warmth
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

	// 効果を更新
	updateConditionEffects(partHealth)

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

// updateConditionEffects は全身の状態の効果を更新する
func updateConditionEffects(partHealth *gc.BodyPartHealth) {
	if cond := partHealth.GetCondition(gc.ConditionHypothermia); cond != nil {
		cond.Effects = calculateHypothermiaEffects(cond.Severity)
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
