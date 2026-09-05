package query

import (
	"errors"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 温度閾値の定数
const (
	// ComfortableTempLower は快適温度の下限
	ComfortableTempLower = 11
	// ComfortableTempUpper は快適温度の上限
	ComfortableTempUpper = 30
)

// Insulation は部位ごとの断熱値
type Insulation struct {
	Cold int // 耐寒。快適温度の下限を下げる
	Heat int // 耐暑。快適温度の上限を上げる
}

// comfortableRange は断熱値から快適温度範囲を計算する
func comfortableRange(insulation Insulation) (lower, upper int) {
	return ComfortableTempLower - insulation.Cold, ComfortableTempUpper + insulation.Heat
}

// sleepTemperatureMargin は入眠可能な温度を快適帯からどれだけ広げるか。
// 快適でなくとも多少の寒暖なら眠れる。値は実プレイで調整する
const sleepTemperatureMargin = 5

// SleepableTemperatureRange は装備断熱込みの快適帯を margin ぶん広げた入眠可能な温度帯を返す。
// この帯を外れると寒すぎ暑すぎで眠れず、火を焚くか屋内に入る必要がある
func SleepableTemperatureRange(world w.World, entity ecs.Entity) (lower, upper int) {
	cl, cu := comfortableRange(CalculateEquippedInsulation(world, entity))
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

// stageBaseTemperature はステージの基本気温を返す。値はステージ生成時に StageField.BaseTemp へ
// 確定済みで、ここは読むだけ。StageField 未確定なら基本気温0として扱う。
// dungeon 登録表を引かないことで query から dungeon への依存を断つ
func stageBaseTemperature(world w.World) int {
	if field := GetCurrentStageField(world); field != nil {
		return field.BaseTemp
	}
	return 0
}

// AmbientTemperatureAt はタイルの周囲気温を返す。ステージの基本気温、囲われに応じて
// 受け方を変えた世界温度、タイルの加算℃、熱源の押し上げの4項の和になる。
func AmbientTemperatureAt(world w.World, x, y consts.Tile) (int, error) {
	if GetDungeon(world) == nil {
		return 0, errors.New("dungeon resource is not set")
	}
	baseTemp := stageBaseTemperature(world)

	gt := GetGameTime(world)
	// 屋外の世界温度。季節ベースに時間帯の揺れを重ねる
	worldTemp := gt.GetSeasonalTemperature() + gt.GetTemperatureModifier()
	shelter, tileModifier := TileEnvironmentAt(world, x, y)

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
	return int(math.Round(HeatSourceWarmthAt(world, x, y) * ambientHeatPerWarmth))
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

// TileEnvironmentAt は指定座標のタイル環境を返す。囲われの緩和度と、水・植生の加算℃
func TileEnvironmentAt(world w.World, x, y consts.Tile) (gc.ShelterType, int) {
	shelter := gc.ShelterNone
	var modifier int
	tileTempQuery := ActiveFilter2[gc.GridElement, gc.TileEnvironment](world).Query()
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

// HeatSourceWarmthAt はタイル座標に届く全熱源の暖かさ合計を返す。
// 各熱源はチェビシェフ距離に応じて線形に減衰し、半径外は効かない。複数の熱源は加算する。
// HeatSource を持つものを数える。暖房かどうかは HeatSource だけで決まり Burning とは独立で、
// 電熱のように燃えない熱源も暖房になる。火は燃え尽きると自分の HeatSource を外すので数から外れる
func HeatSourceWarmthAt(world w.World, x, y consts.Tile) float64 {
	at := consts.Coord[consts.Tile]{X: x, Y: y}
	var warmth float64
	heatQuery := ActiveFilter2[gc.HeatSource, gc.GridElement](world).Query()
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
