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

// sleepTemperatureMargin は入眠可能な温度を快適帯からどれだけ広げるか。
// 快適でなくとも多少の寒暖なら眠れる。値は実プレイで調整する
const sleepTemperatureMargin = 5

// SleepableTemperatureRange は装備断熱込みの快適帯を margin ぶん広げた入眠可能な温度帯を返す。
// この帯を外れると寒すぎ暑すぎで眠れず、火を焚くか屋内に入る必要がある
func SleepableTemperatureRange(world w.World, entity ecs.Entity) (lower, upper int) {
	cl, cu := ComfortableRange(CalculateEquippedInsulation(world, entity))
	return cl - sleepTemperatureMargin, cu + sleepTemperatureMargin
}

// 屋内の温度緩和パラメータ。屋内は世界温度をそのまま受けず、アンカー温度へ引き寄せて受ける。
// 引き寄せ先を 0℃ でなくアンカーにするのは、寒さが浅いときにも屋内が屋外より穏やかで
// あり続けるため。0℃ へ割るだけだと温暖時に屋内が屋外より寒くなる逆転が起きる。
// 値は実プレイで調整する。
const (
	indoorAnchorTemp       = 10
	indoorInfluenceDivisor = 2
)

// shelteredWorldTemp は囲われに応じて受け方を変えた世界温度を返す
func shelteredWorldTemp(shelter gc.ShelterType, worldTemp int) int {
	switch shelter {
	case gc.ShelterFull:
		return indoorAnchorTemp + (worldTemp-indoorAnchorTemp)/indoorInfluenceDivisor
	case gc.ShelterPartial:
		// 屋内より弱くアンカーへ寄せる。係数は実プレイで調整する
		return indoorAnchorTemp + (worldTemp-indoorAnchorTemp)*3/4
	case gc.ShelterNone:
		// 末尾の屋外 return へ落とす。default を置くと exhaustive linter が新値の漏れを検知できなくなる
	}
	// 屋外は世界温度をそのまま受ける。save 由来の未知の値も屋外へ落とす
	return worldTemp
}

// stageBaseTemperature はステージの基本気温を返す。オーバーワールドは0。
// ダンジョンの定義が無ければ判定できないので ok=false を返す
func stageBaseTemperature(world w.World) (int, bool) {
	if query.IsOnOverworld(world) {
		return 0, true
	}
	def, ok := dungeon.GetStageDefinition(query.GetDungeon(world).CurrentStage.Name)
	if !ok {
		return 0, false
	}
	return def.BaseTemperature(), true
}

// naturalRecoveryPerTurn は悪化方向でないときに体温状態タイマーが1ターンで下がる量
const naturalRecoveryPerTurn = 0.25

// AmbientTemperatureAt はタイルの周囲気温を返す。ステージの基本気温、囲われに応じて
// 受け方を変えた世界温度、タイルの加算℃、熱源の押し上げの4項の和になる。
func AmbientTemperatureAt(world w.World, x, y consts.Tile) (int, error) {
	if query.GetDungeon(world) == nil {
		return 0, errors.New("dungeon resource is not set")
	}
	baseTemp, ok := stageBaseTemperature(world)
	if !ok {
		return 0, nil
	}

	gt := query.GetGameTime(world)
	// 屋外の世界温度。季節ベースに時間帯の揺れを重ねる
	worldTemp := gt.GetSeasonalTemperature() + gt.GetTemperatureModifier()
	shelter, tileModifier := tileEnvironmentAt(world, x, y)

	return baseTemp +
		shelteredWorldTemp(shelter, worldTemp) +
		tileModifier +
		ambientHeatAt(world, x, y), nil
}

// ambientHeatPerWarmth は熱源の暖かさ1あたり環境気温へ押し上げる℃。
// 体温タイマーへの直接回復とは別の効きで、火を焚けば周囲気温そのものが上がる。
// 状態は持たず、毎回そのターンの熱源から平衡値を出す。値は実プレイで調整する。
// 焚き火 warmth 0.75 の隣接タイルで +15℃ になり、春の夜が火のそばで快適帯に入る
const ambientHeatPerWarmth = 30

// ambientHeatAt はタイル座標に届く熱源の環境気温への押し上げ℃を返す
func ambientHeatAt(world w.World, x, y consts.Tile) int {
	return int(math.Round(heatSourceWarmthAt(world, x, y) * ambientHeatPerWarmth))
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

// tileEnvironmentAt は指定座標のタイル環境を返す。囲われの緩和度と、水・植生の加算℃
func tileEnvironmentAt(world w.World, x, y consts.Tile) (gc.ShelterType, int) {
	shelter := gc.ShelterNone
	var modifier int
	tileTempQuery := query.ActiveFilter2[gc.GridElement, gc.TileEnvironment](world).Query()
	for tileTempQuery.Next() {
		entity := tileTempQuery.Entity()
		grid := world.Components.GridElement.Get(entity)
		if grid.X == x && grid.Y == y {
			tileTemp := world.Components.TileEnvironment.Get(entity)
			shelter = tileTemp.Shelter
			modifier = tileTemp.Total()
			tileTempQuery.Close()
			break
		}
	}
	return shelter, modifier
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
