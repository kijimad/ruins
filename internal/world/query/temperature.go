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
//
// 緯度勾配は引数の y、すなわちそのタイル固有の緯度で測る。プレイヤーの緯度ではない。
// 同じ帯でも北端のタイルは南端のタイルより寒い、という場所ごとの気温を返すのが意図。
// プレイヤーの体感気温はプレイヤーの居るタイルの (x, y) で呼ぶことで得る。
func AmbientTemperatureAt(world w.World, x, y consts.Tile) (int, error) {
	if GetDungeon(world) == nil {
		return 0, errors.New("dungeon resource is not set")
	}
	baseTemp := stageBaseTemperature(world)

	gt := GetGameTime(world)
	// 屋外の世界温度。季節ベースに時間帯の揺れを重ね、奥地ほど寒くなる緯度勾配を差し引く。
	// 勾配を世界温度に折り込むことで、屋内は shelteredWorldTemp で寒さが緩和され、
	// 深部の施設が暖を取れる避難所になる。末尾で引くと屋内外が同じだけ寒くなり避難所にならない
	worldTemp := gt.GetSeasonalTemperature() + gt.GetTemperatureModifier() - latitudeCold(world, y)
	shelter, tileModifier := TileEnvironmentAt(world, x, y)

	return baseTemp +
		shelteredWorldTemp(shelter, worldTemp) +
		tileModifier +
		ambientHeatAt(world, x, y), nil
}

// 奥地ほど寒い緯度勾配のパラメータ。無限軸を奥へ進んだチャンク距離が増えるほど世界温度を下げる。
// 北極点へ近づくほど寒くなる惑星像を、進行距離の単調減少で表す。値は実プレイで調整する。
const (
	// latitudeColdPerChunk は1チャンク奥へ進むごとに下がる℃
	latitudeColdPerChunk = 1
	// latitudeColdMax は緯度勾配で下げる℃の上限。これ以上奥へ進んでも寒くならない
	latitudeColdMax = 40
)

// NorthDepthChunks は帯ローカル座標 y が湧き位置から北へ何チャンク進んだかを返す。
// 起点は初期帯の中央行 rows/2。手前や帯を持たないステージでは0。
//
// 起点を絶対チャンク0でなく中央行に置くのは、開始地点を穏やかに保つため。緯度勾配の寒さと
// HUD の奥地表示がこの1関数を共有し、気温と UI で同じ距離を指す。絶対軸 Y で測るので帯シフトを
// またいでも連続で、シフトの瞬間に値が飛ばない。北は -Y なので絶対 Y が小さいほど北で、
// 奥行きは中央行の絶対チャンク Y から現在地の絶対チャンク Y を引いた差になる。
func NorthDepthChunks(world w.World, y consts.Tile) int {
	sb := GetSeamlessBand(world)
	// ChunkH<=0 はゼロ除算を、Rows<=0 は SpawnChunkY の起点が壊れるのを防ぐ。帯が正しく生成されていれば
	// どちらも正だが、未初期化の SeamlessBand を渡された退化ケースを起点計算より前に弾く
	if sb == nil || sb.ChunkH <= 0 || sb.Rows <= 0 {
		return 0
	}
	// 絶対 Y は北側で負になりうるので floorDivInt でチャンク境界を連続させる。プレイヤーがチャンク境界
	// ちょうどに居ても、MaybeShift が Player フェーズの安定点で中央行へ収束させるので飛びは起きない
	currentChunkY := floorDivInt(int(sb.LocalToAbsY(y)), int(sb.ChunkH))
	// 起点は湧き位置の中央行。SeamlessBand が「湧き位置はどの行か」の唯一の出どころ
	if depth := int(sb.SpawnChunkY()) - currentChunkY; depth > 0 {
		return depth
	}
	return 0
}

// floorDivInt は負の被除数でも床方向へ丸める整数除算。絶対 Y は北側で負になりうるため、
// Go の / のゼロ方向丸めを床方向へ補正してチャンク境界を連続させる。
//
// overworld.floorDiv が consts.Chunk 版の同じロジックを持つ。query と overworld は依存方向が別で
// 共通 leaf に出すと循環するため、int 版をここに置き重複を許容する。3つ目の利用者が現れたら
// consts か独立した数値 leaf への切り出しを検討する。
func floorDivInt(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// latitudeCold は帯ローカル座標 y に対応する緯度勾配の寒さ、すなわち世界温度から差し引く℃を返す。
// 北へ進んだチャンク距離が増えるほど大きくなる。惑星の基礎的な寒さはステージの基本気温が担い、
// 緯度勾配は「そこからさらに北ほど寒い」加算分だけを表す。
func latitudeCold(world w.World, y consts.Tile) int {
	return latitudeColdForDepth(NorthDepthChunks(world, y))
}

// latitudeColdForDepth は湧き位置から奥へ進んだチャンク距離から差し引く℃を返す純粋計算。
// 距離に比例して単調増加し、latitudeColdMax で頭打ちになる。起点手前や負の距離では0。
func latitudeColdForDepth(chunksDeep int) int {
	if chunksDeep <= 0 {
		return 0
	}
	if cold := chunksDeep * latitudeColdPerChunk; cold < latitudeColdMax {
		return cold
	}
	return latitudeColdMax
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
