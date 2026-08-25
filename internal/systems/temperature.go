package systems

import (
	"errors"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/geometry"
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

// dungeonWorldInfluenceDivisor はダンジョンの周囲気温が世界温度から受ける影響の減衰。
// 屋内なので屋外の世界温度をそのままでなく割って受け、揺れを和らげる。値は実プレイで調整する。
const dungeonWorldInfluenceDivisor = 2

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

// Update は健康状態のタイマーを更新する
func (sys *TemperatureSystem) Update(world w.World) error {
	if query.GetDungeon(world) == nil {
		return errors.New("dungeon resource is not set")
	}

	// HealthStatusとGridElementを持つエンティティを処理。
	var toMark []ecs.Entity
	// プレイヤーの体温トレンドはループ後に書き込む。クエリ反復中の構造変更を避ける
	var trendPlayer ecs.Entity
	var trendDelta float64
	haveTrend := false
	healthQuery := query.ActiveFilter2[gc.HealthStatus, gc.GridElement](world).Query()
	for healthQuery.Next() {
		entity := healthQuery.Entity()
		hs := world.Components.HealthStatus.Get(entity)
		gridElement := world.Components.GridElement.Get(entity)

		// 周囲気温を計算
		ambientTemp, err := AmbientTemperatureAt(world, gridElement.X, gridElement.Y)
		if err != nil {
			continue
		}

		// 装備から断熱値を計算する
		insulation := CalculateEquippedInsulation(world, entity)

		isPlayer := world.Components.Player.Has(entity)

		// 体温進行倍率を取得する
		coldProgressPct, heatProgressPct := consts.PercentBase, consts.PercentBase
		if world.Components.CharModifiers.Has(entity) {
			mods := world.Components.CharModifiers.Get(entity)
			coldProgressPct = mods.ColdProgress
			heatProgressPct = mods.HeatProgress
		}

		// プレイヤーの体温トレンドは低体温・高体温タイマーの更新前後の差から取る
		partHealth := &hs.Parts[gc.BodyPartWholeBody]
		var coldBefore, heatBefore float64
		if isPlayer {
			coldBefore = conditionTimer(partHealth, gc.ConditionHypothermia)
			heatBefore = conditionTimer(partHealth, gc.ConditionHyperthermia)
		}

		// 各部位の健康状態を更新
		hasChange := updateTemperatureConditions(world, hs, ambientTemp, insulation, isPlayer, coldProgressPct, heatProgressPct)

		// 周囲の熱源で低体温を回復する。周囲気温とは別に効き、屋内外で回復量は同じ
		warmth := heatSourceWarmthAt(world, gridElement.X, gridElement.Y)
		if applyHeatSourceRecovery(world, hs, warmth, isPlayer) {
			hasChange = true
		}

		if isPlayer {
			// 温まる方向を正にする。高体温タイマーの増加は暑くなる、低体温タイマーの増加は寒くなる。
			// 熱源の回復も含めた最終的な変化を見るため、回復処理の後に差を取る
			coldAfter := conditionTimer(partHealth, gc.ConditionHypothermia)
			heatAfter := conditionTimer(partHealth, gc.ConditionHyperthermia)
			trendDelta = (heatAfter - heatBefore) - (coldAfter - coldBefore)
			trendPlayer = entity
			haveTrend = true

			// 状態変化があれば属性を再計算
			if hasChange {
				toMark = append(toMark, entity)
			}
		}
	}

	for _, entity := range toMark {
		if !world.Components.StatsChanged.Has(entity) {
			world.Components.StatsChanged.Add(entity, &gc.StatsChanged{})
		}
	}

	// クエリ反復の外でトレンドを書き込む。初回は付与、以降は値の更新になる
	if haveTrend {
		if err := gc.Upsert(world.ECS, world.Components.TemperatureTrend, trendPlayer, &gc.TemperatureTrend{Delta: trendDelta}); err != nil {
			return err
		}
	}

	return nil
}

// conditionTimer は全身部位の指定状態の Timer を返す。状態が無ければ0
func conditionTimer(part *gc.BodyPartHealth, condType gc.ConditionType) float64 {
	if c := part.GetCondition(condType); c != nil {
		return c.Timer
	}
	return 0
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

// heatSourceWarmthAt はタイル座標のチェビシェフ半径内にある全熱源の Warmth 合計を返す。
// 半径内なら固定量、半径外なら効かない離散。複数の熱源は加算する
func heatSourceWarmthAt(world w.World, x, y consts.Tile) float64 {
	at := consts.Coord[consts.Tile]{X: x, Y: y}
	var warmth float64
	heatQuery := query.ActiveFilter2[gc.HeatSource, gc.GridElement](world).Query()
	for heatQuery.Next() {
		entity := heatQuery.Entity()
		src := world.Components.HeatSource.Get(entity)
		grid := world.Components.GridElement.Get(entity)
		if geometry.ChebyshevDistance(at, grid.Coord) <= int(src.Radius) {
			warmth += src.Warmth
		}
	}
	return warmth
}

// applyHeatSourceRecovery は熱源の Warmth ぶん全身の低体温タイマーを下げる。
// 周囲気温由来の悪化とは別に効く。低体温が無ければ回復するものが無いので何もしない。
// Severity が変わればプレイヤーはログを出し、true を返す
func applyHeatSourceRecovery(world w.World, hs *gc.HealthStatus, warmth float64, isPlayer bool) bool {
	if warmth <= 0 {
		return false
	}
	partHealth := &hs.Parts[gc.BodyPartWholeBody]
	if partHealth.GetCondition(gc.ConditionHypothermia) == nil {
		return false
	}

	change := partHealth.UpdateConditionTimer(gc.ConditionHypothermia, -warmth)
	if change.Prev == change.Current {
		return false
	}

	// Severity が変わったときだけ効果を再計算する。効果は Severity にのみ依存するので、
	// タイマーが動いても Severity が変わらなければ再計算は要らない
	updateConditionEffects(partHealth)
	if isPlayer {
		logTemperatureChange(world, change.CondType, change.Current, change.Prev)
	}
	return true
}

// updateTemperatureConditions は環境気温から全身の体温状態タイマーを更新する。
// - 断熱値は装備全体の合算値を使う。
// - isPlayerがtrueの場合、状態変化時にログを出力する。
// - coldProgressPct/heatProgressPctは体温進行倍率%。100が基準で、低いほど進行が遅くなる。
// - 戻り値: 状態のSeverityが変化した場合trueを返す
func updateTemperatureConditions(world w.World, hs *gc.HealthStatus, ambientTemp int, insulation Insulation, isPlayer bool, coldProgressPct, heatProgressPct consts.Percent) bool {
	hasChange := false
	partHealth := &hs.Parts[gc.BodyPartWholeBody]

	// 耐寒を適用した有効温度（寒さ判定用）: 耐寒が高いほど暖かく感じる
	effectiveTempCold := ambientTemp + insulation.Cold
	// 耐暑を適用した有効温度（暑さ判定用）: 耐暑が高いほど涼しく感じる
	effectiveTempHeat := ambientTemp - insulation.Heat

	coldDelta := coldProgressPct.ApplyFloat(calcTimerDelta(effectiveTempCold))
	heatDelta := heatProgressPct.ApplyFloat(calcTimerDelta(effectiveTempHeat))

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
				logTemperatureChange(world, change.CondType, change.Current, change.Prev)
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
	case effectiveTemp <= -50:
		return -1.0 // 極寒。最も厳しい区分で、居座れば急速に凍える
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
	switch condType {
	case gc.ConditionHypothermia:
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
	case gc.ConditionHyperthermia:
		switch severity {
		case gc.SeverityNone:
			return ""
		case gc.SeverityMinor:
			return "The heat is flushing you"
		case gc.SeverityMedium:
			return "The heat is wearing you down"
		case gc.SeveritySevere:
			return "The heat is dangerous"
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
			return "You have warmed up"
		case gc.SeverityMinor:
			return "You are warming up a little"
		case gc.SeverityMedium:
			return "Still cold, but a little better"
		case gc.SeveritySevere:
			return ""
		}
	case gc.ConditionHyperthermia:
		switch severity {
		case gc.SeverityNone:
			return "You have cooled down"
		case gc.SeverityMinor:
			return "You are cooling down a little"
		case gc.SeverityMedium:
			return "Still hot, but a little better"
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
