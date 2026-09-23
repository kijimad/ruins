package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// DriveCubeTiles は運転できるキューブのタイル座標を集めて返す。マクロ地図のマーカー用。
// 位置は GridElement から常に導けるので保持せず、読み取り時に集める。
func DriveCubeTiles(world w.World) []consts.Coord[consts.Tile] {
	var out []consts.Coord[consts.Tile]
	q := ActiveFilter2[gc.GridElement, gc.Drivable](world).Query()
	for q.Next() {
		out = append(out, world.Components.GridElement.Get(q.Entity()).Coord)
	}
	return out
}

// DiscoveredChunks は探索済みタイルを含むチャンクの集合を絶対チャンク座標で返す。タイル単位の
// フォグ ExploredTiles をチャンク粒度へ畳み込み、マクロ地図のフォグにする。StageField が無ければ nil。
// 全画面の地形俯瞰図と HUD の右上地図が同じフォグを共有する。
func DiscoveredChunks(world w.World, sb *gc.SeamlessBand) map[consts.Coord[consts.Chunk]]bool {
	field := GetCurrentStageField(world)
	if field == nil {
		return nil
	}
	out := make(map[consts.Coord[consts.Chunk]]bool)
	// ChunkW は帯が有効なら正なのでゼロ除算しない。X は有界なので絶対チャンク列はタイルを幅で割るだけ。
	// Y は AbsChunkRow が帯ローカルの非負タイル行を絶対チャンク行へ移す。単一出どころを共有する
	for tile := range field.ExploredTiles {
		col := consts.Chunk(int(tile.X) / int(sb.ChunkW))
		out[consts.Coord[consts.Chunk]{X: col, Y: sb.AbsChunkRow(tile.Y)}] = true
	}
	return out
}

// PlayerBandTile はプレイヤーの帯ローカルなタイル座標を返す。プレイヤーが居なければ ok=false。
// マクロ地図のプレイヤーマーカーを置くのに使う。
func PlayerBandTile(world w.World) (consts.Coord[consts.Tile], bool) {
	player, err := GetPlayerEntity(world)
	if err != nil || !world.Components.GridElement.Has(player) {
		return consts.Coord[consts.Tile]{}, false
	}
	return world.Components.GridElement.Get(player).Coord, true
}

// StowedCargo はキューブに畳み込んだ貨物の一覧を返す。LocationStowed で燃料タンクと別管理する。
// 反復中に return するとロックが残るので、対象を集めてから返す。
func StowedCargo(world w.World, cube ecs.Entity) []ecs.Entity {
	var items []ecs.Entity
	q := ecs.NewFilter1[gc.LocationStowed](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.LocationStowed.Get(e).Owner == cube {
			items = append(items, e)
		}
	}
	return items
}

// CubeWeight はキューブが積む物の総重量を返す。運転1タイルの燃料コスト算出に使う。
// 燃料タンクの中身と畳み込んだ貨物の両方が機動に効くので合算する。値は保持せず読み取り時に導く。
func CubeWeight(world w.World, cube ecs.Entity) consts.Milligram {
	var total consts.Milligram
	for _, item := range GetStorageItems(world, cube) {
		total += GetEntityWeight(world, item)
	}
	for _, item := range StowedCargo(world, cube) {
		total += GetEntityWeight(world, item)
	}
	return total
}

// DriveFuelCost は総重量から1タイル運転するのに要する燃料量を返す。空でも基準量がかかり、
// 総重量に比例して増える。重いほど燃費が悪化する。
func DriveFuelCost(total consts.Milligram) consts.Heat {
	kg := int(total / consts.MilligramPerKg)
	return consts.Heat(consts.DriveFuelBase + consts.DriveFuelPerKg*kg)
}

// CubeFuelTotal はキューブ収納にある物の燃焼熱量の総和を返す。運転の燃料源。
// 火への給油と同じ HeatContent を使い燃料値の定義を二重化しない。
func CubeFuelTotal(world w.World, cube ecs.Entity) consts.Heat {
	var total consts.Heat
	// 燃料タンクは LocationInStorage。畳み込んだ貨物は LocationStowed で別管理なので混ざらない
	for _, item := range GetStorageItems(world, cube) {
		total += HeatContent(world, item)
	}
	return total
}

// FuelGaugeRatio は燃料ゲージ充填率を返す。表示基準 FuelGaugeFullHeat に対する比を 0..1 に丸める。
func FuelGaugeRatio(fuel consts.Heat) float64 {
	// 満量基準は正の定数なのでゼロ除算は起きない
	return min(1, max(0, float64(fuel)/float64(consts.FuelGaugeFullHeat)))
}

// PlayerDriving はプレイヤーが運転中なら Driving を返す。Vehicle の生存確認は呼び出し側の責務。
func PlayerDriving(world w.World) (*gc.Driving, bool) {
	player, err := GetPlayerEntity(world)
	if err != nil || !world.Components.Driving.Has(player) {
		return nil, false
	}
	return world.Components.Driving.Get(player), true
}
